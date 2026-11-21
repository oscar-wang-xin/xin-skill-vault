package distribute

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

func TestInstallCopyAndUninstall(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Distribute Demo\ndescription: copy distribute\n---\n\n# Demo\n"), 0o644); err != nil {
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
	targetRoot := filepath.Join(t.TempDir(), "agent-skills")
	target := store.TargetRecord{
		ID: "agent:demo", Kind: "agent", AgentID: "demo", Name: "Demo Agent",
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
	if assignment.Mode != "copy" {
		t.Fatalf("mode = %s", assignment.Mode)
	}
	installed := filepath.Join(targetRoot, record.Slug)
	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(installed, ".skillrepo-manifest.json")); err != nil {
		t.Fatal(err)
	}
	if err := d.Uninstall(record.ID, target.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installed); !os.IsNotExist(err) {
		t.Fatalf("installed path still exists: %v", err)
	}
	if assignment, _ := st.FindAssignment(record.ID, target.ID); assignment != nil {
		t.Fatal("assignment should be deleted")
	}
}
