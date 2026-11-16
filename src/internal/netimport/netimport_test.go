package netimport

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitHubURLs(t *testing.T) {
	tests := []struct {
		raw     string
		owner   string
		repo    string
		branch  string
		subPath string
		archive bool
	}{
		{"https://github.com/anthropics/skills", "anthropics", "skills", "", "", false},
		{"github.com/anthropics/skills/tree/main/skills/pdf", "anthropics", "skills", "main", "skills/pdf", false},
		{"https://github.com/owner/repo/blob/main/skills/foo/SKILL.md", "owner", "repo", "main", "skills/foo", false},
		{"https://raw.githubusercontent.com/owner/repo/main/skills/foo/SKILL.md", "owner", "repo", "main", "skills/foo", false},
		{"https://github.com/owner/repo/archive/refs/heads/main.zip", "", "", "", "", true},
	}
	for _, test := range tests {
		u, err := normalizeURL(test.raw)
		if err != nil {
			t.Fatalf("%s: %v", test.raw, err)
		}
		if test.archive {
			if isArchivePath(u.Path) {
				continue
			}
			t.Fatalf("%s: expected archive", test.raw)
		}
		ref, ok := parseGitHub(u)
		if !ok {
			t.Fatalf("%s: not parsed as GitHub", test.raw)
		}
		if ref.Owner != test.owner || ref.Repo != test.repo || ref.Branch != test.branch || ref.SubPath != test.subPath {
			t.Fatalf("%s: got %+v, want %+v", test.raw, ref, test)
		}
	}
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("../escape.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("nope"))
	_ = writer.Close()
	archive := filepath.Join(t.TempDir(), "bad.zip")
	if err := os.WriteFile(archive, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(archive, t.TempDir()); err == nil {
		t.Fatalf("zip traversal was accepted")
	}
}

func TestFetchDirectZip(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string]string{
		"skill-pack/SKILL.md":            "---\nname: skill-pack\ndescription: Demo\n---\n",
		"skill-pack/references/guide.md": "# Guide\n",
	}
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte(content))
	}
	_ = writer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(buffer.Bytes())
	}))
	defer server.Close()

	dest := t.TempDir()
	result, err := Fetch(context.Background(), server.URL+"/skills.zip", dest)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "archive" {
		t.Fatalf("kind = %q", result.Kind)
	}
	if _, err := os.Stat(filepath.Join(result.Root, "SKILL.md")); err != nil {
		t.Fatalf("extracted skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Root, "references", "guide.md")); err != nil {
		t.Fatalf("extracted reference missing: %v", err)
	}
}

func TestRejectUnsafeURL(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "ftp://example.com/x.zip", "https://user:pass@example.com/x.zip"} {
		if _, err := normalizeURL(raw); err == nil {
			t.Fatalf("%s was accepted", raw)
		}
	}
}
