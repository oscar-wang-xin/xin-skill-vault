package vault

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	Version     string `json:"version"`
	License     string `json:"license"`
}

type FileEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
}

type Inspection struct {
	Meta
	Path         string   `json:"path"`
	Content      string   `json:"content"`
	ContentFiles []string `json:"contentFiles"`
	SizeBytes    int64    `json:"sizeBytes"`
	FileCount    int      `json:"fileCount"`
	HasSkill     bool     `json:"hasSkillMd"`
}

// maxContentFiles keeps the persisted JSON and hover tooltips bounded.
const maxContentFiles = 16

var skipNames = map[string]bool{
	".git":          true,
	".svn":          true,
	".hg":           true,
	"node_modules":  true,
	"__pycache__":   true,
	".venv":         true,
	"venv":          true,
	".DS_Store":     true,
	"Thumbs.db":     true,
	".skill-vault":  true,
	".skills-cache": true,
}

func SkipName(name string) bool { return skipNames[name] }

// FindSkillFile returns the SKILL.md path inside dir, accepting case variants.
func FindSkillFile(dir string) (string, bool) {
	candidates := []string{"SKILL.md", "skill.md", "Skill.md"}
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, true
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), "SKILL.md") {
			return filepath.Join(dir, e.Name()), true
		}
	}
	return "", false
}

// Inspect reads SKILL.md metadata, computes size, file count and a factual
// content composition (executables / scripts / neither) without executing or
// interpreting file contents.
func Inspect(dir string) (Inspection, error) {
	fi, err := os.Stat(dir)
	if err != nil {
		return Inspection{}, fmt.Errorf("目录不存在: %w", err)
	}
	if !fi.IsDir() {
		return Inspection{}, fmt.Errorf("请选择一个技能目录，而不是文件")
	}
	walkDir := resolveLinkDir(dir)
	skillFile, has := FindSkillFile(walkDir)
	if !has {
		return Inspection{}, fmt.Errorf("所选目录缺少 SKILL.md，无法识别为技能")
	}
	raw, err := os.ReadFile(skillFile)
	if err != nil {
		return Inspection{}, fmt.Errorf("读取 SKILL.md 失败: %w", err)
	}
	meta := parseFrontmatter(string(raw))
	meta.Name = strings.TrimSpace(meta.Name)
	if meta.Name == "" {
		meta.Name = filepath.Base(dir)
	}
	meta.Slug = slugify(meta.Name)
	if meta.Slug == "" {
		meta.Slug = slugify(filepath.Base(dir))
	}
	if meta.Slug == "" {
		sum := sha256.Sum256([]byte(dir))
		meta.Slug = "skill-" + hex.EncodeToString(sum[:])[:8]
	}
	if meta.Description == "" {
		meta.Description = fallbackDescription(string(raw))
	}

	insp := Inspection{Meta: meta, Path: dir, HasSkill: true, Content: "none", ContentFiles: []string{}}
	hasExecutable := false
	hasScript := false
	contentCount := 0
	err = filepath.WalkDir(walkDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != walkDir && SkipName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, statErr := d.Info()
		if statErr == nil {
			insp.SizeBytes += info.Size()
			insp.FileCount++
		}
		if path == skillFile {
			// SKILL.md is metadata, not executable content.
			return nil
		}
		kind, label := classifyContentFile(path)
		if kind == "" {
			return nil
		}
		if kind == "executable" {
			hasExecutable = true
		} else {
			hasScript = true
		}
		contentCount++
		if len(insp.ContentFiles) < maxContentFiles {
			rel, relErr := filepath.Rel(walkDir, path)
			if relErr != nil {
				rel = filepath.Base(path)
			}
			insp.ContentFiles = append(insp.ContentFiles, filepath.ToSlash(rel)+"（"+label+"）")
		}
		return nil
	})
	if err != nil {
		return insp, fmt.Errorf("扫描技能目录失败: %w", err)
	}
	switch {
	case hasExecutable:
		insp.Content = "executable"
	case hasScript:
		insp.Content = "script"
	default:
		insp.Content = "none"
	}
	if contentCount > len(insp.ContentFiles) {
		insp.ContentFiles = append(insp.ContentFiles, fmt.Sprintf("另有 %d 个脚本或可执行文件未列出", contentCount-len(insp.ContentFiles)))
	}
	return insp, nil
}

