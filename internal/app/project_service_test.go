package app

import (
	"os"
	"path/filepath"
	"testing"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
)

func TestProjectLifecycleAndSkills(t *testing.T) {
	root := t.TempDir()
	manager, err := config.NewManager(filepath.Join(root, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
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
	service := NewService(manager, st, nil)

	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "SKILL.md"), []byte("---\nname: Project Skill\ndescription: project test\n---\n\n# Project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ImportSkillDirectory(source); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(t.TempDir(), "demo-project")
	project, err := service.CreateProject("Demo Project", projectDir, "test", []string{"claude-code"})
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Agents) != 1 || project.Agents[0].ID != "claude-code" {
		t.Fatalf("project agents = %+v", project.Agents)
	}
	result, err := service.AddProjectSkills(project.ID, []string{"project-skill"}, "copy")
	if err != nil || result.Added != 1 {
		t.Fatalf("add project skills = %+v, %v", result, err)
	}
	installed := filepath.Join(projectDir, ".claude", "skills", "project-skill", "SKILL.md")
	if _, err := os.Stat(installed); err != nil {
		t.Fatalf("project install missing: %v", err)
	}
	projectSkills, err := service.ListProjectSkills(project.ID)
	if err != nil || len(projectSkills) != 1 || len(projectSkills[0].Agents) != 1 {
		t.Fatalf("project skills = %+v, %v", projectSkills, err)
	}
	if _, err := service.SetProjectAgents(project.ID, []string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "skills", "project-skill")); !os.IsNotExist(err) {
		t.Fatalf("project install should be removed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteProject(project.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "keep.txt")); err != nil {
		t.Fatalf("project directory should remain: %v", err)
	}
}
