//go:build windows

package distribute

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJunctionCreateAndRemoveKeepsTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := createJunction(source, target); err != nil {
		t.Skipf("junction unavailable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "keep.txt")); err != nil {
		t.Fatalf("junction target unreadable: %v", err)
	}
	if err := removeInstalled(target, "junction"); err != nil {
		t.Fatalf("remove junction: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("junction should be gone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(source, "keep.txt")); err != nil {
		t.Fatalf("junction removal deleted target contents: %v", err)
	}
}
