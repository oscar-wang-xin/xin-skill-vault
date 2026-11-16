package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/xin-skill-vault/xin-skill-vault/internal/service"
	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
	"github.com/xin-skill-vault/xin-skill-vault/internal/vault"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/trayicon.png
var trayIcon []byte

const appVersion = "0.1.0"

var (
	forceQuit          atomic.Bool
	closePromptPending atomic.Bool
)

func init() {
	application.RegisterEvent[bool]("close-requested")
}

type seedAgent struct {
	ID   string
	Name string
	Rel  string
}

var seedAgents = []seedAgent{
	{"opencode", "OpenCode", ".config/opencode/skills"},
	{"claude-code", "Claude Code", ".claude/skills"},
	{"codex", "Codex", ".codex/skills"},
	{"deepseek-harness", "DeepSeek Harness", ".dsh/skills"},
	{"agent-skills", "Agent Skills（通用）", ".agents/skills"},
	{"cursor", "Cursor", ".cursor/skills"},
	{"gemini", "Gemini CLI", ".gemini/skills"},
	{"pi", "Pi", ".pi/agent/skills"},
	{"workbuddy", "WorkBuddy", ".workbuddy/skills"},
	{"codebuddy", "CodeBuddy", ".codebuddy/skills"},
	{"qwen-code", "Qwen Code", ".qwen/skills"},
	{"windsurf", "Windsurf", ".windsurf/skills"},
	{"trae", "Trae", ".trae/skills"},
	{"cline", "Cline", ".cline/skills"},
	{"roo-code", "Roo Code", ".roo/skills"},
	{"openclaw", "OpenClaw", ".openclaw/skills"},
}

func main() {
	dataDir, dbPath, vaultPath := resolvePaths()
	logger := setupLogger(dataDir)
	defer logger.Close()

	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("Xin Skill Vault 启动失败：%v", err)
	}
	defer st.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("无法定位用户目录：%v", err)
	}
	if err := seedDefaults(st, home); err != nil {
		log.Fatalf("初始化默认数据失败：%v", err)
	}
	vaultPath = settingOr(st, service.SettingVaultPath, vaultPath)
	if custom := os.Getenv("XSV_VAULT_PATH"); custom != "" {
		vaultPath = custom
	}

	manager := vault.New(vaultPath, st)
	if err := manager.Ensure(); err != nil {
		log.Fatalf("初始化技能仓库失败：%v", err)
	}
	if _, err := manager.Scan(); err != nil {
		log.Printf("启动扫描失败（应用继续运行）：%v", err)
	}

	svc := &service.AppService{
		Store:   st,
		Vault:   manager,
		DataDir: dataDir,
		DBPath:  dbPath,
		Version: appVersion,
	}

	background := skinBackground(settingOr(st, service.SettingSkin, "cold-white"))
	startURL := "/"
	if query := os.Getenv("XSV_START_QUERY"); query != "" {
		startURL += "?" + query
	}
	scale := systemScale()
	winWidth, winHeight := 1360, 860
	if scale > 1.01 {
		winWidth = int(float64(winWidth) * scale)
		winHeight = int(float64(winHeight) * scale)
	}
	log.Printf("window physical size: %dx%d (system scale %.2f)", winWidth, winHeight, scale)
	app := application.New(application.Options{
		Name:        "Xin Skill Vault",
		Description: "Agent Skill 仓库管理器 · Xin技能管理",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: false,
			UseVisualHosting:              true,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			_ = st.Close()
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            "Xin Skill Vault · Xin技能管理",
		Width:            winWidth,
		Height:           winHeight,
		MinWidth:         900,
		MinHeight:        520,
		Frameless:        true,
		BackgroundColour: background,
		URL:              startURL,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 44,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})
	svc.Window = window
	svc.ForceQuit = &forceQuit
	svc.ClosePromptPending = &closePromptPending
	setupTray(app, window)
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if forceQuit.Load() {
			log.Printf("close: force quit already set")
			return
		}
		action := settingOr(st, service.SettingCloseAction, "ask")
		log.Printf("close: action=%s", action)
		switch action {
		case "quit":
			forceQuit.Store(true)
			return
		case "tray":
			e.Cancel()
			window.Hide()
			return
		default:
			e.Cancel()
			if closePromptPending.CompareAndSwap(false, true) {
				app.Event.Emit("close-requested", true)
			}
		}
	})
	// The screen manager populates monitor DPI only after the event loop starts.
	// Re-fit once to the primary screen's DIP work area so a 4K/high-DPI window
	// never overflows the display.
	go func() {
		for i := 0; i < 60; i++ {
			time.Sleep(100 * time.Millisecond)
			screen := app.Screen.GetPrimary()
			if screen == nil || screen.Size.Width == 0 || screen.Size.Height == 0 {
				continue
			}
			// screen.Size is in physical pixels for the DPI-aware window on
			// Windows. Target the original 1360x860 CSS design at this DPI,
			// capped to the visible work area.
			width := int(float64(1360) * scale)
			height := int(float64(860) * scale)
			if screen.Size.Width > 0 && width > screen.Size.Width-40 {
				width = screen.Size.Width - 40
			}
			if screen.Size.Height > 0 && height > screen.Size.Height-40 {
				height = screen.Size.Height - 40
			}
			resizeNativeWindow(uintptr(window.NativeWindow()), width, height)
			window.Center()
			log.Printf("window fitted (native): %dx%d physical (screen %dx%d, scale %.2f)", width, height, screen.Size.Width, screen.Size.Height, screen.ScaleFactor)
			return
		}
		log.Printf("screen manager did not report a primary screen; keeping initial window")
	}()
	window.Center()

	if err := app.Run(); err != nil {
		log.Fatalf("应用退出：%v", err)
	}
}

