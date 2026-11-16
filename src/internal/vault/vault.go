package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
)

type Manager struct {
	Root  string
	Store *store.Store
}

type ScanResult struct {
	Scanned int    `json:"scanned"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	At      string `json:"at"`
}

type Importable struct {
	Path             string   `json:"path"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Slug             string   `json:"slug"`
	Content          string   `json:"content"`
	ContentFiles     []string `json:"contentFiles"`
	SizeBytes        int64    `json:"sizeBytes"`
	FileCount        int      `json:"fileCount"`
	Agents           []string `json:"agents"`
	AlreadyInstalled bool     `json:"alreadyInstalled"`
	IsLink           bool     `json:"isLink"`
	SourceID         string   `json:"sourceId"`
	SourceRef        string   `json:"sourceRef"`
}

func New(root string, st *store.Store) *Manager {
	return &Manager{Root: root, Store: st}
}

func (m *Manager) Ensure() error {
	for _, dir := range []string{m.SkillsDir(), m.TrashDir(), filepath.Join(m.Root, ".cache")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建仓库目录失败 %s: %w", dir, err)
		}
	}
	return nil
}

func (m *Manager) SkillsDir() string { return filepath.Join(m.Root, "skills") }
func (m *Manager) TrashDir() string  { return filepath.Join(m.Root, ".trash") }

func (m *Manager) SkillPath(slug string) string { return filepath.Join(m.SkillsDir(), slug) }

// Scan reconciles the SQLite index with the directories on disk. SKILL.md is
// the source of truth for name/description; the database is a rebuildable
// index plus distribution state.
func (m *Manager) Scan() (ScanResult, error) {
	result := ScanResult{At: time.Now().Format(time.RFC3339)}
	entries, err := os.ReadDir(m.SkillsDir())
	if err != nil {
		return result, fmt.Errorf("读取仓库技能目录失败: %w", err)
	}
	keep := map[string]bool{}
	for _, entry := range entries {
		if SkipName(entry.Name()) || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		dir := filepath.Join(m.SkillsDir(), entry.Name())
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, ok := FindSkillFile(dir); !ok {
			continue
		}
		insp, err := Inspect(dir)
		if err != nil {
			continue
		}
		hash, err := HashDir(dir)
		if err != nil {
			continue
		}
		existing, _ := m.Store.GetSkill(insp.Slug)
		if existing.ID == "" {
			result.Added++
		}
		sourceID := existing.SourceID
		if sourceID == "" {
			sourceID = "local"
		}
		skill := store.Skill{
			ID:           insp.Slug,
			Name:         insp.Name,
			Description:  insp.Description,
			SourceID:     sourceID,
			SourceRef:    existing.SourceRef,
			VaultPath:    dir,
			ContentHash:  hash,
			SizeBytes:    insp.SizeBytes,
			FileCount:    insp.FileCount,
			Content:      insp.Content,
			ContentFiles: insp.ContentFiles,
			Version:      insp.Version,
			InstalledAt:  firstNonEmpty(existing.InstalledAt, result.At),
			UpdatedAt:    result.At,
		}
		if err := m.Store.UpsertSkill(skill); err != nil {
			return result, err
		}
		keep[insp.Slug] = true
		result.Scanned++
	}
	if err := m.Store.PruneSkills(keep); err != nil {
		return result, err
	}
	_ = m.Store.SetSetting("vault.lastScanAt", result.At)
	if err := m.CheckHealth(); err != nil {
		return result, err
	}
	return result, nil
}

