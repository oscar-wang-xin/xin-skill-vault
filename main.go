package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"

	appcore "xinskillvault/internal/app"
	"xinskillvault/internal/config"
	"xinskillvault/internal/logx"
	"xinskillvault/internal/store"
	"xinskillvault/internal/version"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	dataDir, err := config.UserConfigDir()
	if err != nil {
		panic(err)
	}
	cfgManager, err := config.NewManager(filepath.Join(dataDir, "config.json"))
	if err != nil {
		panic(err)
	}
	cfg := cfgManager.Current()
	if err := appcore.EnsureLayout(cfg); err != nil {
		panic(err)
	}
	logger, closeLog, err := logx.New(filepath.Join(cfg.VaultPath, ".skillrepo", "logs"))
	if err != nil {
		panic(err)
	}
	defer closeLog()
	logger.Info("starting", "version", version.AppVersion, "config", cfgManager.Path())

	st, err := store.Open(filepath.Join(cfg.VaultPath, ".skillrepo", "vault.db"))
	if err != nil {
		logger.Error("open database", "error", err)
		panic(err)
	}
	service := appcore.NewService(cfgManager, st, logger)

	wailsApp := application.New(application.Options{
		Name:        version.AppName,
		Description: "AI Agent Skill Vault",
		Logger:      logger,
		LogLevel:    slog.LevelInfo,
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Flags: map[string]any{
			"version": version.AppVersion,
		},
		OnShutdown: func() {
			if err := st.Close(); err != nil {
				logger.Error("close database", "error", err)
			}
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            version.AppName,
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		BackgroundColour: application.NewRGB(246, 247, 248),
		URL:              "/",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 44,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
	})

	appcore.SetApplication(wailsApp)

	if err := wailsApp.Run(); err != nil {
		logger.Error("application exited", "error", err)
		os.Exit(1)
	}
}
