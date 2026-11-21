package skill

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const DocName = "SKILL.md"

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Metadata struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Version     string   `yaml:"version" json:"version"`
	Author      string   `yaml:"author" json:"author"`
	License     string   `yaml:"license" json:"license"`
	Icon        string   `yaml:"icon" json:"icon"`
	Slug        string   `yaml:"slug" json:"slug"`
	Tags        []string `yaml:"tags" json:"tags"`
}

type Skill struct {
	Dir     string
	DocPath string
	Meta    Metadata
	Slug    string
	Body    string
}

// ParseDir reads SKILL.md from a skill directory and validates its frontmatter.
func ParseDir(dir string) (*Skill, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve skill directory: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("read skill directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skill path is not a directory: %s", abs)
	}
	docPath := filepath.Join(abs, DocName)
	data, err := os.ReadFile(docPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", DocName, err)
	}
	meta, body, err := ParseDocument(string(data))
	if err != nil {
		return nil, err
	}
	slug, err := ValidateMetadata(meta)
	if err != nil {
		return nil, err
	}
	return &Skill{Dir: abs, DocPath: docPath, Meta: meta, Slug: slug, Body: body}, nil
}

// ValidateMetadata normalizes and validates frontmatter, returning the canonical slug.
func ValidateMetadata(meta Metadata) (string, error) {
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Description = strings.TrimSpace(meta.Description)
	if meta.Name == "" {
		return "", fmt.Errorf("SKILL.md frontmatter is missing name")
	}
	if meta.Description == "" {
		return "", fmt.Errorf("SKILL.md frontmatter is missing description")
	}
	slug := strings.ToLower(strings.TrimSpace(meta.Slug))
	if slug == "" {
		slug = Slugify(meta.Name)
	}
	if !slugPattern.MatchString(slug) {
		return "", fmt.Errorf("invalid slug %q: use lowercase letters, digits and hyphens", slug)
	}
	return slug, nil
}

// ParseDocument splits YAML frontmatter from markdown body.
func ParseDocument(text string) (Metadata, string, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Metadata{}, "", fmt.Errorf("SKILL.md must start with YAML frontmatter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return Metadata{}, "", fmt.Errorf("SKILL.md frontmatter is not closed")
	}
	var meta Metadata
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &meta); err != nil {
		return Metadata{}, "", fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}
	body := strings.Join(lines[end+1:], "\n")
	return meta, strings.TrimLeft(body, "\n"), nil
}

// Slugify creates an ASCII slug from a display name.
func Slugify(name string) string {
	var b strings.Builder
	lastHyphen := true
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case !lastHyphen:
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		sum := sha1.Sum([]byte(name))
		slug = "skill-" + hex.EncodeToString(sum[:4])
	}
	if len(slug) > 64 {
		slug = strings.TrimRight(slug[:64], "-")
	}
	return slug
}
