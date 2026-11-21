package scan

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/agent"
	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

func TestScanNewImportedAndChanged(t *testing.T) {
	root := t.TempDir()
	agentRoot := filepath.Join(root, "agent-skills")
	skillDir := filepath.Join(agentRoot, "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDoc := func(version string) {
		doc := "---\nname: Demo\ndescription: scan test\nversion: " + version + "\n---\n\n# Demo\n"
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeDoc("1")
	vaultRoot := t.TempDir()
	st, err := store.Open(filepath.Join(vaultRoot, ".skillrepo", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	adapters := []agent.Adapter{{ID: "demo-agent", Name: "Demo Agent", Path: agentRoot, Detected: true}}
	result := Scan(adapters, st)
	if len(result.Findings) != 1 || result.Findings[0].Status != "new" {
		t.Fatalf("new scan = %+v", result)
	}
	if _, err := vault.New(vaultRoot, st).ImportDirectory(skillDir); err != nil {
		t.Fatal(err)
	}
	result = Scan(adapters, st)
	if len(result.Findings) != 0 {
		t.Fatalf("imported skill should not be a finding: %+v", result.Findings)
	}
	writeDoc("2")
	result = Scan(adapters, st)
	if len(result.Findings) != 1 || result.Findings[0].Status != "changed" {
		t.Fatalf("changed scan = %+v", result.Findings)
	}
}
