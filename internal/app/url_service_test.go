package app

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestImportSkillURLConflict(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, _ := zw.Create("skill/SKILL.md")
	io.WriteString(w, "---\nname: URL Skill\ndescription: v1\n---\n\n# v1\n")
	zw.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/skill.zip" {
			w.Write(archive.Bytes())
			return
		}
		io.WriteString(w, "---\nname: URL Skill\ndescription: v2\n---\n\n# v2\n")
	}))
	defer server.Close()

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

	result, err := svc.ImportSkillURL(server.URL+"/skill.zip", "skip")
	if err != nil || result.Status != "imported" || result.Skill.SourceType != "url" {
		t.Fatalf("import = %+v, %v", result, err)
	}
	result, err = svc.ImportSkillURL(server.URL+"/SKILL.md", "skip")
	if err != nil || result.Status != "skipped" {
		t.Fatalf("skip = %+v, %v", result, err)
	}
	result, err = svc.ImportSkillURL(server.URL+"/SKILL.md", "overwrite")
	if err != nil || result.Status != "overwritten" {
		t.Fatalf("overwrite = %+v, %v", result, err)
	}
	detail, err := svc.GetSkillDetail("url-skill")
	if err != nil || detail.Description != "v2" {
		t.Fatalf("detail = %+v, %v", detail, err)
	}
}
