package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestPromptLifecycleAndVersions(t *testing.T) {
	root := t.TempDir()
	manager, _ := config.NewManager(filepath.Join(root, "config.json"))
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
	svc := NewService(manager, st, nil)

	category, err := svc.CreatePromptCategory("Writing", "", "")
	if err != nil {
		t.Fatal(err)
	}
	detail, err := svc.CreatePrompt("Outline", category.ID, []string{"writing"}, "# v1\n")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfg.PromptRoot, "Writing", detail.ID+"-v1.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("prompt file missing: %v", err)
	}
	updated, err := svc.SavePrompt(detail.ID, "Outline 2", category.ID, []string{"writing", "outline"}, "# v2\n")
	if err != nil {
		t.Fatal(err)
	}
	if updated.CurrentVersion != 2 || updated.Content != "# v2\n" || len(updated.Versions) != 2 {
		t.Fatalf("updated = %+v", updated)
	}
	restored, err := svc.DeletePromptVersion(detail.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if restored.CurrentVersion != 1 || restored.Content != "# v1\n" || len(restored.Versions) != 1 {
		t.Fatalf("restored = %+v", restored)
	}
	if _, err := svc.DeletePromptVersion(detail.ID, 1); err == nil {
		t.Fatal("deleting last version should fail")
	}
	if err := svc.DeletePrompt(detail.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("prompt file should be removed: %v", err)
	}
}
