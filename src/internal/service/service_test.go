package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
	"github.com/xin-skill-vault/xin-skill-vault/internal/vault"
)

func newTestService(t *testing.T) (*AppService, string) {
	t.Helper()
	base := t.TempDir()
	st, err := store.Open(filepath.Join(base, "data", "vault.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.SeedSources([]store.Source{{ID: "local", Name: "本地仓库", Kind: "local", Builtin: true}}); err != nil {
		t.Fatal(err)
	}
	agentSkills := filepath.Join(base, "agent", "skills")
	if err := st.SeedAgents([]store.Agent{{
		ID: "test", Name: "Test Agent", SkillsPath: agentSkills,
		ConfigPath: filepath.Join(base, "agent"), Enabled: true,
	}}); err != nil {
		t.Fatal(err)
	}
	defaults := map[string]string{
		SettingSkin: "cold-white", SettingLayout: "table", SettingLanguage: "zh-CN",
		SettingCheckOnStart: "true", SettingAutoUpdateLinks: "false", SettingCloseToTray: "true",
	}
	mode := "symlink"
	if runtime.GOOS == "windows" {
		mode = "junction"
	}
	defaults[SettingDefaultMode] = mode
	for k, v := range defaults {
		if err := st.SetSetting(k, v); err != nil {
			t.Fatal(err)
		}
	}
	m := vault.New(filepath.Join(base, "vault"), st)
	if err := m.Ensure(); err != nil {
		t.Fatal(err)
	}
	svc := &AppService{Store: st, Vault: m, DataDir: filepath.Join(base, "data"), DBPath: st.Path, Version: "test"}
	return svc, base
}

func writeTestSkill(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: A test skill for the service layer.\nversion: 1.0.0\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "references", "guide.md"), []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServiceImportDistributeSnapshot(t *testing.T) {
	svc, base := newTestService(t)
	source := filepath.Join(base, "source", "review-kit")
	writeTestSkill(t, source, "review-kit")

	result, err := svc.Import(ImportRequest{
		Path: source, AgentIDs: []string{"test"},
		Mode: currentPlatformMode(),
	})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(result.Snapshot.Skills) != 1 {
		t.Fatalf("skills = %d, want 1", len(result.Snapshot.Skills))
	}
	skill := result.Snapshot.Skills[0]
	if skill.Name != "review-kit" || len(skill.Agents) != 1 || skill.Agents[0] != "test" {
		t.Fatalf("unexpected skill view: %+v", skill)
	}
	if result.Snapshot.Stats.Links+result.Snapshot.Stats.Copies != 1 {
		t.Fatalf("distribution stats = %+v", result.Snapshot.Stats)
	}
	if result.Snapshot.Vault.SQLiteVersion == "" {
		t.Fatalf("sqlite version missing")
	}

	// Toggle distribution off and back on through the matrix endpoint.
	off, err := svc.SetDistribution(skill.ID, "test", false)
	if err != nil {
		t.Fatalf("SetDistribution off: %v", err)
	}
	if len(off.Snapshot.Distributions) != 0 {
		t.Fatalf("distribution still present: %+v", off.Snapshot.Distributions)
	}
	on, err := svc.SetDistribution(skill.ID, "test", true)
	if err != nil {
		t.Fatalf("SetDistribution on: %v", err)
	}
	if len(on.Snapshot.Distributions) != 1 {
		t.Fatalf("distribution not recreated")
	}

	files, err := svc.SkillFiles(skill.ID)
	if err != nil || len(files) != 2 {
		t.Fatalf("SkillFiles = %d, err=%v", len(files), err)
	}

	removed, err := svc.RemoveSkill(skill.ID)
	if err != nil {
		t.Fatalf("RemoveSkill: %v", err)
	}
	if len(removed.Snapshot.Skills) != 0 {
		t.Fatalf("skill still present")
	}
}

func TestServiceSettingsAndSkins(t *testing.T) {
	svc, _ := newTestService(t)
	snap, err := svc.SetSkin("mech-yellow")
	if err != nil {
		t.Fatal(err)
	}
	if snap.Settings[SettingSkin] != "mech-yellow" {
		t.Fatalf("skin = %q", snap.Settings[SettingSkin])
	}
	if _, err := svc.SetSkin("not-a-skin"); err == nil {
		t.Fatalf("invalid skin accepted")
	}
	if _, err := svc.SaveSetting(SettingLayout, "cards"); err != nil {
		t.Fatal(err)
	}
	snap, _ = svc.Snapshot()
	if snap.Settings[SettingLayout] != "cards" {
		t.Fatalf("layout = %q", snap.Settings[SettingLayout])
	}
	if _, err := svc.SaveSetting("unknown.key", "x"); err == nil {
		t.Fatalf("unknown setting accepted")
	}
	if _, err := svc.SaveSetting(SettingFontFamily, "yahei"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSetting(SettingFontScale, "1.28"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSetting(SettingFontFamily, "comic-sans"); err == nil {
		t.Fatalf("invalid font accepted")
	}
	if _, err := svc.SaveSetting(SettingFontScale, "3"); err == nil {
		t.Fatalf("invalid font scale accepted")
	}
	if _, err := svc.SaveSetting(SettingCloseAction, "tray"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSetting(SettingCloseAction, "explode"); err == nil {
		t.Fatalf("invalid close action accepted")
	}
	if err := svc.ResolveClose("explode", false); err == nil {
		t.Fatalf("invalid ResolveClose action accepted")
	}
	snap, _ = svc.Snapshot()
	if snap.Settings[SettingFontFamily] != "yahei" || snap.Settings[SettingFontScale] != "1.28" {
		t.Fatalf("font settings not persisted: %+v", snap.Settings)
	}
	if snap.Settings[SettingCloseAction] != "tray" {
		t.Fatalf("close action not persisted: %+v", snap.Settings)
	}
}

func TestServiceContentLabels(t *testing.T) {
	svc, base := newTestService(t)
	source := filepath.Join(base, "source", "scripted")
	writeTestSkill(t, source, "scripted")
	if err := os.WriteFile(filepath.Join(source, "run.ps1"), []byte("Write-Host 'hello'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := svc.Import(ImportRequest{Path: source})
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("content labels should not produce warnings: %v", result.Warnings)
	}
	var scripted *SkillView
	for i := range result.Snapshot.Skills {
		if result.Snapshot.Skills[i].ID == "scripted" {
			scripted = &result.Snapshot.Skills[i]
			break
		}
	}
	if scripted == nil || scripted.Content != "script" || len(scripted.ContentFiles) == 0 {
		t.Fatalf("content labels missing from snapshot: %+v", scripted)
	}
	if !strings.Contains(strings.Join(scripted.ContentFiles, "\n"), "run.ps1（脚本）") {
		t.Fatalf("expected run.ps1 in content files: %v", scripted.ContentFiles)
	}

	if _, err := svc.SaveSetting("security.blockHigh", "true"); err == nil {
		t.Fatalf("removed high-risk blocking setting should not be accepted")
	}

	binary := filepath.Join(base, "source", "binary")
	writeTestSkill(t, binary, "binary")
	if err := os.WriteFile(filepath.Join(binary, "tool.exe"), []byte("MZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := svc.Import(ImportRequest{Path: binary})
	if err != nil {
		t.Fatal(err)
	}
	for _, skill := range second.Snapshot.Skills {
		if skill.ID == "binary" && skill.Content != "executable" {
			t.Fatalf("content = %q", skill.Content)
		}
	}
}

func TestServiceChangeVaultPath(t *testing.T) {
	svc, base := newTestService(t)
	source := filepath.Join(base, "source", "move-skill")
	writeTestSkill(t, source, "move-skill")
	if _, err := svc.Import(ImportRequest{Path: source}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Distribute(DistributeRequest{SkillID: "move-skill", AgentIDs: []string{"test"}, Mode: vault.LinkMode()}); err != nil {
		t.Fatal(err)
	}
	oldSkills := svc.Vault.SkillsDir()
	oldRoot := svc.Vault.Root
	newRoot := filepath.Join(base, "new-vault")
	moved, err := svc.ChangeVaultPath(newRoot, "move")
	if err != nil {
		t.Fatalf("move vault: %v", err)
	}
	if svc.Vault.Root != newRoot || moved.Snapshot.Vault.Path != newRoot {
		t.Fatalf("vault root not switched: %s / %s", svc.Vault.Root, moved.Snapshot.Vault.Path)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "skills", "move-skill", "SKILL.md")); err != nil {
		t.Fatalf("moved skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(oldSkills, "move-skill")); !os.IsNotExist(err) {
		t.Fatalf("old skill still exists after move")
	}
	if got, ok := svc.Store.GetSetting(SettingVaultPath); !ok || got != newRoot {
		t.Fatalf("vault path setting = %q, %v", got, ok)
	}
	dist, err := svc.Store.GetDistribution("move-skill", "test")
	if err != nil {
		t.Fatal(err)
	}
	if !vault.SameLocation(dist.LinkPath, filepath.Join(newRoot, "skills", "move-skill")) {
		t.Fatalf("distribution not relinked: %s", dist.LinkPath)
	}

	copyRoot := filepath.Join(base, "copy-vault")
	copied, err := svc.ChangeVaultPath(copyRoot, "copy")
	if err != nil {
		t.Fatalf("copy vault: %v", err)
	}
	if copied.Snapshot.Vault.Path != copyRoot {
		t.Fatalf("copy vault root = %s", copied.Snapshot.Vault.Path)
	}
	if _, err := os.Stat(filepath.Join(newRoot, "skills", "move-skill", "SKILL.md")); err != nil {
		t.Fatalf("copy should leave the old vault intact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(copyRoot, "skills", "move-skill", "SKILL.md")); err != nil {
		t.Fatalf("copied skill missing: %v", err)
	}
	if oldRoot == copyRoot {
		t.Fatalf("unexpected roots")
	}
	notEmpty := filepath.Join(base, "not-empty")
	if err := os.MkdirAll(notEmpty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notEmpty, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ChangeVaultPath(notEmpty, "move"); err == nil {
		t.Fatalf("non-empty target should be rejected")
	}
}

func TestServiceImportMany(t *testing.T) {
	svc, base := newTestService(t)
	first := filepath.Join(base, "source", "one")
	second := filepath.Join(base, "source", "two")
	writeTestSkill(t, first, "one")
	writeTestSkill(t, second, "two")
	result, err := svc.ImportMany(ImportManyRequest{
		Items: []ImportItem{
			{Path: first, SourceID: "local", SourceRef: first},
			{Path: second, SourceID: "local", SourceRef: second},
		},
		AgentIDs: []string{"test"},
		Mode:     currentPlatformMode(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Snapshot.Skills) != 2 {
		t.Fatalf("skills = %d, want 2", len(result.Snapshot.Skills))
	}
	if len(result.Snapshot.Distributions) != 2 {
		t.Fatalf("distributions = %d, want 2", len(result.Snapshot.Distributions))
	}
}

func TestServiceGroupsMetadataAndSources(t *testing.T) {
	svc, base := newTestService(t)
	source := filepath.Join(base, "source", "meta")
	writeTestSkill(t, source, "meta")
	result, err := svc.Import(ImportRequest{Path: source})
	if err != nil {
		t.Fatal(err)
	}
	skill := result.Snapshot.Skills[0]

	snap, err := svc.CreateGroup("设计系统")
	if err != nil {
		t.Fatal(err)
	}
	group := snap.Groups[0]
	snap, err = svc.UpdateSkillMeta(SkillMetaUpdate{
		SkillID: skill.ID, Notes: "发布前跑 finish gate", GroupID: group.ID,
		Tags: []string{"ui", "review", "UI"},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated := snap.Skills[0]
	if updated.Notes != "发布前跑 finish gate" || updated.GroupName != "设计系统" {
		t.Fatalf("metadata not saved: %+v", updated)
	}
	if len(updated.Tags) != 2 {
		t.Fatalf("tags not deduped: %v", updated.Tags)
	}
	if _, err := svc.RenameGroup(group.ID, "设计规范"); err != nil {
		t.Fatal(err)
	}
	snap, _ = svc.Snapshot()
	if snap.Groups[0].Name != "设计规范" {
		t.Fatalf("group rename failed")
	}
	if _, err := svc.DeleteGroup(group.ID); err != nil {
		t.Fatal(err)
	}

	added, err := svc.AddSource(SourceInput{Name: "团队规范", Kind: "git", URL: "github.com/team/skills"})
	if err != nil {
		t.Fatal(err)
	}
	if len(added.Snapshot.Sources) != 2 {
		t.Fatalf("sources = %d, want 2", len(added.Snapshot.Sources))
	}
	custom := added.Snapshot.Sources[len(added.Snapshot.Sources)-1]
	if _, err := svc.RemoveSource(custom.ID); err != nil {
		t.Fatal(err)
	}
}

func TestServiceProjectAndDirectory(t *testing.T) {
	svc, base := newTestService(t)
	source := filepath.Join(base, "source", "project-skill")
	writeTestSkill(t, source, "project-skill")
	imported, err := svc.Import(ImportRequest{Path: source})
	if err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(base, "project", ".agents", "skills")
	added, err := svc.AddProject(ProjectInput{Name: "Atlas", SkillsPath: projectPath})
	if err != nil {
		t.Fatal(err)
	}
	var projectID string
	for _, agent := range added.Snapshot.Agents {
		if agent.Kind == "project" {
			projectID = agent.ID
		}
	}
	if projectID == "" {
		t.Fatalf("project target not created")
	}
	if _, err := svc.SetDistribution(imported.Snapshot.Skills[0].ID, projectID, true); err != nil {
		t.Fatal(err)
	}
	view, err := svc.DirectorySkills(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Entries) != 1 || !view.Entries[0].Managed {
		t.Fatalf("directory view = %+v", view)
	}
	if _, err := svc.RemoveDirectoryEntry(projectID, view.Entries[0].Name); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RemoveProject(projectID); err != nil {
		t.Fatal(err)
	}
}

func currentPlatformMode() string {
	if runtime.GOOS == "windows" {
		return "junction"
	}
	return "symlink"
}
