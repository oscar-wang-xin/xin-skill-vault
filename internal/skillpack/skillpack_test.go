package skillpack

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportSkillPackage(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Pack Demo\ndescription: pack\n---\n\n# Pack\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "references", "note.md"), []byte("note"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "pack.skill")
	if err := Export(source, dest, "pack-demo", "Pack Demo", "1.0.0", "pack", "hash"); err != nil {
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
	for _, expected := range []string{"SKILL.md", "references/note.md", ManifestName} {
		if !names[expected] {
			t.Fatalf("missing %s in %v", expected, names)
		}
	}
}
