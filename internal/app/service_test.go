package app

import (
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestServiceSummaryAndConfigSave(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config", "config.json")
	manager, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("config manager: %v", err)
	}
	cfg := manager.Current()
	cfg.VaultPath = filepath.Join(root, "vault")
	cfg.PromptRoot = filepath.Join(root, "prompts")
	if err := manager.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := EnsureLayout(cfg); err != nil {
		t.Fatalf("layout: %v", err)
	}
	st, err := store.Open(filepath.Join(cfg.VaultPath, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	service := NewService(manager, st, nil)

	info := service.GetAppInfo()
	if info.Name == "" || info.SchemaVersion != 1 {
		t.Fatalf("unexpected info: %+v", info)
	}
	summary, err := service.GetVaultSummary()
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.SkillCount != 0 || summary.SchemaVersion != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	next := manager.Current()
	next.Theme = "mint"
	result, err := service.SaveConfig(next)
	if err != nil {
		t.Fatalf("save service config: %v", err)
	}
	if result.RestartRequired {
		t.Fatal("theme-only change must not require restart")
	}
	if manager.Current().Theme != "mint" {
		t.Fatal("theme was not persisted")
	}
}