// resolveLinkDir follows a symlink/junction chain for directory walking.
func resolveLinkDir(path string) string {
	source := ResolveSourcePath(path)
	if source == "" {
		return path
	}
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		return path
	}
	return source
}

var executableExt = map[string]bool{
	".exe": true, ".dll": true, ".msi": true, ".msix": true, ".appx": true,
	".scr": true, ".com": true, ".sys": true, ".drv": true, ".jar": true,
	".bin": true, ".so": true, ".dylib": true, ".app": true,
}

var scriptExt = map[string]bool{
	".ps1": true, ".psm1": true, ".bat": true, ".cmd": true,
	".sh": true, ".bash": true, ".zsh": true,
	".py": true, ".js": true, ".mjs": true, ".cjs": true, ".jsx": true,
	".ts": true, ".tsx": true, ".vbs": true, ".rb": true, ".pl": true,
	".php": true, ".lua": true, ".r": true, ".groovy": true,
}

// classifyContentFile reports factual file composition only. It never reads
// file contents and does not assign risk levels.
func classifyContentFile(path string) (string, string) {
	ext := strings.ToLower(filepath.Ext(path))
	if executableExt[ext] {
		return "executable", "可执行文件"
	}
	if scriptExt[ext] {
		return "script", "脚本"
	}
	return "", ""
}

func parseFrontmatter(content string) Meta {
	var meta Meta
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return meta
	}
	rest := content[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return meta
	}
	block := rest[:end]
	var lastKey string
	var descriptionLines []string
	scanner := bufio.NewScanner(strings.NewReader(block))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if lastKey == "description" {
				descriptionLines = append(descriptionLines, strings.TrimSpace(line))
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		value = strings.TrimSpace(value)
		lastKey = key
		switch key {
		case "name":
			meta.Name = cleanYAMLValue(value)
		case "description":
			if value == "" || value == ">" || value == "|" || value == ">-" || value == "|-" {
				descriptionLines = nil
			} else {
				meta.Description = cleanYAMLValue(value)
			}
		case "version":
			meta.Version = cleanYAMLValue(value)
		case "license":
			meta.License = cleanYAMLValue(value)
		}
	}
	if meta.Description == "" && len(descriptionLines) > 0 {
		meta.Description = strings.Join(descriptionLines, " ")
	}
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	return meta
}

func cleanYAMLValue(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		}
	}
	return strings.TrimSpace(v)
}

func fallbackDescription(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if strings.HasPrefix(content, "---\n") {
		rest := content[4:]
		if end := strings.Index(rest, "\n---"); end >= 0 {
			content = rest[end+4:]
		}
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "|") {
			continue
		}
		line = strings.TrimLeft(line, "-*0123456789. ")
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "`", "")
		if utf8.RuneCountInString(line) > 240 {
			line = string([]rune(line)[:240]) + "…"
		}
		return strings.TrimSpace(line)
	}
	return ""
}

func slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_':
			b.WriteRune(r)
			lastDash = false
		case r > 127 && (r >= 0x4e00 && r <= 0x9fff || r >= 0x3040 && r <= 0x30ff || r >= 0xac00 && r <= 0xd7af):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == ' ' || r == '/' || r == '\\' || r == ':':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-._")
	runes := []rune(out)
	if len(runes) > 64 {
		out = strings.Trim(string(runes[:64]), "-._")
	}
	return out
}

// HashDir computes a deterministic content hash over the whole skill directory.
func HashDir(dir string) (string, error) {
	h := sha256.New()
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && SkipName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	for _, rel := range files {
		path := filepath.Join(dir, rel)
		info, err := os.Lstat(path)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				continue
			}
			io.WriteString(h, "link\x00"+filepath.ToSlash(rel)+"\x00"+target+"\x00")
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		io.WriteString(h, "file\x00"+filepath.ToSlash(rel)+"\x00")
		io.Copy(h, f)
		f.Close()
		io.WriteString(h, "\x00")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ListFiles returns a sorted, bounded list of regular files for the inspector.
func ListFiles(dir string, limit int) ([]FileEntry, error) {
	if limit <= 0 {
		limit = 300
	}
	var out []FileEntry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && SkipName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		out = append(out, FileEntry{
			Path:    filepath.ToSlash(rel),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		})
		if len(out) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && err != fs.SkipAll {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
