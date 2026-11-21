package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestSkillTagsAndNote(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Tag Skill\ndescription: tags\n---\n\n# T\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ImportSkillDirectory(source); err != nil {
		t.Fatal(err)
	}
	updated, err := svc.SetSkillTags("tag-skill", []string{"design", "高频", "design"})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Tags) != 2 {
		t.Fatalf("tags = %+v", updated.Tags)
	}
	tags, err := svc.ListTags()
	if err != nil || len(tags) != 2 {
		t.Fatalf("tag list = %+v, %v", tags, err)
	}
	if _, err := svc.SaveSkillNote("tag-skill", "use for layout"); err != nil {
		t.Fatal(err)
	}
	note, err := svc.GetSkillNote("tag-skill")
	if err != nil || note != "use for layout" {
		t.Fatalf("note = %q, %v", note, err)
	}
}
