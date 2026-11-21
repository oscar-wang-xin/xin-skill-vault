package store

import (
	"path/filepath"
	"testing"
)

func TestOpenMigratesAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	version, err := s.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion: %v", err)
	}
	if version != 3 {
		t.Fatalf("schema version = %d, want 3", version)
	}
	for _, table := range []string{"skills", "targets", "assignments", "prompt_categories", "prompt_versions", "snapshots", "projects", "project_agents", "audit_log"} {
		var name string
		if err := s.DB().QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if v, _ := reopened.SchemaVersion(); v != 3 {
		t.Fatalf("schema version after reopen = %d, want 3", v)
	}
}

func TestCountsOnEmptyDatabase(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "vault.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if n, err := s.SkillCount(); err != nil || n != 0 {
		t.Fatalf("SkillCount = %d, %v", n, err)
	}
	if n, err := s.TargetCount(); err != nil || n != 0 {
		t.Fatalf("TargetCount = %d, %v", n, err)
	}
	if n, err := s.AssignmentCount(); err != nil || n != 0 {
		t.Fatalf("AssignmentCount = %d, %v", n, err)
	}
}
