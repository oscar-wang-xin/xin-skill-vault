package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"xinskillvault/internal/agent"
	"xinskillvault/internal/distribute"
	projectpkg "xinskillvault/internal/project"
	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

type ProjectAgentItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Short    string `json:"short"`
	Color    string `json:"color"`
	Detected bool   `json:"detected"`
}

type ProjectItem struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Path        string             `json:"path"`
	Description string             `json:"description"`
	Agents      []ProjectAgentItem `json:"agents"`
	SkillCount  int                `json:"skillCount"`
	CreatedAt   int64              `json:"createdAt"`
	UpdatedAt   int64              `json:"updatedAt"`
}

type ProjectSkillAgent struct {
	AgentID       string `json:"agentId"`
	AgentName     string `json:"agentName"`
	Short         string `json:"short"`
	Color         string `json:"color"`
	Mode          string `json:"mode"`
	Status        string `json:"status"`
	InstalledPath string `json:"installedPath"`
}

type ProjectSkill struct {
	Slug        string              `json:"slug"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	ContentHash string              `json:"contentHash"`
	Agents      []ProjectSkillAgent `json:"agents"`
}

type ProjectBulkResult struct {
	Added  int      `json:"added"`
	Errors []string `json:"errors"`
}

func projectAgentItem(adapter agent.Adapter) ProjectAgentItem {
	return ProjectAgentItem{ID: adapter.ID, Name: adapter.Name, Short: adapter.Short, Color: adapter.Color, Detected: adapter.Detected}
}

func (s *Service) projectAgents(agentIDs []string) ([]agent.Adapter, error) {
	byID := map[string]agent.Adapter{}
	for _, adapter := range agent.Defaults() {
		byID[adapter.ID] = adapter
	}
	out := []agent.Adapter{}
	seen := map[string]bool{}
	for _, id := range agentIDs {
		if seen[id] {
			continue
		}
		adapter, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("Agent 不存在：%s", id)
		}
		seen[id] = true
		out = append(out, adapter)
	}
	return out, nil
}

func (s *Service) ensureProjectTargets(record store.ProjectRecord, adapters []agent.Adapter) (map[string]store.TargetRecord, error) {
	out := map[string]store.TargetRecord{}
	for _, adapter := range adapters {
		target := store.TargetRecord{
			ID: projectpkg.TargetID(record.ID, adapter.ID), Kind: "project", AgentID: adapter.ID,
			Name: projectpkg.TargetName(record.Name, adapter.Name), Path: projectpkg.SkillPath(record.Path, adapter.ID),
			ConfigPath: record.Path, Platform: runtime.GOOS, Enabled: true, Detected: true,
			Writable: true, DefaultMode: "link", CreatedAt: record.CreatedAt,
		}
		if err := s.store.UpsertTarget(target); err != nil {
			return nil, err
		}
		if err := s.store.UpdateTargetName(target.ID, target.Name); err != nil {
			return nil, err
		}
		out[adapter.ID] = target
	}
	return out, nil
}

func (s *Service) projectItem(record store.ProjectRecord) (ProjectItem, error) {
	agentIDs, err := s.store.ListProjectAgents(record.ID)
	if err != nil {
		return ProjectItem{}, err
	}
	adapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return ProjectItem{}, err
	}
	targets, err := s.ensureProjectTargets(record, adapters)
	if err != nil {
		return ProjectItem{}, err
	}
	skillIDs := map[string]bool{}
	for _, target := range targets {
		assignments, err := s.store.ListAssignmentsByTarget(target.ID)
		if err != nil {
			return ProjectItem{}, err
		}
		for _, assignment := range assignments {
			skillIDs[assignment.SkillID] = true
		}
	}
	agents := make([]ProjectAgentItem, 0, len(adapters))
	for _, adapter := range adapters {
		agents = append(agents, projectAgentItem(adapter))
	}
	return ProjectItem{
		ID: record.ID, Name: record.Name, Path: record.Path, Description: record.Description,
		Agents: agents, SkillCount: len(skillIDs), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}, nil
}

func (s *Service) ListProjects() ([]ProjectItem, error) {
	records, err := s.store.ListProjects()
	if err != nil {
		return nil, err
	}
	out := make([]ProjectItem, 0, len(records))
	for _, record := range records {
		item, err := s.projectItem(record)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) CreateProject(name, path, description string, agentIDs []string) (ProjectItem, error) {
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	if name == "" || path == "" {
		return ProjectItem{}, fmt.Errorf("项目名称和目录不能为空")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ProjectItem{}, err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		return ProjectItem{}, fmt.Errorf("项目路径不是文件夹")
	}
	if err != nil && !os.IsNotExist(err) {
		return ProjectItem{}, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return ProjectItem{}, fmt.Errorf("create project directory: %w", err)
	}
	adapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return ProjectItem{}, err
	}
	now := time.Now().Unix()
	record := store.ProjectRecord{
		ID: newProjectID(), Name: name, Path: abs, Description: description, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.store.CreateProject(record); err != nil {
		return ProjectItem{}, err
	}
	ids := make([]string, 0, len(adapters))
	for _, adapter := range adapters {
		ids = append(ids, adapter.ID)
	}
	if err := s.store.SetProjectAgents(record.ID, ids, now); err != nil {
		return ProjectItem{}, err
	}
	if _, err := s.ensureProjectTargets(record, adapters); err != nil {
		return ProjectItem{}, err
	}
	return s.projectItem(record)
}

func (s *Service) UpdateProject(id, name, description string) (ProjectItem, error) {
	record, err := s.store.FindProject(id)
	if err != nil {
		return ProjectItem{}, err
	}
	if record == nil {
		return ProjectItem{}, fmt.Errorf("项目不存在")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ProjectItem{}, fmt.Errorf("项目名称不能为空")
	}
	record.Name = name
	record.Description = description
	record.UpdatedAt = time.Now().Unix()
	if err := s.store.UpdateProject(*record); err != nil {
		return ProjectItem{}, err
	}
	agentIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return ProjectItem{}, err
	}
	adapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return ProjectItem{}, err
	}
	if _, err := s.ensureProjectTargets(*record, adapters); err != nil {
		return ProjectItem{}, err
	}
	return s.projectItem(*record)
}

func (s *Service) SetProjectAgents(id string, agentIDs []string) (ProjectItem, error) {
	record, err := s.store.FindProject(id)
	if err != nil {
		return ProjectItem{}, err
	}
	if record == nil {
		return ProjectItem{}, fmt.Errorf("项目不存在")
	}
	oldIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return ProjectItem{}, err
	}
	oldAdapters, _ := s.projectAgents(oldIDs)
	newAdapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return ProjectItem{}, err
	}
	keep := map[string]bool{}
	for _, adapter := range newAdapters {
		keep[adapter.ID] = true
	}
	distributor := &distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}
	for _, adapter := range oldAdapters {
		if keep[adapter.ID] {
			continue
		}
		if err := distributor.RemoveTarget(projectpkg.TargetID(id, adapter.ID)); err != nil {
			return ProjectItem{}, err
		}
	}
	ids := make([]string, 0, len(newAdapters))
	for _, adapter := range newAdapters {
		ids = append(ids, adapter.ID)
	}
	if err := s.store.SetProjectAgents(id, ids, time.Now().Unix()); err != nil {
		return ProjectItem{}, err
	}
	if _, err := s.ensureProjectTargets(*record, newAdapters); err != nil {
		return ProjectItem{}, err
	}
	return s.projectItem(*record)
}

func (s *Service) DeleteProject(id string) error {
	record, err := s.store.FindProject(id)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("项目不存在")
	}
	agentIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return err
	}
	adapters, _ := s.projectAgents(agentIDs)
	distributor := &distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}
	for _, adapter := range adapters {
		if err := distributor.RemoveTarget(projectpkg.TargetID(id, adapter.ID)); err != nil {
			return err
		}
	}
	return s.store.DeleteProject(id)
}

func (s *Service) ListProjectSkills(id string) ([]ProjectSkill, error) {
	record, err := s.store.FindProject(id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("项目不存在")
	}
	agentIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return nil, err
	}
	adapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return nil, err
	}
	targets, err := s.ensureProjectTargets(*record, adapters)
	if err != nil {
		return nil, err
	}
	adapterByID := map[string]agent.Adapter{}
	for _, adapter := range adapters {
		adapterByID[adapter.ID] = adapter
	}
	bySlug := map[string]*ProjectSkill{}
	for _, adapter := range adapters {
		target := targets[adapter.ID]
		assignments, err := s.store.ListAssignmentsByTarget(target.ID)
		if err != nil {
			return nil, err
		}
		for _, assignment := range assignments {
			skillRecord, err := s.store.FindSkillByID(assignment.SkillID)
			if err != nil || skillRecord == nil {
				continue
			}
			item := bySlug[skillRecord.Slug]
			if item == nil {
				item = &ProjectSkill{
					Slug: skillRecord.Slug, Name: skillRecord.Name, Description: skillRecord.Description,
					Version: skillRecord.Version, ContentHash: skillRecord.ContentHash,
				}
				bySlug[skillRecord.Slug] = item
			}
			targetAgent := adapterByID[adapter.ID]
			item.Agents = append(item.Agents, ProjectSkillAgent{
				AgentID: targetAgent.ID, AgentName: targetAgent.Name, Short: targetAgent.Short, Color: targetAgent.Color,
				Mode: assignment.Mode, Status: assignment.Status, InstalledPath: assignment.InstalledPath,
			})
		}
	}
	out := make([]ProjectSkill, 0, len(bySlug))
	for _, item := range bySlug {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) AddProjectSkills(id string, slugs []string, mode string) (ProjectBulkResult, error) {
	result := ProjectBulkResult{Errors: []string{}}
	record, err := s.store.FindProject(id)
	if err != nil {
		return result, err
	}
	if record == nil {
		return result, fmt.Errorf("项目不存在")
	}
	agentIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return result, err
	}
	adapters, err := s.projectAgents(agentIDs)
	if err != nil {
		return result, err
	}
	if len(adapters) == 0 {
		return result, fmt.Errorf("项目还没有配置 Agent")
	}
	targets, err := s.ensureProjectTargets(*record, adapters)
	if err != nil {
		return result, err
	}
	distributor := &distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}
	for _, slug := range slugs {
		skillRecord, err := s.store.FindSkillBySlug(slug)
		if err != nil || skillRecord == nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 技能不存在", slug))
			continue
		}
		for _, adapter := range adapters {
			target := targets[adapter.ID]
			existing, err := s.store.FindAssignment(skillRecord.ID, target.ID)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s / %s: %v", slug, adapter.Name, err))
				continue
			}
			if existing != nil {
				continue
			}
			if _, err := distributor.Install(target, slug, mode); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s / %s: %v", slug, adapter.Name, err))
				continue
			}
			result.Added++
		}
	}
	return result, nil
}

func (s *Service) RemoveProjectSkill(id, slug string) error {
	record, err := s.store.FindProject(id)
	if err != nil || record == nil {
		return fmt.Errorf("项目不存在")
	}
	skillRecord, err := s.store.FindSkillBySlug(slug)
	if err != nil || skillRecord == nil {
		return fmt.Errorf("技能不存在")
	}
	agentIDs, err := s.store.ListProjectAgents(id)
	if err != nil {
		return err
	}
	adapters, _ := s.projectAgents(agentIDs)
	distributor := &distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}
	for _, adapter := range adapters {
		targetID := projectpkg.TargetID(id, adapter.ID)
		existing, err := s.store.FindAssignment(skillRecord.ID, targetID)
		if err != nil {
			return err
		}
		if existing == nil {
			continue
		}
		if err := distributor.Uninstall(skillRecord.ID, targetID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) PickProjectDirectory() (string, error) {
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	path, err := hostApp.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		SetTitle("选择项目目录").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("选择项目目录失败：%w", err)
	}
	return path, nil
}

func (s *Service) OpenProjectFolder(id string) error {
	record, err := s.store.FindProject(id)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("项目不存在")
	}
	return openFolder(record.Path)
}

func newProjectID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pr_%d", time.Now().UnixNano())
	}
	return "pr_" + hex.EncodeToString(buf)
}
