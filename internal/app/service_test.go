package app

import (
	"os"
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
	if info.Name == "" || info.SchemaVersion != 4 {
		t.Fatalf("unexpected info: %+v", info)
	}
	summary, err := service.GetVaultSummary()
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.SkillCount != 0 || summary.SchemaVersion != 4 {
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

func TestServiceImportAndListSkills(t *testing.T) {
	root := t.TempDir()
	manager, err := config.NewManager(filepath.Join(root, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := manager.Current()
	cfg.VaultPath = filepath.Join(root, "vault")
	cfg.PromptRoot = filepath.Join(root, "prompts")
	if err := manager.Save(cfg); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLayout(cfg); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(cfg.VaultPath, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	service := NewService(manager, st, nil)

	source := t.TempDir()
	doc := `---
name: Imported Skill
description: service import test
version: 1.0.0
---

# Hello
`
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	item, err := service.ImportSkillDirectory(source)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if item.Slug != "imported-skill" || item.ContentHash == "" {
		t.Fatalf("unexpected item: %+v", item)
	}
	list, err := service.ListSkills()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	detail, err := service.GetSkillDetail("imported-skill")
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail == nil || detail.Name != "Imported Skill" || detail.Body == "" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	summary, err := service.GetVaultSummary()
	if err != nil || summary.SkillCount != 1 {
		t.Fatalf("summary = %+v, %v", summary, err)
	}
}
