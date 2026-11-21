package netimport

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"xinskillvault/internal/skill"
)

const maxDownload = 200 << 20
const maxExtract = 512 << 20

type Fetcher struct {
	Client *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{Client: &http.Client{Timeout: 60 * time.Second}}
}

// Fetch downloads and extracts a URL into a temporary directory, returning the skill root.
func (f *Fetcher) Fetch(ctx context.Context, rawURL, tempRoot string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("仅支持 http/https URL")
	}
	lowerPath := strings.ToLower(parsed.Path)
	if strings.HasSuffix(lowerPath, ".md") && !isGitHubRepo(parsed) {
		target := filepath.Join(tempRoot, skill.DocName)
		if err := f.downloadFile(ctx, parsed.String(), target); err != nil {
			return "", err
		}
		return tempRoot, nil
	}
	if strings.HasSuffix(lowerPath, ".zip") || strings.HasSuffix(lowerPath, ".skill") {
		archive := filepath.Join(tempRoot, "download.zip")
		if err := f.downloadFile(ctx, parsed.String(), archive); err != nil {
			return "", err
		}
		return extractZipSkill(archive, tempRoot)
	}
	if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		archive := filepath.Join(tempRoot, "download.tar.gz")
		if err := f.downloadFile(ctx, parsed.String(), archive); err != nil {
			return "", err
		}
		if err := extractTarGz(archive, tempRoot); err != nil {
			return "", err
		}
		return findSkillRoot(tempRoot, "")
	}
	if isGitHubRepo(parsed) {
		return f.fetchGitHub(ctx, parsed, tempRoot)
	}
	return "", fmt.Errorf("无法识别的导入 URL")
}

func isGitHubRepo(parsed *url.URL) bool {
	return strings.EqualFold(parsed.Host, "github.com") && len(strings.Split(strings.Trim(parsed.Path, "/"), "/")) >= 2
}

func (f *Fetcher) fetchGitHub(ctx context.Context, parsed *url.URL, tempRoot string) (string, error) {
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	owner, repo := parts[0], parts[1]
	branch := ""
	subPath := ""
	if len(parts) >= 4 && parts[2] == "tree" {
		branch = parts[3]
		subPath = strings.Join(parts[4:], "/")
	}
	branches := []string{branch}
	if branch == "" {
		branches = []string{"main", "master"}
	}
	var lastErr error
	for _, candidate := range branches {
		downloadURL := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", owner, repo, candidate)
		archive := filepath.Join(tempRoot, "repo.zip")
		if err := f.downloadFile(ctx, downloadURL, archive); err != nil {
			lastErr = err
			continue
		}
		if err := extractZipAll(archive, tempRoot); err != nil {
			return "", err
		}
		return findSkillRoot(tempRoot, subPath)
	}
	return "", fmt.Errorf("下载 GitHub 仓库失败: %w", lastErr)
}

func (f *Fetcher) downloadFile(ctx context.Context, target, destination string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "XinSkillVault/0.1")
	response, err := f.Client.Do(request)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("下载返回 %d", response.StatusCode)
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, maxDownload+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxDownload {
		return fmt.Errorf("下载超过 200MB 限制")
	}
	return nil
}

func extractZipSkill(archive, tempRoot string) (string, error) {
	extractRoot := filepath.Join(tempRoot, "extract")
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return "", err
	}
	if err := extractZipAll(archive, extractRoot); err != nil {
		return "", err
	}
	return findSkillRoot(extractRoot, "")
}

func extractZipAll(archive, destination string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("打开 zip: %w", err)
	}
	defer reader.Close()
	var total uint64
	for _, entry := range reader.File {
		name, err := safeName(entry.Name)
		if err != nil {
			return err
		}
		total += entry.UncompressedSize64
		if total > maxExtract {
			return fmt.Errorf("解压超过 512MB 限制")
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !within(destination, target) {
			return fmt.Errorf("压缩包包含越界路径：%s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		input, err := entry.Open()
		if err != nil {
			return err
		}
		output, err := os.Create(target)
		if err != nil {
			input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		input.Close()
		output.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func extractTarGz(archive, destination string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	var total int64
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name, err := safeName(header.Name)
		if err != nil {
			return err
		}
		total += header.Size
		if total > maxExtract {
			return fmt.Errorf("解压超过 512MB 限制")
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !within(destination, target) {
			return fmt.Errorf("压缩包包含越界路径：%s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			output, err := os.Create(target)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(output, io.LimitReader(tarReader, header.Size))
			output.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	return nil
}

func findSkillRoot(root, filter string) (string, error) {
	var candidates []string
	if err := filepath.WalkDir(root, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.EqualFold(entry.Name(), skill.DocName) {
			return nil
		}
		dir := filepath.Dir(current)
		if filter != "" {
			rel, err := filepath.Rel(root, dir)
			if err != nil || !strings.HasSuffix(filepath.ToSlash(rel), strings.Trim(filepath.ToSlash(filter), "/")) {
				return nil
			}
		}
		candidates = append(candidates, dir)
		return nil
	}); err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("下载内容里没有找到 SKILL.md")
	}
	best := candidates[0]
	bestDepth := strings.Count(best, string(filepath.Separator))
	for _, candidate := range candidates[1:] {
		depth := strings.Count(candidate, string(filepath.Separator))
		if depth < bestDepth {
			best = candidate
			bestDepth = depth
		} else if depth == bestDepth {
			return "", fmt.Errorf("下载内容包含多个 SKILL.md，无法确定导入根目录")
		}
	}
	return best, nil
}

func safeName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	clean := path.Clean(strings.TrimPrefix(name, "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
		return "", fmt.Errorf("压缩包包含不安全路径：%s", name)
	}
	return clean, nil
}

func within(root, target string) bool {
	rootAbs, _ := filepath.Abs(root)
	targetAbs, _ := filepath.Abs(target)
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
