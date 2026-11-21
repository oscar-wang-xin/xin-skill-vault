package vault

import (
	"archive/zip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
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

// ImportArchive extracts a .zip/.skill archive and imports the directory containing SKILL.md.
func (v *Vault) ImportArchive(archivePath string) (store.SkillRecord, error) {
	var zero store.SkillRecord
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		return zero, fmt.Errorf("resolve archive: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return zero, fmt.Errorf("read archive: %w", err)
	}
	if info.IsDir() {
		return zero, fmt.Errorf("archive path is a directory")
	}
	if info.Size() > 200*1024*1024 {
		return zero, fmt.Errorf("压缩包超过 200MB 限制")
	}
	reader, err := zip.OpenReader(abs)
	if err != nil {
		return zero, fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()
	tempRoot, err := os.MkdirTemp("", "xsv-zip-*")
	if err != nil {
		return zero, fmt.Errorf("create archive temp: %w", err)
	}
	defer os.RemoveAll(tempRoot)
	var docs []string
	var total uint64
	for _, entry := range reader.File {
		name, err := safeArchiveName(entry.Name)
		if err != nil {
			return zero, err
		}
		total += entry.UncompressedSize64
		if total > 512*1024*1024 {
			return zero, fmt.Errorf("压缩包解压后超过 512MB 限制")
		}
		if strings.Contains(name, ":") {
			return zero, fmt.Errorf("压缩包包含不安全路径：%s", entry.Name)
		}
		target := filepath.Join(tempRoot, filepath.FromSlash(name))
		if !within(tempRoot, target) {
			return zero, fmt.Errorf("压缩包包含越界路径：%s", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return zero, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return zero, err
		}
		if err := extractZipFile(entry, target); err != nil {
			return zero, err
		}
		if strings.EqualFold(path.Base(name), skill.DocName) {
			docs = append(docs, target)
		}
	}
	if len(docs) == 0 {
		return zero, fmt.Errorf("压缩包里没有找到 SKILL.md")
	}
	sort.Slice(docs, func(i, j int) bool {
		return strings.Count(docs[i], string(filepath.Separator)) < strings.Count(docs[j], string(filepath.Separator))
	})
	bestDepth := strings.Count(docs[0], string(filepath.Separator))
	if len(docs) > 1 && strings.Count(docs[1], string(filepath.Separator)) == bestDepth {
		return zero, fmt.Errorf("压缩包里存在多个 SKILL.md，无法确定导入根目录")
	}
	record, err := v.ImportDirectory(filepath.Dir(docs[0]))
	if err != nil {
		return zero, err
	}
	record.SourceType = "zip"
	record.SourceRef = abs
	if err := v.Store.UpdateSkill(record); err != nil {
		return zero, err
	}
	return record, nil
}

func safeArchiveName(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	clean := path.Clean(strings.TrimPrefix(name, "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("压缩包包含不安全路径：%s", name)
	}
	return clean, nil
}

func extractZipFile(entry *zip.File, target string) error {
	in, err := entry.Open()
	if err != nil {
		return fmt.Errorf("open archive entry %s: %w", entry.Name, err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", target, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("extract %s: %w", entry.Name, err)
	}
	return out.Close()
}

func (v *Vault) ReadDocument(slug string) (string, error) {
	dir, err := v.SkillDir(slug)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, skill.DocName))
	if err != nil {
		return "", fmt.Errorf("read SKILL.md: %w", err)
	}
	return string(data), nil
}

// SaveDocument snapshots the previous SKILL.md, writes the new document and updates the index.
func (v *Vault) SaveDocument(slug, content, note string) (store.SkillRecord, error) {
	var zero store.SkillRecord
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return zero, err
	}
	if record == nil {
		return zero, fmt.Errorf("技能不存在：%s", slug)
	}
	meta, _, err := skill.ParseDocument(content)
	if err != nil {
		return zero, err
	}
	nextSlug, err := skill.ValidateMetadata(meta)
	if err != nil {
		return zero, err
	}
	if nextSlug != record.Slug {
		return zero, fmt.Errorf("不能在编辑时修改 slug")
	}
	dir, err := v.SkillDir(slug)
	if err != nil {
		return zero, err
	}
	docPath := filepath.Join(dir, skill.DocName)
	oldContent, err := os.ReadFile(docPath)
	if err != nil {
		return zero, fmt.Errorf("read current SKILL.md: %w", err)
	}
	if string(oldContent) == content {
		return *record, nil
	}
	snapshotDir := filepath.Join(v.Root, ".skillrepo", "snapshots", slug)
	if err := os.MkdirAll(snapshotDir, 0o700); err != nil {
		return zero, fmt.Errorf("create snapshot dir: %w", err)
	}
	snapshotID := newID()
	snapshotPath := filepath.Join(snapshotDir, snapshotID+".md")
	if err := writeAtomic(snapshotPath, oldContent, 0o600); err != nil {
		return zero, err
	}
	cleanupSnapshot := func() { _ = os.Remove(snapshotPath) }
	if err := writeAtomic(docPath, []byte(content), 0o644); err != nil {
		cleanupSnapshot()
		return zero, err
	}
	parsed, err := skill.ParseDir(dir)
	if err != nil {
		_ = writeAtomic(docPath, oldContent, 0o644)
		cleanupSnapshot()
		return zero, err
	}
	hash, size, count, err := hashTree(dir)
	if err != nil {
		_ = writeAtomic(docPath, oldContent, 0o644)
		cleanupSnapshot()
		return zero, err
	}
	updated := *record
	updated.Name = parsed.Meta.Name
	updated.Description = parsed.Meta.Description
	updated.Version = parsed.Meta.Version
	updated.Author = parsed.Meta.Author
	updated.License = parsed.Meta.License
	updated.Icon = parsed.Meta.Icon
	updated.ContentHash = hash
	updated.SizeBytes = size
	updated.FileCount = count
	updated.UpdatedAt = time.Now().Unix()
	snapshot := store.SnapshotRecord{
		ID:          snapshotID,
		SkillID:     updated.ID,
		ContentHash: record.ContentHash,
		FilePath:    snapshotPath,
		Note:        note,
		CreatedAt:   updated.UpdatedAt,
	}
	if err := v.Store.UpdateSkillWithSnapshot(updated, snapshot); err != nil {
		_ = writeAtomic(docPath, oldContent, 0o644)
		cleanupSnapshot()
		return zero, err
	}
	return updated, nil
}

func (v *Vault) Snapshots(slug string) ([]store.SnapshotRecord, error) {
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("技能不存在：%s", slug)
	}
	return v.Store.ListSnapshots(record.ID)
}

func writeAtomic(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".write-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(mode); err != nil {
		temp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	return nil
}

func (v *Vault) SetEnabled(slug string, enabled bool) (store.SkillRecord, error) {
	var zero store.SkillRecord
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return zero, err
	}
	if record == nil {
		return zero, fmt.Errorf("技能不存在：%s", slug)
	}
	if err := v.Store.SetSkillEnabled(record.ID, enabled); err != nil {
		return zero, err
	}
	record.Enabled = enabled
	record.UpdatedAt = time.Now().Unix()
	return *record, nil
}

