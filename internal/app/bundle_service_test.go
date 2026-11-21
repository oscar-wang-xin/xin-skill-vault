package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/skillpack"
	"xinskillvault/internal/store"
)

func TestImportBundleArchive(t *testing.T) {
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

	srcRoot := t.TempDir()
	members := []skillpack.BundleSkill{}
	for _, slug := range []string{"alpha-skill", "beta-skill"} {
		dir := filepath.Join(srcRoot, slug)
		os.MkdirAll(dir, 0o755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+slug+"\ndescription: bundle test\n---\n\n# "+slug+"\n"), 0o644)
		members = append(members, skillpack.BundleSkill{Slug: slug, Name: slug, Dir: dir})
	}
	bundlePath := filepath.Join(t.TempDir(), "demo.bundle.zip")
	if err := skillpack.ExportBundle(bundlePath, "Demo Bundle", members); err != nil {
		t.Fatal(err)
	}
	result, err := svc.ImportBundleArchive(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Imported) != 2 || result.Imported[0] != "alpha-skill" || result.Imported[1] != "beta-skill" {
		t.Fatalf("imported = %+v", result)
	}
	again, err := svc.ImportBundleArchive(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Imported) != 0 || len(again.Skipped) != 2 {
		t.Fatalf("duplicate import = %+v", again)
	}
}
