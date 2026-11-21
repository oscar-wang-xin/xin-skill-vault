package distribute

import (
	"fmt"
	"os"

	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

type Report struct {
	SkillSlug     string `json:"skillSlug"`
	SkillName     string `json:"skillName"`
	AgentID       string `json:"agentId"`
	TargetName    string `json:"targetName"`
	Mode          string `json:"mode"`
	InstalledPath string `json:"installedPath"`
	Status        string `json:"status"`
	ExpectedHash  string `json:"expectedHash"`
	ActualHash    string `json:"actualHash"`
}

// Check compares every installed copy/link with the current vault content.
func Check(st *store.Store, v *vault.Vault) ([]Report, error) {
	targets, err := st.ListTargets()
	if err != nil {
		return nil, err
	}
	byID := map[string]store.TargetRecord{}
	for _, target := range targets {
		byID[target.ID] = target
	}
	skills, err := st.ListSkills()
	if err != nil {
		return nil, err
	}
	var out []Report
	for _, skill := range skills {
		assignments, err := st.ListAssignments(skill.ID)
		if err != nil {
			return nil, err
		}
		for _, assignment := range assignments {
			target, ok := byID[assignment.TargetID]
			if !ok {
				continue
			}
			report := Report{
				SkillSlug: skill.Slug, SkillName: skill.Name, AgentID: target.AgentID,
				TargetName: target.Name, Mode: assignment.Mode, InstalledPath: assignment.InstalledPath,
				ExpectedHash: skill.ContentHash, ActualHash: "", Status: "ok",
			}
			if info, err := os.Stat(assignment.InstalledPath); err != nil || !info.IsDir() {
				report.Status = "missing"
			} else {
				actual, _, _, err := vault.FingerprintDir(assignment.InstalledPath)
				if err != nil {
					report.Status = "missing"
				} else {
					report.ActualHash = actual
					switch {
					case actual == skill.ContentHash:
						report.Status = "ok"
					case actual == assignment.SourceHash:
						report.Status = "stale"
					default:
						report.Status = "modified"
					}
				}
			}
			if err := st.UpdateAssignmentState(assignment.ID, report.Status, report.ActualHash); err != nil {
				return nil, err
			}
			out = append(out, report)
		}
	}
	return out, nil
}

func Repair(st *store.Store, v *vault.Vault, slug, agentID string) (store.AssignmentRecord, error) {
	record, err := st.FindSkillBySlug(slug)
	if err != nil {
		return store.AssignmentRecord{}, err
	}
	if record == nil {
		return store.AssignmentRecord{}, fmt.Errorf("技能不存在：%s", slug)
	}
	target, err := st.FindTargetByAgent(agentID)
	if err != nil {
		return store.AssignmentRecord{}, err
	}
	if target == nil {
		return store.AssignmentRecord{}, fmt.Errorf("Agent 目标不存在")
	}
	if !target.Detected {
		return store.AssignmentRecord{}, fmt.Errorf("Agent 目录未检测到，无法修复")
	}
	assignment, err := st.FindAssignment(record.ID, target.ID)
	if err != nil {
		return store.AssignmentRecord{}, err
	}
	if assignment == nil {
		return store.AssignmentRecord{}, fmt.Errorf("没有找到分发记录")
	}
	return (&Distributor{Vault: v, Store: st}).Install(*target, slug, assignment.Mode)
}
