package vault

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xinskillvault/internal/skill"
	"xinskillvault/internal/store"
)

var ignoredNames = map[string]bool{
	".git": true, "node_modules": true, ".skillrepo": true,
	".DS_Store": true, "Thumbs.db": true, ".skillrepo-manifest.json": true,
}

type Vault struct {
	Root  string
	Store *store.Store
}

type FileEntry struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	IsDir bool   `json:"isDir"`
}

func New(root string, st *store.Store) *Vault {
	return &Vault{Root: root, Store: st}
}

func (v *Vault) SkillsDir() string {
	return filepath.Join(v.Root, "skills")
}

// ImportDirectory validates a skill directory, copies it into the vault atomically and indexes it.
func (v *Vault) ImportDirectory(source string) (store.SkillRecord, error) {
	var zero store.SkillRecord
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return zero, fmt.Errorf("resolve source: %w", err)
	}
	info, err := os.Stat(sourceAbs)
	if err != nil {
		return zero, fmt.Errorf("read source: %w", err)
	}
	if !info.IsDir() {
		return zero, fmt.Errorf("source is not a directory: %s", sourceAbs)
	}
	if within(v.Root, sourceAbs) {
		return zero, fmt.Errorf("不能直接从仓库内部导入技能")
	}
	parsed, err := skill.ParseDir(sourceAbs)
	if err != nil {
		return zero, err
	}
	if existing, err := v.Store.FindSkillBySlug(parsed.Slug); err != nil {
		return zero, err
	} else if existing != nil {
		return zero, fmt.Errorf("技能 %q 已存在", parsed.Slug)
	}
	if err := os.MkdirAll(v.SkillsDir(), 0o755); err != nil {
		return zero, fmt.Errorf("create skills directory: %w", err)
	}
	target := filepath.Join(v.SkillsDir(), parsed.Slug)
	if _, err := os.Stat(target); err == nil {
		return zero, fmt.Errorf("技能目录已存在：%s", target)
	}
	cacheDir := filepath.Join(v.Root, ".skillrepo", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return zero, fmt.Errorf("create cache directory: %w", err)
	}
	tempRoot, err := os.MkdirTemp(cacheDir, "import-*")
	if err != nil {
		return zero, fmt.Errorf("create import temp: %w", err)
	}
	defer os.RemoveAll(tempRoot)
	tempSkill := filepath.Join(tempRoot, parsed.Slug)
	if err := copyTree(sourceAbs, tempSkill); err != nil {
		return zero, err
	}
	hash, size, count, err := hashTree(tempSkill)
	if err != nil {
		return zero, err
	}
	if err := os.Rename(tempSkill, target); err != nil {
		return zero, fmt.Errorf("move imported skill into vault: %w", err)
	}
	now := time.Now().Unix()
	record := store.SkillRecord{
		ID:          newID(),
		Slug:        parsed.Slug,
		Name:        parsed.Meta.Name,
		Description: parsed.Meta.Description,
		Version:     parsed.Meta.Version,
		Author:      parsed.Meta.Author,
		License:     parsed.Meta.License,
		Icon:        parsed.Meta.Icon,
		Path:        filepath.ToSlash(filepath.Join("skills", parsed.Slug)),
		ContentHash: hash,
		SizeBytes:   size,
		FileCount:   count,
		Enabled:     true,
		SourceType:  "dir",
		SourceRef:   sourceAbs,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := v.Store.CreateSkill(record); err != nil {
		os.RemoveAll(target)
		return zero, err
	}
	return record, nil
}

func (v *Vault) SkillFiles(slug string) ([]FileEntry, error) {
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("技能不存在：%s", slug)
	}
	root := filepath.Join(v.Root, filepath.FromSlash(record.Path))
	var entries []FileEntry
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		entries = append(entries, FileEntry{Path: filepath.ToSlash(rel), Size: info.Size(), IsDir: entry.IsDir()})
		if len(entries) >= 500 {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list skill files: %w", err)
	}
	return entries, nil
}

func (v *Vault) SkillDir(slug string) (string, error) {
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("技能不存在：%s", slug)
	}
	return filepath.Join(v.Root, filepath.FromSlash(record.Path)), nil
}

func within(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func copyTree(source, target string) error {
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ignoredNames[entry.Name()] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := copyFile(path, dest, info.Mode().Perm()); err != nil {
			return err
		}
		return nil
	})
}

func copyFile(source, target string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", source, err)
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return fmt.Errorf("sync %s: %w", target, err)
	}
	return out.Close()
}

func hashTree(root string) (string, int64, int, error) {
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return "", 0, 0, fmt.Errorf("walk skill files: %w", err)
	}
	sort.Strings(files)
	hasher := sha256.New()
	var size int64
	for _, rel := range files {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil {
			return "", 0, 0, err
		}
		size += info.Size()
		io.WriteString(hasher, filepath.ToSlash(rel))
		io.WriteString(hasher, "\x00")
		file, err := os.Open(path)
		if err != nil {
			return "", 0, 0, err
		}
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			return "", 0, 0, err
		}
		file.Close()
		io.WriteString(hasher, "\x00")
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, len(files), nil
}

func newID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("sk_%d", time.Now().UnixNano())
	}
	return "sk_" + hex.EncodeToString(buffer)
}
