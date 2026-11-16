package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxDiscoveryDepth = 6
const maxDiscoveryResults = 300
const maxDiscoveryDirs = 5000

// ResolveSourcePath follows junctions/symlinks link-by-link to the real source
// directory. filepath.EvalSymlinks intentionally leaves Windows junctions
// unresolved, so os.Readlink is used first, then EvalSymlinks for POSIX links
// and symlinked path components.
func ResolveSourcePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	current := filepath.Clean(abs)
	seen := map[string]bool{}
	for i := 0; i < 16; i++ {
		key := strings.ToLower(current)
		if seen[key] {
			break
		}
		seen[key] = true
		target, ok := LinkTarget(current)
		if !ok {
			break
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		current = filepath.Clean(target)
	}
	if real, err := filepath.EvalSymlinks(current); err == nil {
		current = real
	}
	return current
}

// DiscoverFolder recursively finds skill directories (directories containing
// SKILL.md) under root, stopping at each skill boundary. It is used both by
// local folder import and by network archive extraction.
func (m *Manager) DiscoverFolder(root string) ([]Importable, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	source := ResolveSourcePath(abs)
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("目录不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("请选择一个文件夹")
	}

	existing, _ := m.Store.ListSkills()
	installed := map[string]bool{}
	for _, skill := range existing {
		installed[skill.ID] = true
	}

	seen := map[string]bool{}
	dirs := 0
	var out []Importable
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if len(out) >= maxDiscoveryResults || dirs >= maxDiscoveryDirs || depth > maxDiscoveryDepth {
			return
		}
		canonical := ResolveSourcePath(dir)
		key := strings.ToLower(filepath.Clean(canonical))
		if seen[key] {
			return
		}
		seen[key] = true
		dirs++

		if _, ok := FindSkillFile(canonical); ok {
			insp, err := Inspect(canonical)
			if err == nil {
				out = append(out, Importable{
					Path: canonical, Name: insp.Name, Description: insp.Description,
					Slug: insp.Slug, Content: insp.Content, ContentFiles: insp.ContentFiles,
					SizeBytes: insp.SizeBytes,
					FileCount: insp.FileCount, Agents: []string{},
					AlreadyInstalled: installed[insp.Slug], IsLink: IsLink(dir),
					SourceID: "local", SourceRef: canonical,
				})
			}
			return
		}
		if depth == maxDiscoveryDepth {
			return
		}
		entries, err := os.ReadDir(canonical)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if len(out) >= maxDiscoveryResults || dirs >= maxDiscoveryDirs {
				return
			}
			name := entry.Name()
			if SkipName(name) || strings.HasPrefix(name, ".") {
				continue
			}
			child := filepath.Join(canonical, name)
			childInfo, err := os.Stat(child)
			if err != nil || !childInfo.IsDir() {
				continue
			}
			walk(child, depth+1)
		}
	}
	walk(source, 0)
	sortImportables(out)
	return out, nil
}

func sortImportables(items []Importable) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].AlreadyInstalled != items[j].AlreadyInstalled {
			return !items[i].AlreadyInstalled
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}
