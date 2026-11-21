package vault

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/store"
)

func TestImportDirectoryAndListFiles(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	doc := `---
name: Demo Skill
description: A skill used by the import test
version: 0.1.0
---

# Demo

Body.
`
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "references", "guide.md"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	v := New(root, st)
	record, err := v.ImportDirectory(source)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if record.Slug != "demo-skill" || record.FileCount != 2 || record.ContentHash == "" {
		t.Fatalf("unexpected record: %+v", record)
	}
	if _, err := os.Stat(filepath.Join(v.SkillsDir(), "demo-skill", "SKILL.md")); err != nil {
		t.Fatalf("imported SKILL.md missing: %v", err)
	}
	files, err := v.SkillFiles("demo-skill")
	if err != nil {
		t.Fatalf("files: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("files = %d, want 3 (directory + two files)", len(files))
	}
	if _, err := v.ImportDirectory(source); err == nil {
		t.Fatal("expected duplicate import to fail")
	}
}

func TestImportRejectsVaultInternalSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "skills", "inside")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := "---\nname: Inside\ndescription: inside\n---\n"
	os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(doc), 0o644)
	st, err := store.Open(filepath.Join(root, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := New(root, st).ImportDirectory(source); err == nil {
		t.Fatal("expected internal source to be rejected")
	}
}