// Trash moves a skill directory into .skillrepo/trash and marks its index row deleted.
func (v *Vault) Trash(slug string) (string, error) {
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("技能不存在：%s", slug)
	}
	dir, err := v.SkillDir(slug)
	if err != nil {
		return "", err
	}
	trashRoot := filepath.Join(v.Root, ".skillrepo", "trash")
	if err := os.MkdirAll(trashRoot, 0o755); err != nil {
		return "", fmt.Errorf("create trash dir: %w", err)
	}
	target := filepath.Join(trashRoot, fmt.Sprintf("%s-%d-%s", slug, time.Now().Unix(), newID()[3:9]))
	if err := os.Rename(dir, target); err != nil {
		return "", fmt.Errorf("move skill to trash: %w", err)
	}
	now := time.Now().Unix()
	if err := v.Store.SoftDeleteSkill(record.ID, now); err != nil {
		_ = os.Rename(target, dir)
		return "", err
	}
	return target, nil
}

func (v *Vault) RestoreSnapshot(slug, snapshotID string) (store.SkillRecord, error) {
	var zero store.SkillRecord
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return zero, err
	}
	if record == nil {
		return zero, fmt.Errorf("技能不存在：%s", slug)
	}
	snapshot, err := v.Store.FindSnapshot(record.ID, snapshotID)
	if err != nil {
		return zero, err
	}
	if snapshot == nil {
		return zero, fmt.Errorf("快照不存在")
	}
	if isZipFile(snapshot.FilePath) {
		temp, err := os.MkdirTemp("", "xsv-restore-*")
		if err != nil {
			return zero, err
		}
		defer os.RemoveAll(temp)
		if err := extractDirectoryArchive(snapshot.FilePath, temp); err != nil {
			return zero, err
		}
		parsed, err := skill.ParseDir(temp)
		if err != nil {
			return zero, err
		}
		if parsed.Slug != record.Slug {
			return zero, fmt.Errorf("快照的 slug 与当前技能不一致")
		}
		return v.ReplaceFromDirectory(slug, temp, "", "", "恢复快照")
	}
	data, err := os.ReadFile(snapshot.FilePath)
	if err != nil {
		return zero, fmt.Errorf("read snapshot: %w", err)
	}
	return v.SaveDocument(slug, string(data), "恢复快照")
}

