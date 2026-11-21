package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xinskillvault/internal/agent"
	"xinskillvault/internal/skill"
	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

type Finding struct {
	Key         string `json:"key"`
	AgentID     string `json:"agentId"`
	AgentName   string `json:"agentName"`
	AgentPath   string `json:"agentPath"`
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Hash        string `json:"hash"`
	SizeBytes   int64  `json:"sizeBytes"`
	FileCount   int    `json:"fileCount"`
}

type AgentResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Short    string `json:"short"`
	Color    string `json:"color"`
	Path     string `json:"path"`
	Mode     string `json:"mode"`
	Detected bool   `json:"detected"`
	Count    int    `json:"count"`
}

type Result struct {
	ScannedAt     int64         `json:"scannedAt"`
	Agents        []AgentResult `json:"agents"`
	Findings      []Finding     `json:"findings"`
	Errors        []string      `json:"errors"`
	ScannedSkills int           `json:"scannedSkills"`
}

// Scan inspects each detected Agent skill directory and reports new/changed skills.
func Scan(adapters []agent.Adapter, st *store.Store) Result {
	result := Result{ScannedAt: time.Now().Unix(), Findings: []Finding{}, Errors: []string{}}
	for _, adapter := range adapters {
		item := AgentResult{
			ID: adapter.ID, Name: adapter.Name, Short: adapter.Short, Color: adapter.Color,
			Path: adapter.Path, Mode: adapter.Mode, Detected: adapter.Detected,
		}
		if !adapter.Detected {
			result.Agents = append(result.Agents, item)
			continue
		}
		findings, count, errs := scanAdapter(adapter, st)
		item.Count = count
		result.Agents = append(result.Agents, item)
		result.Findings = append(result.Findings, findings...)
		result.ScannedSkills += count
		result.Errors = append(result.Errors, errs...)
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		if result.Findings[i].Status != result.Findings[j].Status {
			return statusRank(result.Findings[i].Status) < statusRank(result.Findings[j].Status)
		}
		return result.Findings[i].Name < result.Findings[j].Name
	})
	return result
}

func scanAdapter(adapter agent.Adapter, st *store.Store) ([]Finding, int, []string) {
	var findings []Finding
	var errors []string
	count := 0
	root := adapter.Path
	candidates := []string{}
	if hasSkillDoc(root) {
		candidates = append(candidates, root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return findings, count, []string{fmt.Sprintf("%s: %v", adapter.Name, err)}
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == "node_modules" || name == "Thumbs.db" {
			continue
		}
		path := filepath.Join(root, name)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if hasSkillDoc(path) {
			candidates = append(candidates, path)
		}
	}
	seen := map[string]bool{}
	for _, dir := range candidates {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		parsed, err := skill.ParseDir(dir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		hash, size, fileCount, err := vault.FingerprintDir(dir)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		count++
		record, err := st.FindSkillBySlug(parsed.Slug)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		status := "new"
		if record != nil {
			if record.ContentHash == hash {
				continue
			}
			status = "changed"
		}
		findings = append(findings, Finding{
			Key:     adapter.ID + ":" + parsed.Slug + ":" + dir,
			AgentID: adapter.ID, AgentName: adapter.Name, AgentPath: root, Path: dir,
			Slug: parsed.Slug, Name: parsed.Meta.Name, Description: parsed.Meta.Description,
			Version: parsed.Meta.Version, Status: status, Hash: hash, SizeBytes: size, FileCount: fileCount,
		})
	}
	return findings, count, errors
}

func hasSkillDoc(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, skill.DocName))
	return err == nil && !info.IsDir()
}

func statusRank(status string) int {
	if status == "new" {
		return 0
	}
	return 1
}
