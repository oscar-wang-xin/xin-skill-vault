package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestBundleEntityLifecycle(t *testing.T) {
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
	for _, slug := range []string{"bundle-a", "bundle-b"} {
		source := t.TempDir()
		os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: "+slug+"\ndescription: bundle\n---\n\n# B\n"), 0o644)
		if _, err := svc.ImportSkillDirectory(source); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := svc.CreateBundle("Delivery", "delivery bundle", []string{"bundle-a", "bundle-b"})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.SkillCount != 2 || len(bundle.Slugs) != 2 {
		t.Fatalf("bundle = %+v", bundle)
	}
	updated, err := svc.UpdateBundle(bundle.ID, "Delivery 2", "updated", []string{"bundle-a"})
	if err != nil || updated.Name != "Delivery 2" || updated.SkillCount != 1 {
		t.Fatalf("updated = %+v, %v", updated, err)
	}
	if skills, err := svc.ListBundleSkills(bundle.ID); err != nil || len(skills) != 1 {
		t.Fatalf("bundle skills = %+v, %v", skills, err)
	}
	if err := svc.DeleteBundle(bundle.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := svc.ListBundles(); len(list) != 0 {
		t.Fatalf("bundles after delete = %+v", list)
	}
}