func resolvePaths() (dataDir, dbPath, vaultPath string) {
	if custom := os.Getenv("XSV_DATA_DIR"); custom != "" {
		dataDir = custom
	} else {
		configDir, err := os.UserConfigDir()
		if err != nil || configDir == "" {
			configDir = "."
		}
		dataDir = filepath.Join(configDir, "XinSkillVault")
	}
	if custom := os.Getenv("XSV_VAULT_PATH"); custom != "" {
		vaultPath = custom
	} else {
		home, _ := os.UserHomeDir()
		vaultPath = filepath.Join(home, ".skill-vault")
	}
	return dataDir, filepath.Join(dataDir, "vault.db"), vaultPath
}

func seedDefaults(st *store.Store, home string) error {
	if err := st.SeedSources([]store.Source{
		{ID: "local", Name: "本地仓库", Kind: "local", Builtin: true},
		{ID: "anthropics", Name: "Anthropic Skills", Kind: "git", URL: "github.com/anthropics/skills", Builtin: true},
		{ID: "clawhub", Name: "ClawHub", Kind: "registry", URL: "clawhub.ai", Builtin: true},
		{ID: "network", Name: "网络导入", Kind: "url", URL: "", Builtin: true},
	}); err != nil {
		return err
	}

	agents := make([]store.Agent, 0, len(seedAgents))
	for i, a := range seedAgents {
		skillsPath := filepath.Join(home, filepath.FromSlash(a.Rel))
		configPath := filepath.Dir(skillsPath)
		detected := dirExists(skillsPath) || dirExists(configPath)
		agents = append(agents, store.Agent{
			ID: a.ID, Name: a.Name, Kind: "agent", SkillsPath: skillsPath, ConfigPath: configPath,
			Detected: detected, Enabled: detected, SortOrder: i,
		})
	}
	if err := st.SeedAgents(agents); err != nil {
		return err
	}

	defaults := map[string]string{
		service.SettingSkin:            "cold-white",
		service.SettingLayout:          "table",
		service.SettingLanguage:        "zh-CN",
		service.SettingFontFamily:      "system",
		service.SettingFontScale:       "1.12",
		service.SettingCheckOnStart:    "true",
		service.SettingAutoUpdateLinks: "false",
		service.SettingCloseAction:     "ask",
		service.SettingCloseToTray:     "true",
		service.SettingVaultPath:       filepath.Join(home, ".skill-vault"),
	}
	if runtime.GOOS == "windows" {
		defaults[service.SettingDefaultMode] = "junction"
	} else {
		defaults[service.SettingDefaultMode] = "symlink"
	}
	for k, v := range defaults {
		if _, ok := st.GetSetting(k); !ok {
			if err := st.SetSetting(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func skinBackground(skin string) application.RGBA {
	switch skin {
	case "cold-gray":
		return application.NewRGB(230, 232, 232)
	case "mech-yellow":
		return application.NewRGB(24, 26, 23)
	case "paper-dark":
		return application.NewRGB(22, 23, 19)
	case "paper":
		return application.NewRGB(243, 239, 230)
	default:
		return application.NewRGB(232, 238, 243)
	}
}

func settingOr(st *store.Store, key, fallback string) string {
	if v, ok := st.GetSetting(key); ok && v != "" {
		return v
	}
	return fallback
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func setupTray(app *application.App, window *application.WebviewWindow) {
	tray := app.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("Xin Skill Vault · Xin技能管理")
	showWindow := func() {
		log.Printf("tray: show window")
		window.UnMinimise()
		window.Show().Focus()
	}
	menu := app.NewMenu()
	menu.Add("显示主窗口").OnClick(func(*application.Context) { showWindow() })
	menu.Add("隐藏到托盘").OnClick(func(*application.Context) {
		log.Printf("tray: hide window")
		window.Hide()
	})
	menu.AddSeparator()
	menu.Add("退出 Xin Skill Vault").OnClick(func(*application.Context) {
		log.Printf("tray: quit requested")
		forceQuit.Store(true)
		app.Quit()
	})
	tray.SetMenu(menu)
	tray.OnClick(showWindow)
}

type logFile struct {
	file *os.File
}

func setupLogger(dataDir string) *logFile {
	dir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.SetOutput(os.Stderr)
		return &logFile{}
	}
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.SetOutput(os.Stderr)
		return &logFile{}
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
	return &logFile{file: f}
}

func (l *logFile) Close() {
	if l.file != nil {
		_ = l.file.Close()
	}
}