func (v *Vault) DeleteSnapshot(slug, snapshotID string) error {
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("技能不存在：%s", slug)
	}
	snapshot, err := v.Store.FindSnapshot(record.ID, snapshotID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("快照不存在")
	}
	if err := v.Store.DeleteSnapshot(snapshot.ID); err != nil {
		return err
	}
	_ = os.Remove(snapshot.FilePath)
	return nil
}

// FingerprintDir exposes the same deterministic hash used by imports.
func FingerprintDir(root string) (string, int64, int, error) {
	return hashTree(root)
}

// CopyDir copies a skill tree into an empty target, skipping ignored metadata.
func CopyDir(source, target string) error {
	return copyTree(source, target)
}

// ReplaceFromDirectory replaces a vault skill with a directory after snapshotting the old tree.
func (v *Vault) ReplaceFromDirectory(slug, source, sourceType, sourceRef, note string) (store.SkillRecord, error) {
	var zero store.SkillRecord
	record, err := v.Store.FindSkillBySlug(slug)
	if err != nil {
		return zero, err
	}
	if record == nil {
		return zero, fmt.Errorf("技能不存在：%s", slug)
	}
	sourceAbs, err := filepath.Abs(source)
	if err != nil {
		return zero, err
	}
	if within(v.Root, sourceAbs) {
		return zero, fmt.Errorf("不能从仓库内部替换技能")
	}
	parsed, err := skill.ParseDir(sourceAbs)
	if err != nil {
		return zero, err
	}
	if parsed.Slug != record.Slug {
		return zero, fmt.Errorf("技能 slug 不一致：%s", parsed.Slug)
	}
	hash, _, _, err := hashTree(sourceAbs)
	if err != nil {
		return zero, err
	}
	if hash == record.ContentHash {
		return *record, nil
	}
	dir, err := v.SkillDir(slug)
	if err != nil {
		return zero, err
	}
	if _, err := os.Stat(dir); err != nil {
		return zero, fmt.Errorf("仓库技能目录不存在: %w", err)
	}
	snapshotDir := filepath.Join(v.Root, ".skillrepo", "snapshots", slug)
	if err := os.MkdirAll(snapshotDir, 0o700); err != nil {
		return zero, fmt.Errorf("create snapshot dir: %w", err)
	}
	snapshotID := newID()
	snapshotPath := filepath.Join(snapshotDir, snapshotID+".zip")
	if err := zipDirectory(dir, snapshotPath); err != nil {
		return zero, err
	}
	cleanupSnapshot := func() { _ = os.Remove(snapshotPath) }
	cacheDir := filepath.Join(v.Root, ".skillrepo", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		cleanupSnapshot()
		return zero, err
	}
	tempNew, err := os.MkdirTemp(cacheDir, "replace-new-*")
	if err != nil {
		cleanupSnapshot()
		return zero, err
	}
	cleanupNew := func() { _ = os.RemoveAll(tempNew) }
	if err := copyTree(sourceAbs, tempNew); err != nil {
		cleanupNew()
		cleanupSnapshot()
		return zero, err
	}
	backup := filepath.Join(cacheDir, "replace-old-"+newID())
	if err := os.Rename(dir, backup); err != nil {
		cleanupNew()
		cleanupSnapshot()
		return zero, fmt.Errorf("move old skill aside: %w", err)
	}
	restoreOld := func() {
		_ = os.RemoveAll(dir)
		_ = os.Rename(backup, dir)
	}
	if err := os.Rename(tempNew, dir); err != nil {
		restoreOld()
		cleanupSnapshot()
		return zero, fmt.Errorf("move replacement into vault: %w", err)
	}
	parsedNext, err := skill.ParseDir(dir)
	if err != nil {
		restoreOld()
		_ = os.RemoveAll(dir)
		cleanupSnapshot()
		return zero, err
	}
	nextHash, nextSize, nextCount, err := hashTree(dir)
	if err != nil {
		restoreOld()
		cleanupSnapshot()
		return zero, err
	}
	updated := *record
	updated.Name = parsedNext.Meta.Name
	updated.Description = parsedNext.Meta.Description
	updated.Version = parsedNext.Meta.Version
	updated.Author = parsedNext.Meta.Author
	updated.License = parsedNext.Meta.License
	updated.Icon = parsedNext.Meta.Icon
	updated.ContentHash = nextHash
	updated.SizeBytes = nextSize
	updated.FileCount = nextCount
	if sourceType != "" {
		updated.SourceType = sourceType
	}
	if sourceRef != "" {
		updated.SourceRef = sourceRef
	}
	updated.UpdatedAt = time.Now().Unix()
	snapshot := store.SnapshotRecord{
		ID: snapshotID, SkillID: updated.ID, ContentHash: record.ContentHash,
		FilePath: snapshotPath, Note: note, CreatedAt: updated.UpdatedAt,
	}
	if err := v.Store.UpdateSkillWithSnapshot(updated, snapshot); err != nil {
		restoreOld()
		cleanupSnapshot()
		return zero, err
	}
	_ = os.RemoveAll(backup)
	return updated, nil
}

func isZipFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 4)
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}
	return header[0] == 'P' && header[1] == 'K' && header[2] == 3 && header[3] == 4
}

func zipDirectory(source, dest string) error {
	file, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create snapshot archive: %w", err)
	}
	archive := zip.NewWriter(file)
	writeErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
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
		if rel == "." || entry.IsDir() {
			return nil
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
		writer, err := archive.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if writeErr != nil {
		archive.Close()
		file.Close()
		_ = os.Remove(dest)
		return fmt.Errorf("write snapshot archive: %w", writeErr)
	}
	if err := archive.Close(); err != nil {
		file.Close()
		_ = os.Remove(dest)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(dest)
		return err
	}
	return nil
}

func extractDirectoryArchive(archivePath, dest string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open snapshot archive: %w", err)
	}
	defer reader.Close()
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	for _, entry := range reader.File {
		name, err := safeArchiveName(entry.Name)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		if !within(dest, target) {
			return fmt.Errorf("快照包含越界路径：%s", entry.Name)
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
	return nil
}
