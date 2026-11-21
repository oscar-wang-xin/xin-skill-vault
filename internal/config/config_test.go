package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerCreatesAndPersistsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	manager, err := NewManager(path)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	current := manager.Current()
	if current.VaultPath == "" || current.PromptRoot == "" {
		t.Fatalf("defaults missing: %+v", current)
	}
	if !current.Neumorph || current.Theme != "mech-gray" {
		t.Fatalf("unexpected defaults: %+v", current)
	}

	current.Theme = "business-blue"
	current.Neumorph = false
	current.VaultPath = filepath.Join(dir, "vault")
	if err := manager.Save(current); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := NewManager(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := reloaded.Current()
	if got.Theme != "business-blue" || got.Neumorph || got.VaultPath != current.VaultPath {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestManagerRejectsSameVaultAndPromptRoot(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	cfg := manager.Current()
	cfg.PromptRoot = cfg.VaultPath
	if err := manager.Save(cfg); err == nil {
		t.Fatal("expected identical paths to be rejected")
	}
}