// CheckHealth verifies every distribution on disk and records broken links or
// drifted copies. It never modifies skill content.
func (m *Manager) CheckHealth() error {
	dists, err := m.Store.ListDistributions()
	if err != nil {
		return err
	}
	for _, d := range dists {
		skill, err := m.Store.GetSkill(d.SkillID)
		if err != nil {
			continue
		}
		status := "ok"
		hash := ""
		info, statErr := os.Lstat(d.LinkPath)
		if statErr != nil {
			status = "broken"
		} else if d.Mode == "copy" {
			if !isRealDir(d.LinkPath) {
				status = "broken"
			} else {
				hash, _ = HashDir(d.LinkPath)
				if hash != skill.ContentHash {
					status = "drifted"
				}
			}
		} else {
			if !IsLink(d.LinkPath) || !SameLocation(d.LinkPath, skill.VaultPath) {
				status = "broken"
			}
		}
		if info == nil && statErr == nil {
			status = "broken"
		}
		if err := m.Store.UpdateDistributionStatus(d.SkillID, d.AgentID, status, hash); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Distribute(skillID string, agentIDs []string, mode string) ([]string, error) {
	skill, err := m.Store.GetSkill(skillID)
	if err != nil {
		return nil, fmt.Errorf("技能不存在: %s", skillID)
	}
	if info, err := os.Stat(skill.VaultPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("技能目录已丢失: %s", skill.VaultPath)
	}
	if mode == "" {
		mode = defaultLinkMode()
	}
	if mode != "copy" && mode != "link" && mode != LinkMode() {
		return nil, fmt.Errorf("不支持的安装方式: %s", mode)
	}
	var warnings []string
	for _, agentID := range agentIDs {
		agent, err := m.Store.GetAgent(agentID)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("未知 Agent：%s", agentID))
			continue
		}
		if err := os.MkdirAll(agent.SkillsPath, 0o755); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s：无法创建技能目录（%v）", agent.Name, err))
			continue
		}
		_ = m.Store.MarkAgentDetected(agent.ID, true)
		linkPath := filepath.Join(agent.SkillsPath, skill.ID)
		if _, err := os.Lstat(linkPath); err == nil {
			if mode != "copy" && IsLink(linkPath) && SameLocation(linkPath, skill.VaultPath) {
				_ = m.Store.UpsertDistribution(store.Distribution{
					SkillID: skill.ID, AgentID: agent.ID, Mode: LinkMode(),
					LinkPath: linkPath, Status: "ok", ContentHash: skill.ContentHash,
				})
				continue
			}
			warnings = append(warnings, fmt.Sprintf("%s：目标已存在，未覆盖 %s", agent.Name, linkPath))
			continue
		}
		hash := skill.ContentHash
		if mode == "copy" {
			if err := CopyDir(skill.VaultPath, linkPath); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s：复制失败（%v）", agent.Name, err))
				continue
			}
			hash, _ = HashDir(linkPath)
		} else {
			if err := CreateDirLink(linkPath, skill.VaultPath); err != nil {
				warnings = append(warnings, fmt.Sprintf("%s：%v", agent.Name, err))
				continue
			}
		}
		actualMode := LinkMode()
		if mode == "copy" {
			actualMode = "copy"
		}
		if err := m.Store.UpsertDistribution(store.Distribution{
			SkillID: skill.ID, AgentID: agent.ID, Mode: actualMode,
			LinkPath: linkPath, Status: "ok", ContentHash: hash,
		}); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s：写入索引失败（%v）", agent.Name, err))
		}
	}
	m.Store.AddActivity("distribute", skill.Name, fmt.Sprintf("%d 个 Agent · %s", len(agentIDs), mode))
	return warnings, nil
}

func (m *Manager) RemoveDistribution(skillID, agentID string) error {
	d, err := m.Store.GetDistribution(skillID, agentID)
	if err != nil {
		return fmt.Errorf("该分发不存在")
	}
	agent, err := m.Store.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("Agent 不存在")
	}
	if !isWithin(agent.SkillsPath, d.LinkPath) || filepath.Clean(agent.SkillsPath) == filepath.Clean(d.LinkPath) {
		return fmt.Errorf("拒绝删除技能目录之外的路径: %s", d.LinkPath)
	}
	if _, err := os.Lstat(d.LinkPath); err == nil {
		if IsLink(d.LinkPath) {
			if err := RemoveLink(d.LinkPath); err != nil {
				return fmt.Errorf("删除链接失败: %w", err)
			}
		} else {
			if err := os.RemoveAll(d.LinkPath); err != nil {
				return fmt.Errorf("删除副本失败: %w", err)
			}
		}
	}
	if err := m.Store.DeleteDistribution(skillID, agentID); err != nil {
		return err
	}
	m.Store.AddActivity("unlink", skillID, agent.Name)
	return nil
}

func (m *Manager) Import(srcPath string) (store.Skill, error) {
	return m.ImportWithSource(srcPath, "local", "")
}

func (m *Manager) ImportWithSource(srcPath, sourceID, sourceRef string) (store.Skill, error) {
	abs, err := filepath.Abs(srcPath)
	if err != nil {
		return store.Skill{}, err
	}
	source := ResolveSourcePath(abs)
	if isWithin(m.SkillsDir(), source) {
		return store.Skill{}, fmt.Errorf("该目录已经在仓库中")
	}
	insp, err := Inspect(source)
	if err != nil {
		return store.Skill{}, err
	}
	target := m.SkillPath(insp.Slug)
	if _, err := os.Stat(target); err == nil {
		return store.Skill{}, fmt.Errorf("仓库中已存在同名技能「%s」，请先移除或重命名", insp.Slug)
	}
	if err := CopyDir(source, target); err != nil {
		_ = os.RemoveAll(target)
		return store.Skill{}, fmt.Errorf("复制技能失败: %w", err)
	}
	installed, err := Inspect(target)
	if err != nil {
		_ = os.RemoveAll(target)
		return store.Skill{}, err
	}
	hash, err := HashDir(target)
	if err != nil {
		return store.Skill{}, err
	}
	if sourceID == "" {
		sourceID = "local"
	}
	if sourceRef == "" {
		sourceRef = filepath.ToSlash(source)
	}
	nowValue := time.Now().Format(time.RFC3339)
	skill := store.Skill{
		ID:           installed.Slug,
		Name:         installed.Name,
		Description:  installed.Description,
		SourceID:     sourceID,
		SourceRef:    sourceRef,
		VaultPath:    target,
		ContentHash:  hash,
		SizeBytes:    installed.SizeBytes,
		FileCount:    installed.FileCount,
		Content:      installed.Content,
		ContentFiles: installed.ContentFiles,
		Version:      installed.Version,
		InstalledAt:  nowValue,
		UpdatedAt:    nowValue,
	}
	if err := m.Store.UpsertSkill(skill); err != nil {
		return store.Skill{}, err
	}
	m.Store.AddActivity("import", skill.Name, sourceRef)
	return skill, nil
}

