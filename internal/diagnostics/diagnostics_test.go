package diagnostics

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportDiagnostics(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "diag.zip")
	report := Report{AppName: "Test", AppVersion: "0.1.0", SchemaVersion: 4}
	if err := Export(dest, report, map[string]string{"vaultPath": "x"}, ""); err != nil {
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
	for _, expected := range []string{"diagnostics.json", "config.json", "README.txt"} {
		if !names[expected] {
			t.Fatalf("missing %s in %v", expected, names)
		}
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
}
