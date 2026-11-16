package vault

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T) (*Manager, *store.Store, string, string) {
	t.Helper()
	base := t.TempDir()
	st, err := store.Open(filepath.Join(base, "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.SeedSources([]store.Source{
		{ID: "local", Name: "本地仓库", Kind: "local", Builtin: true},
		{ID: "anthropics", Name: "Anthropic Skills", Kind: "git", Builtin: true},
	}); err != nil {
		t.Fatal(err)
	}
	agentSkills := filepath.Join(base, "agent", "skills")
	if err := st.SeedAgents([]store.Agent{{
		ID: "test-agent", Name: "Test Agent", SkillsPath: agentSkills,
		ConfigPath: filepath.Join(base, "agent"), Enabled: true,
	}}); err != nil {
		t.Fatal(err)
	}
	m := New(filepath.Join(base, "vault"), st)
	if err := m.Ensure(); err != nil {
		t.Fatal(err)
	}
	return m, st, base, agentSkills
}

func writeSkill(t *testing.T, dir, name, desc string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "SKILL.md"), "---\nname: "+name+"\ndescription: "+desc+"\nversion: 1.2.3\n---\n\n# "+name+"\n\nUse this skill carefully.\n")
	writeFile(t, filepath.Join(dir, "references", "guide.md"), "# Guide\n\nSome reference content.\n")
}

