package vault

import (
	"archive/zip"
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

func TestImportArchiveAndSaveDocument(t *testing.T) {
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	v := New(root, st)

	archivePath := filepath.Join(t.TempDir(), "demo.zip")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(archiveFile)
	doc := "---\nname: Archive Skill\ndescription: archive import test\nversion: 0.1.0\n---\n\n# Archive\n"
	writeZip := func(name, body string) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	writeZip("bundle/SKILL.md", doc)
	writeZip("bundle/references/notes.md", "notes")
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatal(err)
	}

	record, err := v.ImportArchive(archivePath)
	if err != nil {
		t.Fatalf("import archive: %v", err)
	}
	if record.SourceType != "zip" || record.SourceRef != archivePath || record.Slug != "archive-skill" {
		t.Fatalf("unexpected archive record: %+v", record)
	}
	stored, err := st.FindSkillBySlug("archive-skill")
	if err != nil || stored == nil || stored.SourceType != "zip" || stored.SourceRef != archivePath {
		t.Fatalf("source metadata not persisted: %+v, %v", stored, err)
	}

	next := "---\nname: Archive Skill\ndescription: archive import test v2\nversion: 0.2.0\n---\n\n# Archive v2\n"
	updated, err := v.SaveDocument("archive-skill", next, "编辑前")
	if err != nil {
		t.Fatalf("save document: %v", err)
	}
	if updated.ContentHash == record.ContentHash || updated.Version != "0.2.0" {
		t.Fatalf("document update did not refresh index: %+v", updated)
	}
	content, err := v.ReadDocument("archive-skill")
	if err != nil || content != next {
		t.Fatalf("read document = %q, %v", content, err)
	}
	snapshots, err := v.Snapshots("archive-skill")
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("snapshots = %+v, %v", snapshots, err)
	}
	if _, err := os.Stat(snapshots[0].FilePath); err != nil {
		t.Fatalf("snapshot file missing: %v", err)
	}
	if _, err := v.SaveDocument("archive-skill", "not valid", ""); err == nil {
		t.Fatal("expected invalid document to be rejected")
	}
	after, _ := v.ReadDocument("archive-skill")
	if after != next {
		t.Fatal("invalid save must not change SKILL.md")
	}
}

func TestSkillEnableTrashAndSnapshotActions(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	doc := "---\nname: Lifecycle Skill\ndescription: lifecycle test\n---\n\n# Life\n"
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	v := New(root, st)
	record, err := v.ImportDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := v.SetEnabled(record.Slug, false)
	if err != nil || disabled.Enabled {
		t.Fatalf("disable failed: %+v, %v", disabled, err)
	}
	next := "---\nname: Lifecycle Skill\ndescription: lifecycle test v2\n---\n\n# Life 2\n"
	if _, err := v.SaveDocument(record.Slug, next, "编辑前"); err != nil {
		t.Fatal(err)
	}
	snaps, err := v.Snapshots(record.Slug)
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots: %+v, %v", snaps, err)
	}
	restored, err := v.RestoreSnapshot(record.Slug, snaps[0].ID)
	if err != nil || restored.Description != "lifecycle test" {
		t.Fatalf("restore failed: %+v, %v", restored, err)
	}
	content, _ := v.ReadDocument(record.Slug)
	if content != doc {
		t.Fatalf("restored content = %q", content)
	}
	afterRestore, _ := v.Snapshots(record.Slug)
	if len(afterRestore) != 2 {
		t.Fatalf("restore should create a new snapshot, got %d", len(afterRestore))
	}
	if err := v.DeleteSnapshot(record.Slug, afterRestore[0].ID); err != nil {
		t.Fatal(err)
	}
	afterDelete, _ := v.Snapshots(record.Slug)
	if len(afterDelete) != 1 {
		t.Fatalf("delete snapshot left %d", len(afterDelete))
	}
	trashPath, err := v.Trash(record.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(trashPath); err != nil {
		t.Fatalf("trash missing: %v", err)
	}
	if _, err := v.Store.FindSkillBySlug(record.Slug); err != nil {
		t.Fatal(err)
	}
	list, _ := v.Store.ListSkills()
	if len(list) != 0 {
		t.Fatalf("trashed skill still listed: %+v", list)
	}
}