func (m *Manager) RemoveSkill(skillID string) error {
	skill, err := m.Store.GetSkill(skillID)
	if err != nil {
		return fmt.Errorf("技能不存在: %s", skillID)
	}
	dists, _ := m.Store.ListDistributions()
	for _, d := range dists {
		if d.SkillID != skillID {
			continue
		}
		if err := m.RemoveDistribution(skillID, d.AgentID); err != nil {
			// Keep removing the vault copy; the UI reports leftover links on
			// the next health check.
			continue
		}
	}
	if _, err := os.Stat(skill.VaultPath); err == nil {
		stamp := time.Now().Format("20060102-150405")
		trash := filepath.Join(m.TrashDir(), fmt.Sprintf("%s-%s", skillID, stamp))
		if err := os.MkdirAll(m.TrashDir(), 0o755); err != nil {
			return err
		}
		if err := os.Rename(skill.VaultPath, trash); err != nil {
			if err := os.RemoveAll(skill.VaultPath); err != nil {
				return fmt.Errorf("删除技能目录失败: %w", err)
			}
		}
	}
	if err := m.Store.DeleteSkill(skillID); err != nil {
		return err
	}
	m.Store.AddActivity("remove", skill.Name, skill.VaultPath)
	return nil
}

func (m *Manager) Importables() ([]Importable, error) {
	agents, err := m.Store.ListAgents()
	if err != nil {
		return nil, err
	}
	existing, _ := m.Store.ListSkills()
	installed := map[string]bool{}
	for _, s := range existing {
		installed[s.ID] = true
	}
	seen := map[string]int{}
	var out []Importable
	for _, agent := range agents {
		entries, err := os.ReadDir(agent.SkillsPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if SkipName(entry.Name()) || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			dir := filepath.Join(agent.SkillsPath, entry.Name())
			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				continue
			}
			link := IsLink(dir)
			if link {
				vaultOwned := false
				for _, existingSkill := range existing {
					if SameLocation(dir, existingSkill.VaultPath) {
						vaultOwned = true
						break
					}
				}
				if vaultOwned {
					continue
				}
			}
			// Import the real source, not the agent-side junction/symlink.
			source := ResolveSourcePath(dir)
			sourceInfo, err := os.Stat(source)
			if err != nil || !sourceInfo.IsDir() {
				continue
			}
			if isWithin(m.SkillsDir(), source) {
				continue
			}
			if _, ok := FindSkillFile(source); !ok {
				continue
			}
			key := strings.ToLower(filepath.Clean(source))
			if idx, ok := seen[key]; ok {
				out[idx].Agents = append(out[idx].Agents, agent.Name)
				out[idx].IsLink = out[idx].IsLink || link
				continue
			}
			insp, err := Inspect(source)
			if err != nil {
				continue
			}
			item := Importable{
				Path:             source,
				Name:             insp.Name,
				Description:      insp.Description,
				Slug:             insp.Slug,
				Content:          insp.Content,
				ContentFiles:     insp.ContentFiles,
				SizeBytes:        insp.SizeBytes,
				FileCount:        insp.FileCount,
				Agents:           []string{agent.Name},
				AlreadyInstalled: installed[insp.Slug],
				IsLink:           link,
				SourceID:         "local",
				SourceRef:        source,
			}
			seen[key] = len(out)
			out = append(out, item)
		}
	}
	sortImportables(out)
	return out, nil
}

func (m *Manager) AllSkillFiles(skillID string) ([]FileEntry, error) {
	skill, err := m.Store.GetSkill(skillID)
	if err != nil {
		return nil, fmt.Errorf("技能不存在")
	}
	return ListFiles(skill.VaultPath, 400)
}

// ---------- helpers ----------

func defaultLinkMode() string {
	if LinkMode() == "junction" {
		return "junction"
	}
	return "symlink"
}

func isRealDir(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.IsDir() && !IsLink(path)
}

func isWithin(base, path string) bool {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(baseAbs, pathAbs)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
