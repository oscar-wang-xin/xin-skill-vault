package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xinskillvault/internal/store"
)

type PromptCategoryItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parentId"`
	DirPath   string `json:"dirPath"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt int64  `json:"createdAt"`
}

type PromptItem struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	CategoryID     string   `json:"categoryId"`
	CategoryName   string   `json:"categoryName"`
	Tags           []string `json:"tags"`
	CurrentVersion int      `json:"currentVersion"`
	ContentHash    string   `json:"contentHash"`
	CreatedAt      int64    `json:"createdAt"`
	UpdatedAt      int64    `json:"updatedAt"`
}

type PromptVersionItem struct {
	Version      int      `json:"version"`
	Title        string   `json:"title"`
	CategoryID   string   `json:"categoryId"`
	CategoryName string   `json:"categoryName"`
	Tags         []string `json:"tags"`
	Note         string   `json:"note"`
	CreatedAt    int64    `json:"createdAt"`
}

type PromptDetail struct {
	PromptItem
	Content  string              `json:"content"`
	Versions []PromptVersionItem `json:"versions"`
}

func (s *Service) promptRoot() (string, error) {
	root := s.config.Current().PromptRoot
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("提示词目录未配置")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create prompt root: %w", err)
	}
	return root, nil
}

func (s *Service) promptCategories() (map[string]store.PromptCategoryRecord, error) {
	records, err := s.store.ListPromptCategories()
	if err != nil {
		return nil, err
	}
	out := map[string]store.PromptCategoryRecord{}
	for _, record := range records {
		out[record.ID] = record
	}
	return out, nil
}

func (s *Service) promptCategoryItem(record store.PromptCategoryRecord) PromptCategoryItem {
	return PromptCategoryItem{
		ID: record.ID, Name: record.Name, ParentID: record.ParentID,
		DirPath: record.DirPath, SortOrder: record.SortOrder, CreatedAt: record.CreatedAt,
	}
}

func (s *Service) ListPromptCategories() ([]PromptCategoryItem, error) {
	records, err := s.store.ListPromptCategories()
	if err != nil {
		return nil, err
	}
	out := make([]PromptCategoryItem, 0, len(records))
	for _, record := range records {
		out = append(out, s.promptCategoryItem(record))
	}
	return out, nil
}

func safePromptDir(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" {
		return "", nil
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) {
		return "", fmt.Errorf("目录必须是提示词根目录下的相对路径")
	}
	return clean, nil
}

func sanitizePromptDirName(name string) string {
	name = strings.TrimSpace(name)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	name = replacer.Replace(name)
	if name == "" || name == "." || name == ".." {
		return "category"
	}
	return name
}

func (s *Service) ensurePromptCategoryDir(relative string) (string, error) {
	root, err := s.promptRoot()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(relative)), 0o755); err != nil {
		return "", fmt.Errorf("create prompt category dir: %w", err)
	}
	return filepath.Join(root, filepath.FromSlash(relative)), nil
}

func (s *Service) CreatePromptCategory(name, parentID, dirPath string) (PromptCategoryItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PromptCategoryItem{}, fmt.Errorf("分类名称不能为空")
	}
	parent := ""
	if parentID != "" {
		record, err := s.store.FindPromptCategory(parentID)
		if err != nil {
			return PromptCategoryItem{}, err
		}
		if record == nil {
			return PromptCategoryItem{}, fmt.Errorf("父分类不存在")
		}
		parent = record.DirPath
	}
	relative, err := safePromptDir(dirPath)
	if err != nil {
		return PromptCategoryItem{}, err
	}
	if relative == "" {
		relative = filepath.ToSlash(filepath.Join(parent, sanitizePromptDirName(name)))
	}
	if _, err := s.ensurePromptCategoryDir(relative); err != nil {
		return PromptCategoryItem{}, err
	}
	record := store.PromptCategoryRecord{
		ID: newPromptCategoryID(), Name: name, ParentID: parentID, DirPath: relative, CreatedAt: time.Now().Unix(),
	}
	if err := s.store.CreatePromptCategory(record); err != nil {
		return PromptCategoryItem{}, err
	}
	return s.promptCategoryItem(record), nil
}

func (s *Service) UpdatePromptCategory(id, name string) (PromptCategoryItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return PromptCategoryItem{}, fmt.Errorf("分类名称不能为空")
	}
	record, err := s.store.FindPromptCategory(id)
	if err != nil {
		return PromptCategoryItem{}, err
	}
	if record == nil {
		return PromptCategoryItem{}, fmt.Errorf("分类不存在")
	}
	if err := s.store.UpdatePromptCategory(id, name); err != nil {
		return PromptCategoryItem{}, err
	}
	record.Name = name
	return s.promptCategoryItem(*record), nil
}

func (s *Service) DeletePromptCategory(id string) error {
	return s.store.DeletePromptCategory(id)
}

func (s *Service) promptCategoryNameMap() (map[string]string, error) {
	records, err := s.store.ListPromptCategories()
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, record := range records {
		out[record.ID] = record.Name
	}
	return out, nil
}

func (s *Service) promptItem(record store.PromptRecord, names map[string]string) PromptItem {
	return PromptItem{
		ID: record.ID, Title: record.Title, CategoryID: record.CategoryID,
		CategoryName: names[record.CategoryID], Tags: record.Tags,
		CurrentVersion: record.CurrentVersion, ContentHash: record.CurrentHash,
		CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}
}

func (s *Service) ListPrompts(category string) ([]PromptItem, error) {
	records, err := s.store.ListPrompts(category)
	if err != nil {
		return nil, err
	}
	names, err := s.promptCategoryNameMap()
	if err != nil {
		return nil, err
	}
	out := make([]PromptItem, 0, len(records))
	for _, record := range records {
		out = append(out, s.promptItem(record, names))
	}
	return out, nil
}

func promptFilePath(relativeDir, promptID string, version int) string {
	return filepath.ToSlash(filepath.Join(relativeDir, fmt.Sprintf("%s-v%d.md", promptID, version)))
}

func (s *Service) categoryDir(categoryID string) (string, error) {
	if categoryID == "" {
		return "", nil
	}
	record, err := s.store.FindPromptCategory(categoryID)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("分类不存在")
	}
	return record.DirPath, nil
}

func (s *Service) writePromptFile(relative, content string) (string, error) {
	root, err := s.promptRoot()
	if err != nil {
		return "", err
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".prompt-*.tmp")
	if err != nil {
		return "", err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.WriteString(content); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(name, target); err != nil {
		return "", err
	}
	return relative, nil
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (s *Service) CreatePrompt(title, categoryID string, tags []string, content string) (PromptDetail, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return PromptDetail{}, fmt.Errorf("提示词标题不能为空")
	}
	dir, err := s.categoryDir(categoryID)
	if err != nil {
		return PromptDetail{}, err
	}
	id := newPromptID()
	now := time.Now().Unix()
	version := 1
	relative := promptFilePath(dir, id, version)
	if _, err := s.writePromptFile(relative, content); err != nil {
		return PromptDetail{}, err
	}
	record := store.PromptRecord{
		ID: id, Title: title, CategoryID: categoryID, Tags: tags, CurrentVersion: version,
		FilePath: relative, CurrentHash: contentHash(content), CreatedAt: now, UpdatedAt: now,
	}
	versionRecord := store.PromptVersionRecord{
		ID: newPromptVersionID(), PromptID: id, Version: version, Title: title,
		CategoryID: categoryID, Tags: tags, FilePath: relative, ContentHash: contentHash(content),
		Note: "初始版本", CreatedAt: now,
	}
	if err := s.store.CreatePromptWithVersion(record, versionRecord); err != nil {
		return PromptDetail{}, err
	}
	return s.GetPrompt(id)
}

func (s *Service) GetPrompt(id string) (PromptDetail, error) {
	record, err := s.store.FindPrompt(id)
	if err != nil {
		return PromptDetail{}, err
	}
	if record == nil {
		return PromptDetail{}, fmt.Errorf("提示词不存在")
	}
	names, err := s.promptCategoryNameMap()
	if err != nil {
		return PromptDetail{}, err
	}
	root, err := s.promptRoot()
	if err != nil {
		return PromptDetail{}, err
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(record.FilePath)))
	if err != nil {
		return PromptDetail{}, fmt.Errorf("读取提示词文件: %w", err)
	}
	versions, err := s.store.ListPromptVersions(id)
	if err != nil {
		return PromptDetail{}, err
	}
	versionItems := make([]PromptVersionItem, 0, len(versions))
	for _, item := range versions {
		versionItems = append(versionItems, PromptVersionItem{
			Version: item.Version, Title: item.Title, CategoryID: item.CategoryID,
			CategoryName: names[item.CategoryID], Tags: item.Tags, Note: item.Note, CreatedAt: item.CreatedAt,
		})
	}
	return PromptDetail{
		PromptItem: s.promptItem(*record, names), Content: string(content), Versions: versionItems,
	}, nil
}

func (s *Service) SavePrompt(id, title, categoryID string, tags []string, content string) (PromptDetail, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return PromptDetail{}, fmt.Errorf("提示词标题不能为空")
	}
	record, err := s.store.FindPrompt(id)
	if err != nil || record == nil {
		return PromptDetail{}, fmt.Errorf("提示词不存在")
	}
	dir, err := s.categoryDir(categoryID)
	if err != nil {
		return PromptDetail{}, err
	}
	next := record.CurrentVersion + 1
	relative := promptFilePath(dir, id, next)
	if _, err := s.writePromptFile(relative, content); err != nil {
		return PromptDetail{}, err
	}
	now := time.Now().Unix()
	updated := *record
	updated.Title = title
	updated.CategoryID = categoryID
	updated.Tags = tags
	updated.CurrentVersion = next
	updated.FilePath = relative
	updated.CurrentHash = contentHash(content)
	updated.UpdatedAt = now
	version := store.PromptVersionRecord{
		ID: newPromptVersionID(), PromptID: id, Version: next, Title: title,
		CategoryID: categoryID, Tags: tags, FilePath: relative, ContentHash: contentHash(content),
		Note: "保存新版本", CreatedAt: now,
	}
	if err := s.store.AddPromptVersion(updated, version); err != nil {
		return PromptDetail{}, err
	}
	return s.GetPrompt(id)
}

func (s *Service) GetPromptVersion(id string, version int) (string, error) {
	record, err := s.store.FindPromptVersion(id, version)
	if err != nil || record == nil {
		return "", fmt.Errorf("版本不存在")
	}
	root, err := s.promptRoot()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(record.FilePath)))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) DeletePromptVersion(id string, version int) (PromptDetail, error) {
	record, err := s.store.FindPrompt(id)
	if err != nil || record == nil {
		return PromptDetail{}, fmt.Errorf("提示词不存在")
	}
	versions, err := s.store.ListPromptVersions(id)
	if err != nil {
		return PromptDetail{}, err
	}
	if len(versions) <= 1 {
		return PromptDetail{}, fmt.Errorf("至少保留一个版本")
	}
	target, err := s.store.FindPromptVersion(id, version)
	if err != nil || target == nil {
		return PromptDetail{}, fmt.Errorf("版本不存在")
	}
	root, _ := s.promptRoot()
	_ = os.Remove(filepath.Join(root, filepath.FromSlash(target.FilePath)))
	if err := s.store.DeletePromptVersion(id, version); err != nil {
		return PromptDetail{}, err
	}
	if record.CurrentVersion == version {
		remaining, err := s.store.ListPromptVersions(id)
		if err != nil || len(remaining) == 0 {
			return PromptDetail{}, fmt.Errorf("没有可切换的版本")
		}
		content, err := s.GetPromptVersion(id, remaining[0].Version)
		if err != nil {
			return PromptDetail{}, err
		}
		target = &remaining[0]
		target.FilePath = remaining[0].FilePath
		if err := s.store.SetPromptCurrent(id, *target, time.Now().Unix()); err != nil {
			return PromptDetail{}, err
		}
		_ = content
	}
	return s.GetPrompt(id)
}

func (s *Service) DeletePrompt(id string) error {
	record, err := s.store.FindPrompt(id)
	if err != nil || record == nil {
		return fmt.Errorf("提示词不存在")
	}
	versions, _ := s.store.ListPromptVersions(id)
	root, _ := s.promptRoot()
	for _, version := range versions {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(version.FilePath)))
	}
	return s.store.DeletePrompt(id)
}

func newPromptID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pm_%d", time.Now().UnixNano())
	}
	return "pm_" + hex.EncodeToString(buf)
}

func newPromptVersionID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pv_%d", time.Now().UnixNano())
	}
	return "pv_" + hex.EncodeToString(buf)
}

func newPromptCategoryID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pc_%d", time.Now().UnixNano())
	}
	return "pc_" + hex.EncodeToString(buf)
}
