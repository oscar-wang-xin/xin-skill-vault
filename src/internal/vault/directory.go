package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
)

type DirectoryEntry struct {
	Name             string   `json:"name"`
	DisplayName      string   `json:"displayName"`
	Description      string   `json:"description"`
	Path             string   `json:"path"`
	SourcePath       string   `json:"sourcePath"`
	LinkTarget       string   `json:"linkTarget"`
	Kind             string   `json:"kind"`   // junction | symlink | copy | external-link | external-dir | broken
	Status           string   `json:"status"` // managed | external | broken | drifted
	Mode             string   `json:"mode"`
	Content          string   `json:"content"`
	ContentFiles     []string `json:"contentFiles"`
	SizeBytes        int64    `json:"sizeBytes"`
	FileCount        int      `json:"fileCount"`
	IsLink           bool     `json:"isLink"`
	Managed          bool     `json:"managed"`
	VaultSkillID     string   `json:"vaultSkillId"`
	AlreadyInVault   bool     `json:"alreadyInVault"`
	DistributionPath string   `json:"distributionPath"`
}

// DirectoryEntries lists the physical skill entries inside an Agent or project
// directory and classifies each one: managed link/copy, external link, external
// real directory, or broken link.
func (m *Manager) DirectoryEntries(target store.Agent) ([]DirectoryEntry, error) {
	entries, err := os.ReadDir(target.SkillsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []DirectoryEntry{}, nil
		}
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	dists, _ := m.Store.ListDistributions()
	distMap := map[string]store.Distribution{}
	for _, dist := range dists {
		if dist.AgentID == target.ID {
			distMap[dist.SkillID] = dist
		}
	}
	vaultSkills, _ := m.Store.ListSkills()
	vaultByID := map[string]store.Skill{}
	vaultByPath := map[string]store.Skill{}
	for _, skill := range vaultSkills {
		vaultByID[skill.ID] = skill
		vaultByPath[strings.ToLower(filepath.Clean(skill.VaultPath))] = skill
	}
	installed := map[string]bool{}
	for _, skill := range vaultSkills {
		installed[skill.ID] = true
	}

	var out []DirectoryEntry
	for _, entry := range entries {
		if SkipName(entry.Name()) || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(target.SkillsPath, entry.Name())
		link := IsLink(path)
		source := ResolveSourcePath(path)
		sourceInfo, _ := os.Stat(source)
		_, pathErr := os.Stat(path)
		broken := pathErr != nil

		item := DirectoryEntry{
			Name: entry.Name(), Path: path, SourcePath: source, IsLink: link,
			Kind: "external-dir", Status: "external", AlreadyInVault: installed[entry.Name()],
		}
		if target, ok := LinkTarget(path); ok {
			item.LinkTarget = target
		}
		if broken {
			item.Kind = "broken"
			item.Status = "broken"
		} else if sourceInfo != nil && sourceInfo.IsDir() {
			if insp, err := Inspect(source); err == nil {
				item.DisplayName = insp.Name
				item.Description = insp.Description
				item.Content = insp.Content
				item.ContentFiles = insp.ContentFiles
				item.SizeBytes = insp.SizeBytes
				item.FileCount = insp.FileCount
				item.AlreadyInVault = installed[insp.Slug]
			}
		}
		if item.DisplayName == "" {
			item.DisplayName = entry.Name()
		}

		if dist, ok := distMap[entry.Name()]; ok {
			item.Managed = true
			item.Mode = dist.Mode
			item.Kind = dist.Mode
			if dist.Status == "ok" {
				item.Status = "managed"
			} else {
				item.Status = dist.Status
			}
			item.VaultSkillID = dist.SkillID
			item.DistributionPath = dist.LinkPath
		} else if skill, ok := vaultByPath[strings.ToLower(filepath.Clean(source))]; ok {
			item.Managed = true
			item.Status = "managed"
			item.VaultSkillID = skill.ID
			item.Kind = "unregistered-link"
		} else if isWithin(m.SkillsDir(), source) {
			if skill, ok := vaultByID[entry.Name()]; ok {
				item.Managed = true
				item.Status = "managed"
				item.VaultSkillID = skill.ID
				item.Kind = "unregistered-link"
			}
		}
		if !item.Managed {
			if link {
				item.Kind = "external-link"
			} else {
				item.Kind = "external-dir"
			}
			item.Status = "external"
		}
		if broken {
			item.Status = "broken"
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Managed != out[j].Managed {
			return out[i].Managed
		}
		return strings.ToLower(out[i].DisplayName) < strings.ToLower(out[j].DisplayName)
	})
	return out, nil
}

// RemoveDirectoryEntry removes a physical entry from an Agent/project skills
// directory. Links are removed directly. Real directories are renamed in place
// with a .removed-<timestamp> suffix, so nothing is destroyed silently.
func (m *Manager) RemoveDirectoryEntry(target store.Agent, name string) error {
	if name == "" || filepath.Base(name) != name {
		return fmt.Errorf("非法的目录项名称")
	}
	path := filepath.Join(target.SkillsPath, name)
	if filepath.Dir(path) != filepath.Clean(target.SkillsPath) || !isWithin(target.SkillsPath, path) {
		return fmt.Errorf("拒绝删除技能目录之外的路径")
	}
	if _, err := os.Lstat(path); err != nil {
		return fmt.Errorf("目录项不存在")
	}
	if IsLink(path) {
		if err := RemoveLink(path); err != nil {
			return fmt.Errorf("删除链接失败: %w", err)
		}
	} else {
		stamp := time.Now().Format("20060102-150405")
		trash := path + ".removed-" + stamp
		if err := os.Rename(path, trash); err != nil {
			return fmt.Errorf("移动目录失败: %w", err)
		}
	}
	_ = m.Store.DeleteDistribution(name, target.ID)
	m.Store.AddActivity("remove-directory", name, target.Name)
	return nil
}

func (m *Manager) DirectoryEntryPath(target store.Agent, name string) (string, error) {
	if name == "" || filepath.Base(name) != name {
		return "", fmt.Errorf("非法的目录项名称")
	}
	path := filepath.Join(target.SkillsPath, name)
	if !isWithin(target.SkillsPath, path) {
		return "", fmt.Errorf("路径不在目标目录内")
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("目录项不存在")
	}
	return path, nil
}
