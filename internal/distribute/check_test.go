package distribute

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

func TestCheckDetectsStaleMissingAndRepair(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	doc := "---\nname: Drift Demo\ndescription: drift test\nversion: 1\n---\n\n# Drift\n"
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(root, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	v := vault.New(root, st)
	record, err := v.ImportDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "agent")
	target := store.TargetRecord{
		ID: "agent:demo", Kind: "agent", AgentID: "demo", Name: "Demo",
		Path: targetRoot, Enabled: true, Detected: true, Writable: true, DefaultMode: "copy", CreatedAt: 1,
	}
	if err := st.UpsertTarget(target); err != nil {
		t.Fatal(err)
	}
	d := Distributor{Vault: v, Store: st}
	assignment, err := d.Install(target, record.Slug, "copy")
	if err != nil {
		t.Fatal(err)
	}
	reports, err := Check(st, v)
	if err != nil || len(reports) != 1 || reports[0].Status != "ok" {
		t.Fatalf("initial drift = %+v, %v", reports, err)
	}
	if _, err := os.Stat(assignment.InstalledPath); err != nil {
		t.Fatal(err)
	}
	next := "---\nname: Drift Demo\ndescription: drift test v2\nversion: 2\n---\n\n# Drift 2\n"
	if _, err := v.ReplaceFromDirectory(record.Slug, sourceWithDoc(t, next), "scan", source, "scan update"); err != nil {
		t.Fatal(err)
	}
	reports, err = Check(st, v)
	if err != nil || len(reports) != 1 || reports[0].Status != "stale" {
		t.Fatalf("stale drift = %+v, %v", reports, err)
	}
	if _, err := Repair(st, v, record.Slug, target.AgentID); err != nil {
		t.Fatal(err)
	}
	reports, _ = Check(st, v)
	if reports[0].Status != "ok" {
		t.Fatalf("repair status = %+v", reports[0])
	}
	if err := os.RemoveAll(assignment.InstalledPath); err != nil {
		t.Fatal(err)
	}
	reports, _ = Check(st, v)
	if reports[0].Status != "missing" {
		t.Fatalf("missing status = %+v", reports[0])
	}
}

func sourceWithDoc(t *testing.T, doc string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}
