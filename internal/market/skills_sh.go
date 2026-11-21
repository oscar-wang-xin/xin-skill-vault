package market

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xinskillvault/internal/skill"
)

type SkillsSh struct {
	client *http.Client
}

var skillsShCache struct {
	sync.Mutex
	items     []Skill
	loadedAt  time.Time
	loadError error
}

func NewSkillsSh() *SkillsSh {
	return &SkillsSh{client: &http.Client{Timeout: 40 * time.Second}}
}

func (p *SkillsSh) ID() string { return "skills-sh" }

type sitemapIndex struct {
	Locations []string `xml:"sitemap>loc"`
}

func (p *SkillsSh) catalog(ctx context.Context) ([]Skill, error) {
	skillsShCache.Lock()
	defer skillsShCache.Unlock()
	if len(skillsShCache.items) > 0 && time.Since(skillsShCache.loadedAt) < 6*time.Hour {
		return skillsShCache.items, nil
	}
	items, err := p.fetchCatalog(ctx)
	if err != nil {
		if len(skillsShCache.items) > 0 {
			return skillsShCache.items, nil
		}
		return nil, err
	}
	skillsShCache.items = items
	skillsShCache.loadedAt = time.Now()
	return items, nil
}

func (p *SkillsSh) fetchCatalog(ctx context.Context) ([]Skill, error) {
	indexBody, err := p.get(ctx, "https://www.skills.sh/sitemap.xml")
	if err != nil {
		return nil, err
	}
	var index sitemapIndex
	if err := xml.Unmarshal(indexBody, &index); err != nil {
		return nil, fmt.Errorf("解析 skills.sh sitemap: %w", err)
	}
	var skills []Skill
	seen := map[string]bool{}
	for _, location := range index.Locations {
		if !strings.Contains(location, "sitemap-skills") {
			continue
		}
		body, err := p.get(ctx, location)
		if err != nil {
			continue
		}
		var sitemap struct {
			Locations []string `xml:"url>loc"`
		}
		if err := xml.Unmarshal(body, &sitemap); err != nil {
			continue
		}
		for _, link := range sitemap.Locations {
			parsed, err := url.Parse(link)
			if err != nil {
				continue
			}
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(parts) < 3 || parts[0] == "site" {
				continue
			}
			owner, repo, slug := parts[0], parts[1], parts[len(parts)-1]
			key := owner + "/" + repo + "/" + slug
			if seen[key] {
				continue
			}
			seen[key] = true
			skills = append(skills, Skill{
				ID: "skills-sh:" + key, Source: "skills-sh", SourceName: "skills.sh",
				Slug: slug, Namespace: owner + "/" + repo, Name: titleFromSlug(slug),
				Author: owner, Tags: []string{}, URL: link, Repository: owner + "/" + repo,
			})
		}
	}
	if len(skills) == 0 {
		return nil, fmt.Errorf("skills.sh sitemap 没有技能")
	}
	return skills, nil
}

func (p *SkillsSh) Search(ctx context.Context, query, category string, page int) ([]Skill, int, error) {
	items, err := p.catalog(ctx)
	if err != nil {
		return nil, 0, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	matches := make([]Skill, 0, 24)
	for _, item := range items {
		if query != "" {
			haystack := strings.ToLower(item.Name + " " + item.Slug + " " + item.Namespace)
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		matches = append(matches, item)
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * 24
	if start >= len(matches) {
		return []Skill{}, len(matches), nil
	}
	end := start + 24
	if end > len(matches) {
		end = len(matches)
	}
	return matches[start:end], len(matches), nil
}

func (p *SkillsSh) Download(ctx context.Context, item Skill, destination string) error {
	repository := strings.TrimSpace(item.Namespace)
	if item.Repository != "" {
		repository = item.Repository
	}
	if repository == "" || strings.Count(repository, "/") != 1 {
		return fmt.Errorf("skills.sh 技能缺少 GitHub 仓库")
	}
	var archive []byte
	var lastErr error
	for _, branch := range []string{"main", "master"} {
		body, err := p.get(ctx, fmt.Sprintf("https://codeload.github.com/%s/zip/refs/heads/%s", repository, branch))
		if err == nil {
			archive = body
			break
		}
		lastErr = err
	}
	if len(archive) == 0 {
		return fmt.Errorf("下载 GitHub 仓库失败: %w", lastErr)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return fmt.Errorf("打开 GitHub 仓库包: %w", err)
	}
	root := ""
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(entry.Name, "/")
		if !strings.HasSuffix(strings.ToLower(name), "/skill.md") && !strings.EqualFold(name, skill.DocName) {
			continue
		}
		if strings.EqualFold(path.Base(path.Dir(name)), item.Slug) {
			root = path.Dir(name)
			break
		}
		if root == "" {
			root = path.Dir(name)
		}
	}
	if root == "" {
		return fmt.Errorf("仓库里没有找到 SKILL.md")
	}
	tempDir, err := os.MkdirTemp("", "xsv-skillssh-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name := strings.TrimPrefix(entry.Name, "/")
		if root != "." && !strings.HasPrefix(name, root+"/") {
			continue
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(name, root), "/")
		rel = path.Clean(rel)
		if rel == "." || strings.HasPrefix(rel, "../") {
			continue
		}
		target := filepath.Join(tempDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := entry.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		in.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return zipDirectory(tempDir, destination)
}

func (p *SkillsSh) get(ctx context.Context, target string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "XinSkillVault/0.1")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh 返回 %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, 200<<20))
}

func titleFromSlug(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '-' || r == '_' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func zipDirectory(source, destination string) error {
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(file)
	walkErr := filepath.WalkDir(source, func(current string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		writer, err := archive.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		input, err := os.Open(current)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		input.Close()
		return copyErr
	})
	if walkErr != nil {
		archive.Close()
		file.Close()
		os.Remove(destination)
		return walkErr
	}
	if err := archive.Close(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
