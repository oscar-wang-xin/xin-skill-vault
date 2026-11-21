package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xinskillvault/internal/agent"
	"xinskillvault/internal/skill"
)

type AgentSkillItem struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	Mode          string `json:"mode"`
	Status        string `json:"status"`
	InstalledPath string `json:"installedPath"`
	Managed       bool   `json:"managed"`
}

func (s *Service) ListAgentSkills(agentID string) ([]AgentSkillItem, error) {
	var adapter *agent.Adapter
	for _, item := range agent.Defaults() {
		if item.ID == agentID {
			adapter = &item
			break
		}
	}
	if adapter == nil || !adapter.Detected {
		return nil, fmt.Errorf("Agent 不存在或未检测到")
	}
	targets, err := s.ensureTargets()
	if err != nil {
		return nil, err
	}
	target, ok := targets[agentID]
	if !ok {
		return nil, fmt.Errorf("Agent 目标不存在")
	}
	assignments, err := s.store.ListAssignmentsByTarget(target.ID)
	if err != nil {
		return nil, err
	}
	managed := map[string]bool{}
	out := []AgentSkillItem{}
	for _, assignment := range assignments {
		record, err := s.store.FindSkillByID(assignment.SkillID)
		if err != nil || record == nil {
			continue
		}
		managed[record.Slug] = true
		out = append(out, AgentSkillItem{
			Slug: record.Slug, Name: record.Name, Description: record.Description,
			Version: record.Version, Mode: assignment.Mode, Status: assignment.Status,
			InstalledPath: assignment.InstalledPath, Managed: true,
		})
	}
	entries, err := os.ReadDir(adapter.Path)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" {
				continue
			}
			dir := filepath.Join(adapter.Path, name)
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			parsed, err := skill.ParseDir(dir)
			if err != nil || managed[parsed.Slug] {
				continue
			}
			managed[parsed.Slug] = true
			out = append(out, AgentSkillItem{
				Slug: parsed.Slug, Name: parsed.Meta.Name, Description: parsed.Meta.Description,
				Version: parsed.Meta.Version, Status: "unmanaged", InstalledPath: dir, Managed: false,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