func TestImportDistributeAndRemove(t *testing.T) {
	m, st, base, agentSkills := setup(t)
	source := filepath.Join(base, "incoming", "pdf-tools")
	writeSkill(t, source, "pdf-tools", "Read and edit PDF files.")

	skill, err := m.Import(source)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if skill.ID != "pdf-tools" {
		t.Fatalf("unexpected slug: %q", skill.ID)
	}
	if skill.Version != "1.2.3" {
		t.Fatalf("version not parsed: %q", skill.Version)
	}
	if skill.FileCount != 2 {
		t.Fatalf("file count = %d, want 2", skill.FileCount)
	}
	if _, err := os.Stat(filepath.Join(m.SkillPath(skill.ID), "SKILL.md")); err != nil {
		t.Fatalf("imported skill missing: %v", err)
	}

	// A rescan must not overwrite source attribution with "local".
	skill.SourceID = "anthropics"
	skill.SourceRef = "skills/pdf-tools"
	if err := st.UpsertSkill(skill); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Scan(); err != nil {
		t.Fatal(err)
	}
	rescanned, err := st.GetSkill(skill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rescanned.SourceID != "anthropics" || rescanned.SourceRef != "skills/pdf-tools" {
		t.Fatalf("source attribution lost on scan: %+v", rescanned)
	}
	skill = rescanned

	if _, err := m.Distribute(skill.ID, []string{"test-agent"}, defaultLinkMode()); err != nil {
		t.Fatalf("Distribute: %v", err)
	}
	linkPath := filepath.Join(agentSkills, skill.ID)
	if _, err := os.Stat(linkPath); err != nil {
		t.Fatalf("distribution missing: %v", err)
	}
	if !IsLink(linkPath) {
		t.Fatalf("expected a link at %s", linkPath)
	}
	if !SameLocation(linkPath, skill.VaultPath) {
		t.Fatalf("link does not resolve to the vault copy")
	}
	if err := m.CheckHealth(); err != nil {
		t.Fatal(err)
	}
	dist, err := st.GetDistribution(skill.ID, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if dist.Status != "ok" {
		t.Fatalf("status = %q, want ok", dist.Status)
	}
	if dist.Mode != defaultLinkMode() {
		t.Fatalf("mode = %q, want %q", dist.Mode, defaultLinkMode())
	}

	if err := m.RemoveDistribution(skill.ID, "test-agent"); err != nil {
		t.Fatalf("RemoveDistribution: %v", err)
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("link still exists after removal: %v", err)
	}

	if _, err := m.Distribute(skill.ID, []string{"test-agent"}, "copy"); err != nil {
		t.Fatalf("copy distribute: %v", err)
	}
	if IsLink(linkPath) {
		t.Fatalf("copy should not be a link")
	}
	if _, err := os.Stat(filepath.Join(linkPath, "references", "guide.md")); err != nil {
		t.Fatalf("copied content missing: %v", err)
	}
	if err := m.CheckHealth(); err != nil {
		t.Fatal(err)
	}
	dist, _ = st.GetDistribution(skill.ID, "test-agent")
	if dist.Mode != "copy" || dist.Status != "ok" {
		t.Fatalf("copy distribution = %+v", dist)
	}

	if err := m.RemoveSkill(skill.ID); err != nil {
		t.Fatalf("RemoveSkill: %v", err)
	}
	if _, err := st.GetSkill(skill.ID); err == nil {
		t.Fatalf("skill row still present")
	}
	entries, _ := os.ReadDir(m.TrashDir())
	if len(entries) == 0 {
		t.Fatalf("expected a trash entry")
	}
}

func TestStatusDetection(t *testing.T) {
	m, st, _, agentSkills := setup(t)
	source := filepath.Join(t.TempDir(), "demo")
	writeSkill(t, source, "demo-skill", "Demo skill.")
	skill, err := m.Import(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Distribute(skill.ID, []string{"test-agent"}, "copy"); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(agentSkills, skill.ID)
	writeFile(t, filepath.Join(linkPath, "references", "guide.md"), "drifted")
	if err := m.CheckHealth(); err != nil {
		t.Fatal(err)
	}
	dist, _ := st.GetDistribution(skill.ID, "test-agent")
	if dist.Status != "drifted" {
		t.Fatalf("status = %q, want drifted", dist.Status)
	}
	if err := os.RemoveAll(linkPath); err != nil {
		t.Fatal(err)
	}
	if err := m.CheckHealth(); err != nil {
		t.Fatal(err)
	}
	dist, _ = st.GetDistribution(skill.ID, "test-agent")
	if dist.Status != "broken" {
		t.Fatalf("status = %q, want broken", dist.Status)
	}
}

func TestImportables(t *testing.T) {
	m, st, base, agentSkills := setup(t)
	writeSkill(t, filepath.Join(agentSkills, "agent-skill"), "agent-skill", "Found in an agent directory.")
	items, err := m.Importables()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d importables, want 1", len(items))
	}
	if items[0].Name != "agent-skill" || items[0].Slug != "agent-skill" {
		t.Fatalf("unexpected importable: %+v", items[0])
	}
	if len(items[0].Agents) != 1 || items[0].Agents[0] != "Test Agent" {
		t.Fatalf("agent names not aggregated: %+v", items[0].Agents)
	}
	// Importing does not take over the original directory (v1 behavior), so
	// the source stays listed but is marked as already installed.
	skill, err := m.Import(items[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	items, _ = m.Importables()
	if len(items) != 1 || !items[0].AlreadyInstalled {
		t.Fatalf("imported source not marked installed: %+v", items)
	}

	// A different agent that links to the vault must never be offered as an
	// import source: the link points at the vault copy itself.
	agent2Skills := filepath.Join(base, "agent2", "skills")
	if err := st.SeedAgents([]store.Agent{{
		ID: "test-agent-2", Name: "Test Agent 2", SkillsPath: agent2Skills,
		ConfigPath: filepath.Join(base, "agent2"), Detected: true, Enabled: true,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Distribute(skill.ID, []string{"test-agent-2"}, defaultLinkMode()); err != nil {
		t.Fatal(err)
	}
	items, _ = m.Importables()
	for _, item := range items {
		for _, agentName := range item.Agents {
			if agentName == "Test Agent 2" {
				t.Fatalf("vault-owned link offered as importable: %+v", item)
			}
		}
	}
}

func TestFrontmatterAndSlug(t *testing.T) {
	meta := parseFrontmatter("---\nname: \"My Skill\"\ndescription: >-\n  A long description\n  on two lines.\nversion: 2.0.0\n---\nBody\n")
	if meta.Name != "My Skill" {
		t.Fatalf("name = %q", meta.Name)
	}
	if meta.Description != "A long description on two lines." {
		t.Fatalf("description = %q", meta.Description)
	}
	if got := slugify("My Skill / v2"); got != "my-skill-v2" {
		t.Fatalf("slug = %q", got)
	}
	if runtime.GOOS == "windows" {
		if got := slugify("中文技能"); got != "中文技能" {
			t.Fatalf("unicode slug = %q", got)
		}
	}
}

func TestDiscoverFolderRecursive(t *testing.T) {
	m, _, base, _ := setup(t)
	root := filepath.Join(base, "downloads", "skill-pack")
	writeSkill(t, filepath.Join(root, "skills", "one"), "one", "First skill.")
	writeSkill(t, filepath.Join(root, "nested", "deep", "two"), "two", "Second skill.")
	writeSkill(t, filepath.Join(root, "skills", "one", "nested", "three"), "three", "Should not be a separate skill.")
	writeFile(t, filepath.Join(root, "README.md"), "# pack\n")

	items, err := m.DiscoverFolder(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d discovered skills, want 2: %+v", len(items), items)
	}
	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		if item.SourceID != "local" || item.SourceRef == "" {
			t.Fatalf("source fields missing: %+v", item)
		}
	}
	if !names["one"] || !names["two"] {
		t.Fatalf("unexpected discoveries: %v", names)
	}
}

func TestImportablesTracesLinkSource(t *testing.T) {
	m, _, base, agentSkills := setup(t)
	source := filepath.Join(base, "external", "source-skill")
	writeSkill(t, source, "source-skill", "A skill outside the vault.")
	linkPath := filepath.Join(agentSkills, "source-skill")
	if err := os.MkdirAll(agentSkills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CreateDirLink(linkPath, source); err != nil {
		t.Fatal(err)
	}

	items, err := m.Importables()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("importables = %d, want 1: %+v", len(items), items)
	}
	if !items[0].IsLink {
		t.Fatalf("expected link source")
	}
	if !SameLocation(items[0].Path, source) {
		t.Fatalf("importable path %q is not the real source %q", items[0].Path, source)
	}
	skill, err := m.Import(items[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if skill.SourceID != "local" {
		t.Fatalf("source id = %q", skill.SourceID)
	}
	if _, err := os.Stat(filepath.Join(skill.VaultPath, "references", "guide.md")); err != nil {
		t.Fatalf("source content was not copied: %v", err)
	}
	if !SameLocation(filepath.Dir(items[0].Path), filepath.Dir(source)) {
		t.Fatalf("source directory mismatch")
	}
}

func TestInspectReportsContentFiles(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "content-skill", "A skill with scripts.")
	writeFile(t, filepath.Join(dir, "scripts", "install.ps1"), "Write-Host 'install'`n")
	writeFile(t, filepath.Join(dir, "bin", "helper.exe"), "MZ-not-really-a-binary")

	insp, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if insp.Content != "executable" {
		t.Fatalf("content = %q", insp.Content)
	}
	joined := strings.Join(insp.ContentFiles, "\n")
	for _, want := range []string{"scripts/install.ps1（脚本）", "bin/helper.exe（可执行文件）"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing content file %q in %q", want, joined)
		}
	}
}

func TestInspectScriptOnlyDoesNotJudgeContent(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "script-skill", "A skill with a shell script.")
	writeFile(t, filepath.Join(dir, "scripts", "bootstrap.sh"), "curl -fsSL https://example.com/install.sh | sh`n")

	insp, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if insp.Content != "script" {
		t.Fatalf("content = %q", insp.Content)
	}
	if len(insp.ContentFiles) != 1 || !strings.Contains(insp.ContentFiles[0], "bootstrap.sh（脚本）") {
		t.Fatalf("unexpected content files: %+v", insp.ContentFiles)
	}
}

func TestInspectDocsOnlyHasNoScripts(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "docs-only", "Only documentation.")
	insp, err := Inspect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if insp.Content != "none" {
		t.Fatalf("content = %q", insp.Content)
	}
	if len(insp.ContentFiles) != 0 {
		t.Fatalf("unexpected content files: %+v", insp.ContentFiles)
	}
}
