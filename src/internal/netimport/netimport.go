package netimport

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxDownload = 96 << 20  // 96 MiB compressed
	maxExtract  = 512 << 20 // 512 MiB extracted
	maxEntries  = 20000
)

type Result struct {
	Root      string
	SubPath   string
	Kind      string
	SourceURL string
}

type httpStatusError struct {
	Status int
	URL    string
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("下载失败（HTTP %d）：%s", e.Status, e.URL)
}

type githubRef struct {
	Owner   string
	Repo    string
	Branch  string
	SubPath string
}

func Fetch(ctx context.Context, rawURL, dest string) (Result, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return Result{}, err
	}
	if ref, ok := parseGitHub(u); ok {
		return fetchGitHub(ctx, ref, dest, rawURL)
	}
	if isArchivePath(u.Path) {
		return fetchArchive(ctx, u, dest, rawURL)
	}
	return Result{}, fmt.Errorf("不支持的地址。请提供 GitHub 仓库/目录、zip 或 tar.gz 地址")
}

func normalizeURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("请输入网络地址")
	}
	if strings.HasPrefix(raw, "git@github.com:") {
		raw = "https://github.com/" + strings.TrimPrefix(raw, "git@github.com:")
	}
	if !strings.Contains(raw, "://") {
		lower := strings.ToLower(raw)
		switch {
		case strings.HasPrefix(lower, "github.com/"):
			raw = "https://" + raw
		case strings.Count(raw, "/") == 1 && !strings.Contains(raw, "."):
			raw = "https://github.com/" + raw
		default:
			raw = "https://" + raw
		}
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("地址格式不正确：%w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("只支持 http/https 地址")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("地址缺少主机名")
	}
	if u.User != nil {
		return nil, fmt.Errorf("地址不能包含用户名或密码")
	}
	u.Fragment = ""
	return u, nil
}

func parseGitHub(u *url.URL) (githubRef, bool) {
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	parts := splitPath(u.Path)
	switch host {
	case "github.com":
		if len(parts) < 2 {
			return githubRef{}, false
		}
		if len(parts) > 2 && (parts[2] == "archive" || parts[2] == "releases" || parts[2] == "raw") {
			return githubRef{}, false
		}
		ref := githubRef{Owner: parts[0], Repo: strings.TrimSuffix(parts[1], ".git")}
		if len(parts) >= 4 && (parts[2] == "tree" || parts[2] == "blob") {
			ref.Branch = parts[3]
			rest := strings.Join(parts[4:], "/")
			if parts[2] == "blob" {
				rest = path.Dir(rest)
				if rest == "." {
					rest = ""
				}
			}
			ref.SubPath = rest
		}
		return ref, ref.Owner != "" && ref.Repo != ""
	case "raw.githubusercontent.com":
		if len(parts) < 4 {
			return githubRef{}, false
		}
		ref := githubRef{Owner: parts[0], Repo: parts[1], Branch: parts[2]}
		rest := path.Dir(strings.Join(parts[3:], "/"))
		if rest == "." {
			rest = ""
		}
		ref.SubPath = rest
		return ref, true
	}
	return githubRef{}, false
}

