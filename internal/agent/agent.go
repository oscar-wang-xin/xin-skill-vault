package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Adapter struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Short      string `json:"short"`
	Color      string `json:"color"`
	Path       string `json:"path"`
	ConfigPath string `json:"configPath"`
	Mode       string `json:"mode"`
	Detected   bool   `json:"detected"`
}

func Defaults() []Adapter {
	home, _ := os.UserHomeDir()
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData, _ = os.UserConfigDir()
	}
	join := func(parts ...string) string { return filepath.Join(parts...) }
	adapters := []Adapter{
		{ID: "claude-code", Name: "Claude Code", Short: "CC", Color: "#D97757", Path: join(home, ".claude", "skills"), ConfigPath: join(home, ".claude"), Mode: "symlink"},
		{ID: "codex", Name: "Codex CLI", Short: "CX", Color: "#111827", Path: join(home, ".codex", "skills"), ConfigPath: join(home, ".codex"), Mode: "symlink"},
		{ID: "cursor", Name: "Cursor", Short: "CU", Color: "#2563EB", Path: join(home, ".cursor", "skills"), ConfigPath: join(home, ".cursor"), Mode: "symlink"},
		{ID: "opencode", Name: "OpenCode", Short: "OC", Color: "#0EA5E9", Path: join(home, ".config", "opencode", "skills"), ConfigPath: join(home, ".config", "opencode"), Mode: "symlink"},
		{ID: "workbuddy", Name: "WorkBuddy", Short: "WB", Color: "#7C3AED", Path: join(home, ".workbuddy", "skills"), ConfigPath: join(home, ".workbuddy"), Mode: "copy"},
		{ID: "dsh", Name: "DeepSeek Harness", Short: "DS", Color: "#2563EB", Path: join(home, ".dsh", "skills"), ConfigPath: join(home, ".dsh"), Mode: "symlink"},
		{ID: "alma", Name: "Alma", Short: "AL", Color: "#0E9F74", Path: join(home, ".config", "alma", "skills"), ConfigPath: join(home, ".config", "alma"), Mode: "symlink"},
		{ID: "codebuddy", Name: "CodeBuddy", Short: "CB", Color: "#D97706", Path: join(home, ".codebuddy", "skills"), ConfigPath: join(home, ".codebuddy"), Mode: "symlink"},
		{ID: "reasonix", Name: "Reasonix", Short: "RX", Color: "#64748B", Path: join(home, ".reasonix", "skills"), ConfigPath: join(home, ".reasonix"), Mode: "symlink"},
		{ID: "agents", Name: "Universal (.agents)", Short: "AG", Color: "#0F766E", Path: join(home, ".agents", "skills"), ConfigPath: join(home, ".agents"), Mode: "symlink"},
	}
	if runtime.GOOS == "windows" {
		adapters = append(adapters, Adapter{
			ID: "cherry-studio", Name: "Cherry Studio", Short: "CH", Color: "#E11D48",
			Path: join(appData, "CherryStudio", "Data", "Skills"), ConfigPath: join(appData, "CherryStudio"), Mode: "symlink",
		})
	} else if runtime.GOOS == "darwin" {
		adapters = append(adapters, Adapter{
			ID: "cherry-studio", Name: "Cherry Studio", Short: "CH", Color: "#E11D48",
			Path:       join(home, "Library", "Application Support", "CherryStudio", "Data", "Skills"),
			ConfigPath: join(home, "Library", "Application Support", "CherryStudio"), Mode: "symlink",
		})
	} else {
		adapters = append(adapters, Adapter{
			ID: "cherry-studio", Name: "Cherry Studio", Short: "CH", Color: "#E11D48",
			Path:       join(home, ".config", "CherryStudio", "Data", "Skills"),
			ConfigPath: join(home, ".config", "CherryStudio"), Mode: "symlink",
		})
	}
	for i := range adapters {
		adapters[i].Path = clean(adapters[i].Path)
		adapters[i].ConfigPath = clean(adapters[i].ConfigPath)
		if info, err := os.Stat(adapters[i].Path); err == nil && info.IsDir() {
			adapters[i].Detected = true
		}
	}
	return adapters
}

func clean(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return strings.TrimRight(abs, string(filepath.Separator))
}
