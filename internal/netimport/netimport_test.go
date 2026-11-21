package netimport

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFetchZipAndRaw(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	add := func(name, body string) {
		w, _ := zw.Create(name)
		io.WriteString(w, body)
	}
	add("demo/SKILL.md", "---\nname: URL Demo\ndescription: url test\n---\n\n# Demo\n")
	add("demo/references/note.md", "note")
	zw.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/skill.zip" {
			w.Write(archive.Bytes())
			return
		}
		if r.URL.Path == "/SKILL.md" {
			io.WriteString(w, "---\nname: Raw Demo\ndescription: raw\n---\n")
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	fetcher := NewFetcher()

	zipRoot := t.TempDir()
	dir, err := fetcher.Fetch(context.Background(), server.URL+"/skill.zip", zipRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatalf("zip skill missing: %v", err)
	}
	rawRoot := t.TempDir()
	dir, err = fetcher.Fetch(context.Background(), server.URL+"/SKILL.md", rawRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatalf("raw skill missing: %v", err)
	}
}

func TestSafeName(t *testing.T) {
	if _, err := safeName("../x"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
	if _, err := safeName("C:/x"); err == nil {
		t.Fatal("drive path should be rejected")
	}
}
