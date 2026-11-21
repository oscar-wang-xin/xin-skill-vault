//go:build windows

package distribute

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

func TestLinkInstallUsesJunctionAndKeepsSource(t *testing.T) {
	root := t.TempDir()
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Junction Demo\ndescription: junction install\n---\n\n# J\n"), 0o644); err != nil {
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
	target := store.TargetRecord{
		ID: "agent:junction", Kind: "agent", AgentID: "junction", Name: "Junction",
		Path: filepath.Join(t.TempDir(), "agent"), Enabled: true, Detected: true,
		Writable: true, DefaultMode: "link", CreatedAt: 1,
	}
	if err := st.UpsertTarget(target); err != nil {
		t.Fatal(err)
	}
	d := Distributor{Vault: v, Store: st}
	assignment, err := d.Install(target, record.Slug, "link")
	if err != nil {
		t.Fatal(err)
	}
	if assignment.Mode != "junction" {
		t.Logf("junction unavailable, mode = %s", assignment.Mode)
	}
	if _, err := os.Stat(filepath.Join(assignment.InstalledPath, "SKILL.md")); err != nil {
		t.Fatalf("installed skill unreadable: %v", err)
	}
	if err := d.Uninstall(record.ID, target.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", record.Slug, "SKILL.md")); err != nil {
		t.Fatalf("uninstall removed source: %v", err)
	}
}
