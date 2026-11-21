package skillpack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportBundle(t *testing.T) {
	root := t.TempDir()
	skills := []BundleSkill{}
	for _, slug := range []string{"alpha", "beta"} {
		dir := filepath.Join(root, slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+slug+"\ndescription: test\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		skills = append(skills, BundleSkill{Slug: slug, Name: slug, Dir: dir})
	}
	dest := filepath.Join(root, "bundle.zip")
	if err := ExportBundle(dest, "demo", skills); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	names := map[string]bool{}
	for _, entry := range reader.File {
		names[entry.Name] = true
	}
	for _, expected := range []string{"bundle.json", "skills/alpha/SKILL.md", "skills/beta/SKILL.md"} {
		if !names[expected] {
			t.Fatalf("missing %s in %v", expected, names)
		}
	}
}