func splitPath(value string) []string {
	var out []string
	for _, part := range strings.Split(filepath.ToSlash(value), "/") {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isArchivePath(p string) bool {
	lower := strings.ToLower(p)
	return strings.HasSuffix(lower, ".zip") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

func fetchGitHub(ctx context.Context, ref githubRef, dest, sourceURL string) (Result, error) {
	var branches []string
	if ref.Branch != "" {
		branches = []string{ref.Branch}
	} else {
		branches = appendUnique(nil, defaultBranch(ctx, ref.Owner, ref.Repo))
		branches = appendUnique(branches, "main")
		branches = appendUnique(branches, "master")
	}
	var lastErr error
	for _, branch := range branches {
		if branch == "" {
			continue
		}
		archiveURL := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s",
			url.PathEscape(ref.Owner), url.PathEscape(ref.Repo), escapeBranch(branch))
		archivePath := filepath.Join(dest, "source.zip")
		_ = os.Remove(archivePath)
		if err := download(ctx, archiveURL, archivePath); err != nil {
			lastErr = err
			var statusErr *httpStatusError
			if errors.As(err, &statusErr) && (statusErr.Status == http.StatusNotFound || statusErr.Status == http.StatusForbidden) {
				continue
			}
			return Result{}, err
		}
		extractDir := filepath.Join(dest, "src")
		if err := os.MkdirAll(extractDir, 0o755); err != nil {
			return Result{}, err
		}
		if err := extractZip(archivePath, extractDir); err != nil {
			return Result{}, fmt.Errorf("解压 GitHub 仓库失败：%w", err)
		}
		root := singleTopDir(extractDir)
		subPath := filepath.FromSlash(ref.SubPath)
		if subPath != "" {
			target, err := safeJoin(root, subPath)
			if err != nil {
				return Result{}, err
			}
			if info, err := os.Stat(target); err != nil || !info.IsDir() {
				return Result{}, fmt.Errorf("仓库中未找到目录：%s", ref.SubPath)
			}
		}
		return Result{Root: root, SubPath: subPath, Kind: "github", SourceURL: sourceURL}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无法确定默认分支")
	}
	return Result{}, fmt.Errorf("下载 GitHub 仓库失败：%w", lastErr)
}

func fetchArchive(ctx context.Context, u *url.URL, dest, sourceURL string) (Result, error) {
	lower := strings.ToLower(u.Path)
	ext := ".zip"
	if strings.HasSuffix(lower, ".tar.gz") {
		ext = ".tar.gz"
	} else if strings.HasSuffix(lower, ".tgz") {
		ext = ".tgz"
	}
	archivePath := filepath.Join(dest, "source"+ext)
	if err := download(ctx, u.String(), archivePath); err != nil {
		return Result{}, err
	}
	extractDir := filepath.Join(dest, "src")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return Result{}, err
	}
	if ext == ".zip" {
		if err := extractZip(archivePath, extractDir); err != nil {
			return Result{}, fmt.Errorf("解压 zip 失败：%w", err)
		}
	} else {
		if err := extractTarGz(archivePath, extractDir); err != nil {
			return Result{}, fmt.Errorf("解压 tar.gz 失败：%w", err)
		}
	}
	return Result{Root: singleTopDir(extractDir), Kind: "archive", SourceURL: sourceURL}, nil
}

func defaultBranch(ctx context.Context, owner, repo string) string {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", url.PathEscape(owner), url.PathEscape(repo))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "main"
	}
	req.Header.Set("User-Agent", "XinSkillVault/0.1")
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "main"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "main"
	}
	var payload struct {
		DefaultBranch string `json:"default_branch"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload) != nil {
		return "main"
	}
	if payload.DefaultBranch == "" {
		return "main"
	}
	return payload.DefaultBranch
}

func escapeBranch(branch string) string {
	parts := strings.Split(branch, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func download(ctx context.Context, rawURL, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "XinSkillVault/0.1")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &httpStatusError{Status: resp.StatusCode, URL: rawURL}
	}
	if resp.ContentLength > maxDownload {
		return fmt.Errorf("压缩包过大（%d MB），上限 96 MB", resp.ContentLength>>20)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, io.LimitReader(resp.Body, maxDownload+1))
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n > maxDownload {
		_ = os.Remove(dst)
		return fmt.Errorf("压缩包超过 96 MB 上限")
	}
	return nil
}

func extractZip(archive, dest string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer reader.Close()
	if len(reader.File) > maxEntries {
		return fmt.Errorf("压缩包文件数超过上限（%d）", maxEntries)
	}
	var total int64
	for _, entry := range reader.File {
		target, err := safeJoin(dest, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if !entry.Mode().IsRegular() {
			// Archive symlinks can escape the extraction root; skip them.
			continue
		}
		total += int64(entry.UncompressedSize64)
		if total > maxExtract {
			return fmt.Errorf("解压后体积超过 512 MB 上限")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := entry.Open()
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if entry.Mode()&0o111 != 0 {
			mode = 0o755
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, maxExtract+1))
		rc.Close()
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractTarGz(archive, dest string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	var total int64
	entries := 0
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		entries++
		if entries > maxEntries {
			return fmt.Errorf("压缩包文件数超过上限（%d）", maxEntries)
		}
		target, err := safeJoin(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			total += header.Size
			if total > maxExtract {
				return fmt.Errorf("解压后体积超过 512 MB 上限")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if header.FileInfo().Mode()&0o111 != 0 {
				mode = 0o755
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, io.LimitReader(reader, maxExtract+1))
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		default:
			// Skip symlinks, hardlinks and special files.
			continue
		}
	}
	return nil
}

func safeJoin(base, name string) (string, error) {
	name = filepath.FromSlash(name)
	if name == "" || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", fmt.Errorf("压缩包包含非法路径：%s", name)
	}
	target := filepath.Join(base, filepath.Clean(name))
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("压缩包路径逃逸：%s", name)
	}
	return target, nil
}

func singleTopDir(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return root
	}
	return filepath.Join(root, entries[0].Name())
}
