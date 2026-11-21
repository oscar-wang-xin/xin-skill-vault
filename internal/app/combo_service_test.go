package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestComboLifecycle(t *testing.T) {
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
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Combo Skill\ndescription: combo test\n---\n\n# C\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSkillDirectory(source); err != nil {
		t.Fatal(err)
	}
	combo, err := svc.CreateCombo("Frontend", "frontend combo", "✦", []string{"combo-skill"})
	if err != nil {
		t.Fatal(err)
	}
	if combo.SkillCount != 1 || len(combo.Slugs) != 1 {
		t.Fatalf("combo = %+v", combo)
	}
	updated, err := svc.UpdateCombo(combo.ID, "Frontend 2", "updated", "◆", []string{"combo-skill"})
	if err != nil || updated.Name != "Frontend 2" {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	skills, err := svc.ListComboSkills(combo.ID)
	if err != nil || len(skills) != 1 || skills[0].Slug != "combo-skill" {
		t.Fatalf("skills = %+v, %v", skills, err)
	}
	if err := svc.DeleteCombo(combo.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := svc.ListCombos(); len(list) != 0 {
		t.Fatalf("combos after delete = %+v", list)
	}
}
