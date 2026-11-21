package project

import (
	"path/filepath"
	"strings"
)

// SkillPath returns the project-level skill directory for a supported Agent.
func SkillPath(projectRoot, agentID string) string {
	root := filepath.Clean(projectRoot)
	switch agentID {
	case "claude-code":
		return filepath.Join(root, ".claude", "skills")
	case "cursor":
		return filepath.Join(root, ".cursor", "skills")
	case "opencode":
		return filepath.Join(root, ".opencode", "skills")
	case "codex":
		return filepath.Join(root, ".codex", "skills")
	case "alma":
		return filepath.Join(root, ".alma", "skills")
	case "cherry-studio":
		return filepath.Join(root, ".cherry-studio", "skills")
	case "codebuddy":
		return filepath.Join(root, ".codebuddy", "skills")
	case "workbuddy":
		return filepath.Join(root, ".workbuddy", "skills")
	default:
		return filepath.Join(root, ".agents", "skills")
	}
}

func TargetID(projectID, agentID string) string {
	return "project:" + projectID + ":" + agentID
}

func TargetName(projectName, agentName string) string {
	name := strings.TrimSpace(projectName)
	if name == "" {
		name = "Project"
	}
	return name + " / " + agentName
}
