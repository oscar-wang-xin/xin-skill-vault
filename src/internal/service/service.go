package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/xin-skill-vault/xin-skill-vault/internal/netimport"
	"github.com/xin-skill-vault/xin-skill-vault/internal/store"
	"github.com/xin-skill-vault/xin-skill-vault/internal/vault"
)

const (
	SettingSkin            = "ui.skin"
	SettingLayout          = "ui.layout"
	SettingLanguage        = "ui.language"
	SettingFontFamily      = "ui.fontFamily"
	SettingFontScale       = "ui.fontScale"
	SettingDefaultMode     = "install.defaultMode"
	SettingCheckOnStart    = "updates.checkOnStart"
	SettingAutoUpdateLinks = "updates.autoUpdateLinks"
	SettingCloseToTray     = "window.closeToTray"
	SettingCloseAction     = "window.closeAction"
	SettingVaultPath       = "vault.path"
	SettingLastScanAt      = "vault.lastScanAt"
)

var allowedSkins = map[string]bool{
	"cold-white":  true,
	"cold-gray":   true,
	"mech-yellow": true,
	"paper":       true,
	"paper-dark":  true,
	"system":      true,
}

var allowedFontFamilies = map[string]bool{
	"system": true,
	"segoe":  true,
	"yahei":  true,
	"noto":   true,
}

type AppService struct {
	Store   *store.Store
	Vault   *vault.Manager
	DataDir string
	DBPath  string
	Version string

	Window             application.Window
	ForceQuit          *atomic.Bool
	ClosePromptPending *atomic.Bool
}

type Stats struct {
	Skills         int   `json:"skills"`
	Links          int   `json:"links"`
	Copies         int   `json:"copies"`
	EnabledAgents  int   `json:"enabledAgents"`
	DetectedAgents int   `json:"detectedAgents"`
	Broken         int   `json:"broken"`
	Updates        int   `json:"updates"`
	VaultSizeBytes int64 `json:"vaultSizeBytes"`
}

type VaultInfo struct {
	Path          string `json:"path"`
	DBPath        string `json:"dbPath"`
	SQLiteVersion string `json:"sqliteVersion"`
	AppVersion    string `json:"appVersion"`
	Platform      string `json:"platform"`
	SchemaVersion int    `json:"schemaVersion"`
	DBSizeBytes   int64  `json:"dbSizeBytes"`
	LastScanAt    string `json:"lastScanAt"`
}

type SkillView struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Notes           string   `json:"notes"`
	Tags            []string `json:"tags"`
	GroupID         string   `json:"groupId"`
	GroupName       string   `json:"groupName"`
	SourceID        string   `json:"sourceId"`
	SourceName      string   `json:"sourceName"`
	VaultPath       string   `json:"vaultPath"`
	ContentHash     string   `json:"contentHash"`
	Content         string   `json:"content"`
	ContentFiles    []string `json:"contentFiles"`
	Version         string   `json:"version"`
	SizeBytes       int64    `json:"sizeBytes"`
	FileCount       int      `json:"fileCount"`
	InstalledAt     string   `json:"installedAt"`
	UpdatedAt       string   `json:"updatedAt"`
	UpdateAvailable bool     `json:"updateAvailable"`
	Agents          []string `json:"agents"`
	LinkCount       int      `json:"linkCount"`
	CopyCount       int      `json:"copyCount"`
	BrokenCount     int      `json:"brokenCount"`
}

type AgentView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	SkillsPath string `json:"skillsPath"`
	ConfigPath string `json:"configPath"`
	Detected   bool   `json:"detected"`
	Enabled    bool   `json:"enabled"`
	Installed  int    `json:"installed"`
}

type GroupView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SkillCount int    `json:"skillCount"`
	SortOrder  int    `json:"sortOrder"`
}

type SourceView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	URL        string `json:"url"`
	Builtin    bool   `json:"builtin"`
	CreatedAt  string `json:"createdAt"`
	SyncedAt   string `json:"syncedAt"`
	SkillCount int    `json:"skillCount"`
}

type Snapshot struct {
	Skills        []SkillView          `json:"skills"`
	Agents        []AgentView          `json:"agents"`
	Sources       []SourceView         `json:"sources"`
	Groups        []GroupView          `json:"groups"`
	Distributions []store.Distribution `json:"distributions"`
	Activity      []store.Activity     `json:"activity"`
	Settings      map[string]string    `json:"settings"`
	Stats         Stats                `json:"stats"`
	Vault         VaultInfo            `json:"vault"`
	GeneratedAt   string               `json:"generatedAt"`
}

type MutationResult struct {
	Snapshot Snapshot `json:"snapshot"`
	Warnings []string `json:"warnings"`
	Message  string   `json:"message"`
}

type ImportItem struct {
	Path      string `json:"path"`
	SourceID  string `json:"sourceId"`
	SourceRef string `json:"sourceRef"`
}

type ImportRequest struct {
	Path     string   `json:"path"`
	AgentIDs []string `json:"agentIds"`
	Mode     string   `json:"mode"`
}

type DistributeRequest struct {
	SkillID  string   `json:"skillId"`
	AgentIDs []string `json:"agentIds"`
	Mode     string   `json:"mode"`
}

type ImportManyRequest struct {
	Items    []ImportItem `json:"items"`
	AgentIDs []string     `json:"agentIds"`
	Mode     string       `json:"mode"`
}

type DistributeManyRequest struct {
	SkillIDs []string `json:"skillIds"`
	AgentIDs []string `json:"agentIds"`
	Mode     string   `json:"mode"`
}

type SourceInput struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type ProjectInput struct {
	Name       string `json:"name"`
	SkillsPath string `json:"skillsPath"`
}

type SkillMetaUpdate struct {
	SkillID string   `json:"skillId"`
	Notes   string   `json:"notes"`
	GroupID string   `json:"groupId"`
	Tags    []string `json:"tags"`
}

type DirectoryView struct {
	Target   AgentView              `json:"target"`
	Path     string                 `json:"path"`
	Entries  []vault.DirectoryEntry `json:"entries"`
	Managed  int                    `json:"managed"`
	External int                    `json:"external"`
	Broken   int                    `json:"broken"`
}

func (s *AppService) Snapshot() (Snapshot, error) { return s.buildSnapshot() }

func (s *AppService) Refresh() (MutationResult, error) {
	res, err := s.Vault.Scan()
	if err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	msg := fmt.Sprintf("索引已刷新，共 %d 个技能", res.Scanned)
	return MutationResult{Snapshot: snap, Message: msg}, nil
}

func (s *AppService) CheckUpdates() (MutationResult, error) {
	skills, _ := s.Store.ListSkills()
	ids := make([]string, 0, len(skills))
	for _, sk := range skills {
		ids = append(ids, sk.ID)
	}
	if err := s.Store.TouchChecked(ids); err != nil {
		return MutationResult{}, err
	}
	_ = s.Store.SetSetting("updates.lastCheckAt", time.Now().Format(time.RFC3339))
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: "已检查全部技能，当前没有可用更新"}, nil
}

func (s *AppService) SaveSetting(key, value string) (Snapshot, error) {
	if !isAllowedSetting(key) {
		return Snapshot{}, fmt.Errorf("未知设置项: %s", key)
	}
	if key == SettingSkin && !allowedSkins[value] {
		return Snapshot{}, fmt.Errorf("未知皮肤: %s", value)
	}
	if key == SettingFontFamily && !allowedFontFamilies[value] {
		return Snapshot{}, fmt.Errorf("未知字体: %s", value)
	}
	if key == SettingFontScale {
		scale, err := strconv.ParseFloat(value, 64)
		if err != nil || scale < 0.85 || scale > 1.5 {
			return Snapshot{}, fmt.Errorf("字号缩放必须在 0.85 到 1.5 之间")
		}
	}
	if key == SettingCloseAction && value != "ask" && value != "quit" && value != "tray" {
		return Snapshot{}, fmt.Errorf("关闭行为必须是 ask、quit 或 tray")
	}
	if err := s.Store.SetSetting(key, value); err != nil {
		return Snapshot{}, err
	}
	return s.buildSnapshot()
}

func (s *AppService) SetSkin(id string) (Snapshot, error) {
	if !allowedSkins[id] {
		return Snapshot{}, fmt.Errorf("未知皮肤: %s", id)
	}
	if err := s.Store.SetSetting(SettingSkin, id); err != nil {
		return Snapshot{}, err
	}
	return s.buildSnapshot()
}

func (s *AppService) SetAgentEnabled(agentID string, enabled bool) (MutationResult, error) {
	agent, err := s.Store.GetAgent(agentID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("Agent 不存在")
	}
	if enabled {
		if err := os.MkdirAll(agent.SkillsPath, 0o755); err != nil {
			return MutationResult{}, fmt.Errorf("启用 %s 失败: %w", agent.Name, err)
		}
		_ = s.Store.MarkAgentDetected(agent.ID, true)
	}
	if err := s.Store.SetAgentEnabled(agentID, enabled); err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	msg := fmt.Sprintf("已停用 %s", agent.Name)
	if enabled {
		msg = fmt.Sprintf("已启用 %s", agent.Name)
	}
	return MutationResult{Snapshot: snap, Message: msg}, nil
}

func (s *AppService) PickFolder() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("窗口尚未就绪")
	}
	path, err := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择包含 SKILL.md 的技能目录").
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *AppService) PickVaultFolder() (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("窗口尚未就绪")
	}
	path, err := app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择新的技能仓库目录").
		SetDirectory(s.Vault.Root).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	return path, nil
}

// ChangeVaultPath switches the central vault to a new directory and either
// moves or copies the existing skills into it. Linked distributions are
// recreated against the new vault; copy distributions stay independent.
func (s *AppService) ChangeVaultPath(newPath, mode string) (MutationResult, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "move" && mode != "copy" {
		return MutationResult{}, fmt.Errorf("请选择移动或复制")
	}
	newPath = strings.TrimSpace(newPath)
	if newPath == "" {
		return MutationResult{}, fmt.Errorf("请选择新的仓库目录")
	}
	if strings.HasPrefix(newPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return MutationResult{}, fmt.Errorf("无法定位用户目录: %w", err)
		}
		if newPath == "~" {
			newPath = home
		} else {
			newPath = filepath.Join(home, strings.TrimLeft(newPath[1:], "\\/"))
		}
	}
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return MutationResult{}, fmt.Errorf("无效的目录: %w", err)
	}
	abs = filepath.Clean(abs)
	oldRoot := filepath.Clean(s.Vault.Root)
	if sameFilesystemPath(oldRoot, abs) {
		return MutationResult{}, fmt.Errorf("新目录与当前仓库目录相同")
	}
	if pathWithin(oldRoot, abs) {
		return MutationResult{}, fmt.Errorf("新目录不能位于当前仓库内部")
	}
	if info, statErr := os.Lstat(abs); statErr == nil {
		if vault.IsLink(abs) {
			return MutationResult{}, fmt.Errorf("不能使用链接目录作为仓库位置")
		}
		if !info.IsDir() {
			return MutationResult{}, fmt.Errorf("目标路径不是目录")
		}
		entries, readErr := os.ReadDir(abs)
		if readErr != nil {
			return MutationResult{}, fmt.Errorf("读取目标目录失败: %w", readErr)
		}
		if len(entries) > 0 {
			return MutationResult{}, fmt.Errorf("目标目录必须为空，请选择空目录或新目录")
		}
	} else if !os.IsNotExist(statErr) {
		return MutationResult{}, fmt.Errorf("检查目标目录失败: %w", statErr)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return MutationResult{}, fmt.Errorf("创建目标目录失败: %w", err)
	}

	warnings := []string{}
	oldSkills := s.Vault.SkillsDir()
	newSkills := filepath.Join(abs, "skills")
	migrated := false
	if info, statErr := os.Stat(oldSkills); statErr == nil && info.IsDir() {
		if mode == "move" {
			if renameErr := os.Rename(oldSkills, newSkills); renameErr == nil {
				migrated = true
			} else if copyErr := vault.CopyDir(oldSkills, newSkills); copyErr != nil {
				return MutationResult{}, fmt.Errorf("移动技能失败: %w", copyErr)
			} else {
				if removeErr := os.RemoveAll(oldSkills); removeErr != nil {
					warnings = append(warnings, fmt.Sprintf("旧目录清理失败：%v", removeErr))
				}
				migrated = true
			}
		} else if copyErr := vault.CopyDir(oldSkills, newSkills); copyErr != nil {
			return MutationResult{}, fmt.Errorf("复制技能失败: %w", copyErr)
		} else {
			migrated = true
		}
	}

	if err := s.Store.SetSetting(SettingVaultPath, abs); err != nil {
		if mode == "move" && migrated {
			if rollbackErr := os.Rename(newSkills, oldSkills); rollbackErr != nil {
				warnings = append(warnings, fmt.Sprintf("设置保存失败，且回滚目录失败：%v", rollbackErr))
			}
		}
		return MutationResult{}, fmt.Errorf("保存仓库位置失败: %w", err)
	}

	s.Vault.Root = abs
	if err := s.Vault.Ensure(); err != nil {
		return MutationResult{}, fmt.Errorf("初始化新仓库失败: %w", err)
	}
	if _, err := s.Vault.Scan(); err != nil {
		return MutationResult{}, fmt.Errorf("扫描新仓库失败: %w", err)
	}

	relinkWarnings, relinkErr := s.relinkLinkedDistributions()
	warnings = append(warnings, relinkWarnings...)
	if relinkErr != nil {
		warnings = append(warnings, fmt.Sprintf("部分分发需要手动修复：%v", relinkErr))
	}

	msg := "仓库位置已切换，原技能已复制到新目录"
	if mode == "move" {
		msg = "仓库位置已切换，原技能已移动到新目录"
	}
	if len(warnings) > 0 {
		msg = fmt.Sprintf("%s（%d 条提示）", msg, len(warnings))
	}
	s.Store.AddActivity("vault", "仓库位置", fmt.Sprintf("%s · %s", abs, map[string]string{"move": "移动", "copy": "复制"}[mode]))
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Warnings: warnings, Message: msg}, nil
}

type relinkTarget struct {
	SkillID string
	AgentID string
}

func (s *AppService) relinkLinkedDistributions() ([]string, error) {
	dists, err := s.Store.ListDistributions()
	if err != nil {
		return nil, err
	}
	links := make([]relinkTarget, 0, len(dists))
	for _, d := range dists {
		if d.Mode != "copy" {
			links = append(links, relinkTarget{SkillID: d.SkillID, AgentID: d.AgentID})
		}
	}
	if len(links) == 0 {
		return nil, nil
	}
	warnings := []string{}
	ready := make([]relinkTarget, 0, len(links))
	for _, link := range links {
		if err := s.Vault.RemoveDistribution(link.SkillID, link.AgentID); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s：移除旧链接失败（%v）", link.SkillID, err))
			continue
		}
		ready = append(ready, link)
	}
	for _, link := range ready {
		if _, err := s.Vault.Distribute(link.SkillID, []string{link.AgentID}, vault.LinkMode()); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s：重建链接失败（%v）", link.SkillID, err))
		}
	}
	return warnings, nil
}

func sameFilesystemPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func pathWithin(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if runtime.GOOS == "windows" {
		rel = strings.ToLower(rel)
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func (s *AppService) ResolveClose(action string, remember bool) error {
	if action != "quit" && action != "tray" {
		return fmt.Errorf("未知关闭行为: %s", action)
	}
	if s.ClosePromptPending != nil {
		s.ClosePromptPending.Store(false)
	}
	if remember {
		if err := s.Store.SetSetting(SettingCloseAction, action); err != nil {
			return err
		}
	}
	if action == "quit" {
		if s.ForceQuit != nil {
			s.ForceQuit.Store(true)
		}
		if app := application.Get(); app != nil {
			app.Quit()
		}
		return nil
	}
	if s.Window != nil {
		s.Window.Hide()
	}
	return nil
}

func (s *AppService) CancelClose() {
	if s.ClosePromptPending != nil {
		s.ClosePromptPending.Store(false)
	}
}

func (s *AppService) InspectPath(path string) (vault.Inspection, error) {
	if strings.TrimSpace(path) == "" {
		return vault.Inspection{}, fmt.Errorf("请先选择目录")
	}
	return vault.Inspect(path)
}

func (s *AppService) Importables() ([]vault.Importable, error) {
	items, err := s.Vault.Importables()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []vault.Importable{}
	}
	return items, nil
}

func (s *AppService) DiscoverFolder(path string) ([]vault.Importable, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("请先选择文件夹")
	}
	items, err := s.Vault.DiscoverFolder(path)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []vault.Importable{}
	}
	return items, nil
}

func (s *AppService) NetworkImport(rawURL string) ([]vault.Importable, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("请输入 GitHub、zip 或 tar.gz 地址")
	}
	s.cleanupOldImportCache()
	_ = s.Store.SeedSources([]store.Source{{ID: "network", Name: "网络导入", Kind: "url", Builtin: true}})
	session := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	dest := filepath.Join(s.DataDir, "cache", "imports", session)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	result, err := netimport.Fetch(ctx, rawURL, dest)
	if err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	scanRoot := filepath.Join(result.Root, result.SubPath)
	items, err := s.Vault.DiscoverFolder(scanRoot)
	if err != nil {
		_ = os.RemoveAll(dest)
		return nil, err
	}
	if len(items) == 0 {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("下载完成，但没有找到包含 SKILL.md 的技能")
	}
	for i := range items {
		items[i].SourceID = "network"
		items[i].SourceRef = rawURL
		items[i].Agents = []string{}
	}
	return items, nil
}

func (s *AppService) Import(req ImportRequest) (MutationResult, error) {
	return s.importItems([]ImportItem{{Path: req.Path}}, req.AgentIDs, req.Mode)
}

func (s *AppService) ImportMany(req ImportManyRequest) (MutationResult, error) {
	return s.importItems(req.Items, req.AgentIDs, req.Mode)
}

func (s *AppService) importItems(items []ImportItem, agentIDs []string, mode string) (MutationResult, error) {
	if len(items) == 0 {
		return MutationResult{}, fmt.Errorf("请先选择要导入的技能")
	}
	if mode == "" {
		mode = s.defaultMode()
	}
	warnings := []string{}
	imported := 0
	failed := 0
	network := false
	seen := map[string]bool{}
	for _, item := range items {
		path := strings.TrimSpace(item.Path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		insp, err := vault.Inspect(path)
		if err != nil {
			failed++
			warnings = append(warnings, fmt.Sprintf("%s：%v", filepath.Base(path), err))
			continue
		}
		sourceID := strings.TrimSpace(item.SourceID)
		if sourceID == "" {
			sourceID = "local"
		}
		sourceRef := strings.TrimSpace(item.SourceRef)
		if sourceID == "network" {
			network = true
		}
		skill, err := s.Vault.ImportWithSource(path, sourceID, sourceRef)
		if err != nil {
			failed++
			warnings = append(warnings, fmt.Sprintf("「%s」：%v", insp.Name, err))
			continue
		}
		imported++
		if len(agentIDs) > 0 {
			distWarnings, distErr := s.Vault.Distribute(skill.ID, agentIDs, mode)
			if distErr != nil {
				warnings = append(warnings, fmt.Sprintf("%s：分发失败（%v）", skill.Name, distErr))
			} else {
				warnings = append(warnings, distWarnings...)
			}
		}
	}
	if imported == 0 && failed > 0 {
		return MutationResult{}, fmt.Errorf("没有技能被导入：%s", warnings[0])
	}
	if network {
		s.cleanupImportCache()
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	msg := fmt.Sprintf("已导入 %d 个技能", imported)
	if failed > 0 {
		msg += fmt.Sprintf("，%d 个失败", failed)
	}
	if len(agentIDs) > 0 && imported > 0 {
		msg += fmt.Sprintf("，分发到 %d 个目标", len(agentIDs))
	}
	return MutationResult{Snapshot: snap, Warnings: warnings, Message: msg}, nil
}

func (s *AppService) cleanupImportCache() {
	_ = os.RemoveAll(filepath.Join(s.DataDir, "cache", "imports"))
}

func (s *AppService) cleanupOldImportCache() {
	root := filepath.Join(s.DataDir, "cache", "imports")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
}

func (s *AppService) Distribute(req DistributeRequest) (MutationResult, error) {
	if len(req.AgentIDs) == 0 {
		return MutationResult{}, fmt.Errorf("请至少选择一个目标")
	}
	if req.Mode == "" {
		req.Mode = s.defaultMode()
	}
	warnings, err := s.Vault.Distribute(req.SkillID, req.AgentIDs, req.Mode)
	if err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{
		Snapshot: snap,
		Warnings: warnings,
		Message:  fmt.Sprintf("已分发到 %d 个目标", len(req.AgentIDs)-len(warnings)),
	}, nil
}

func (s *AppService) DistributeMany(req DistributeManyRequest) (MutationResult, error) {
	skillIDs := dedupeStrings(req.SkillIDs)
	agentIDs := dedupeStrings(req.AgentIDs)
	if len(skillIDs) == 0 {
		return MutationResult{}, fmt.Errorf("请至少选择一个技能")
	}
	if len(agentIDs) == 0 {
		return MutationResult{}, fmt.Errorf("请至少选择一个分发目标")
	}
	if req.Mode == "" {
		req.Mode = s.defaultMode()
	}
	warnings := []string{}
	success := 0
	for _, skillID := range skillIDs {
		w, err := s.Vault.Distribute(skillID, agentIDs, req.Mode)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s：%v", skillID, err))
			continue
		}
		warnings = append(warnings, w...)
		success++
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{
		Snapshot: snap,
		Warnings: warnings,
		Message:  fmt.Sprintf("已处理 %d 个技能 × %d 个目标", success, len(agentIDs)),
	}, nil
}

func (s *AppService) UpdateSkillMeta(req SkillMetaUpdate) (Snapshot, error) {
	skill, err := s.Store.GetSkill(req.SkillID)
	if err != nil {
		return Snapshot{}, fmt.Errorf("技能不存在")
	}
	_ = skill
	if req.GroupID != "" {
		if _, err := s.Store.GetSkillGroup(req.GroupID); err != nil {
			return Snapshot{}, fmt.Errorf("分组不存在")
		}
	}
	notes := strings.TrimSpace(req.Notes)
	if len([]rune(notes)) > 8000 {
		return Snapshot{}, fmt.Errorf("备注过长（最多 8000 字）")
	}
	if err := s.Store.UpdateSkillMeta(req.SkillID, notes, req.GroupID, normalizeTags(req.Tags)); err != nil {
		return Snapshot{}, err
	}
	s.Store.AddActivity("edit", req.SkillID, "备注 / 标签 / 分组")
	return s.buildSnapshot()
}

func (s *AppService) CreateGroup(name string) (Snapshot, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Snapshot{}, fmt.Errorf("分组名称不能为空")
	}
	if len([]rune(name)) > 48 {
		return Snapshot{}, fmt.Errorf("分组名称最多 48 字")
	}
	groups, _ := s.Store.ListSkillGroups()
	for _, group := range groups {
		if strings.EqualFold(group.Name, name) {
			return Snapshot{}, fmt.Errorf("分组「%s」已存在", name)
		}
	}
	id := s.uniqueGroupID(name)
	if err := s.Store.CreateSkillGroup(store.SkillGroup{
		ID: id, Name: name, SortOrder: len(groups) + 1, CreatedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return Snapshot{}, err
	}
	s.Store.AddActivity("group", name, "创建分组")
	return s.buildSnapshot()
}

func (s *AppService) RenameGroup(id, name string) (Snapshot, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Snapshot{}, fmt.Errorf("分组名称不能为空")
	}
	groups, _ := s.Store.ListSkillGroups()
	for _, group := range groups {
		if group.ID != id && strings.EqualFold(group.Name, name) {
			return Snapshot{}, fmt.Errorf("分组「%s」已存在", name)
		}
	}
	if err := s.Store.RenameSkillGroup(id, name); err != nil {
		return Snapshot{}, err
	}
	return s.buildSnapshot()
}

func (s *AppService) DeleteGroup(id string) (Snapshot, error) {
	group, err := s.Store.GetSkillGroup(id)
	if err != nil {
		return Snapshot{}, fmt.Errorf("分组不存在")
	}
	if err := s.Store.DeleteSkillGroup(id); err != nil {
		return Snapshot{}, err
	}
	s.Store.AddActivity("group", group.Name, "删除分组")
	return s.buildSnapshot()
}

func (s *AppService) AddSource(req SourceInput) (MutationResult, error) {
	name := strings.TrimSpace(req.Name)
	kind := strings.TrimSpace(req.Kind)
	rawURL := strings.TrimSpace(req.URL)
	if name == "" {
		return MutationResult{}, fmt.Errorf("来源名称不能为空")
	}
	if len([]rune(name)) > 64 {
		return MutationResult{}, fmt.Errorf("来源名称最多 64 字")
	}
	switch kind {
	case "git", "registry", "local", "url":
	default:
		return MutationResult{}, fmt.Errorf("不支持的来源类型")
	}
	if rawURL == "" {
		return MutationResult{}, fmt.Errorf("请填写来源地址或本地路径")
	}
	id := s.uniqueSourceID(name)
	if err := s.Store.AddSource(store.Source{
		ID: id, Name: name, Kind: kind, URL: rawURL, CreatedAt: time.Now().Format(time.RFC3339),
	}); err != nil {
		return MutationResult{}, err
	}
	s.Store.AddActivity("source", name, "手动添加来源")
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已添加来源「%s」", name)}, nil
}

func (s *AppService) RemoveSource(id string) (MutationResult, error) {
	src, err := s.Store.GetSource(id)
	if err != nil {
		return MutationResult{}, fmt.Errorf("来源不存在")
	}
	if src.Builtin {
		return MutationResult{}, fmt.Errorf("内置来源不能移除")
	}
	count, err := s.Store.CountSkillsBySource(id)
	if err != nil {
		return MutationResult{}, err
	}
	if count > 0 {
		return MutationResult{}, fmt.Errorf("该来源下还有 %d 个技能，不能移除", count)
	}
	if err := s.Store.DeleteSource(id); err != nil {
		return MutationResult{}, err
	}
	s.Store.AddActivity("source", src.Name, "移除来源")
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已移除来源「%s」", src.Name)}, nil
}

func (s *AppService) AddProject(req ProjectInput) (MutationResult, error) {
	name := strings.TrimSpace(req.Name)
	skillsPath := strings.TrimSpace(req.SkillsPath)
	if name == "" || skillsPath == "" {
		return MutationResult{}, fmt.Errorf("请填写项目名称和技能目录")
	}
	abs, err := filepath.Abs(skillsPath)
	if err != nil {
		return MutationResult{}, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return MutationResult{}, fmt.Errorf("创建项目技能目录失败: %w", err)
	}
	id := s.uniqueAgentID("project-" + slugifyLabel(name))
	if err := s.Store.AddAgent(store.Agent{
		ID: id, Name: name, Kind: "project", SkillsPath: abs, Detected: true,
		Enabled: true, SortOrder: 1000,
	}); err != nil {
		return MutationResult{}, fmt.Errorf("添加项目目录失败: %w", err)
	}
	s.Store.AddActivity("project", name, abs)
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已添加项目目录「%s」", name)}, nil
}

func (s *AppService) RemoveProject(id string) (MutationResult, error) {
	target, err := s.Store.GetAgent(id)
	if err != nil {
		return MutationResult{}, fmt.Errorf("项目目录不存在")
	}
	if target.Kind != "project" {
		return MutationResult{}, fmt.Errorf("只能移除手动添加的项目目录")
	}
	count, err := s.Store.CountDistributionsForAgent(id)
	if err != nil {
		return MutationResult{}, err
	}
	if count > 0 {
		return MutationResult{}, fmt.Errorf("该项目目录下还有 %d 条分发，请先移除", count)
	}
	if err := s.Store.DeleteAgent(id); err != nil {
		return MutationResult{}, err
	}
	s.Store.AddActivity("project", target.Name, "移除项目目录")
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已移除项目目录「%s」", target.Name)}, nil
}

func (s *AppService) DirectorySkills(targetID string) (DirectoryView, error) {
	target, err := s.Store.GetAgent(targetID)
	if err != nil {
		return DirectoryView{}, fmt.Errorf("目标不存在")
	}
	entries, err := s.Vault.DirectoryEntries(target)
	if err != nil {
		return DirectoryView{}, err
	}
	if entries == nil {
		entries = []vault.DirectoryEntry{}
	}
	view := DirectoryView{
		Target: AgentView{
			ID: target.ID, Name: target.Name, Kind: target.Kind, SkillsPath: target.SkillsPath,
			ConfigPath: target.ConfigPath, Detected: target.Detected, Enabled: target.Enabled,
		},
		Path: target.SkillsPath, Entries: entries,
	}
	for _, entry := range entries {
		if entry.Managed {
			view.Managed++
		} else {
			view.External++
		}
		if entry.Status == "broken" || entry.Status == "drifted" {
			view.Broken++
		}
	}
	return view, nil
}

func (s *AppService) RemoveDirectoryEntry(targetID, name string) (MutationResult, error) {
	target, err := s.Store.GetAgent(targetID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("目标不存在")
	}
	if err := s.Vault.RemoveDirectoryEntry(target, name); err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已从 %s 移除「%s」", target.Name, name)}, nil
}

func (s *AppService) RebuildDirectoryDistribution(targetID, skillID, mode string) (MutationResult, error) {
	dist, err := s.Store.GetDistribution(skillID, targetID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("该分发不存在")
	}
	if mode == "" {
		mode = s.defaultMode()
	}
	if mode == dist.Mode {
		return MutationResult{}, fmt.Errorf("已经是该模式")
	}
	if err := s.Vault.RemoveDistribution(skillID, targetID); err != nil {
		return MutationResult{}, err
	}
	warnings, err := s.Vault.Distribute(skillID, []string{targetID}, mode)
	if err != nil {
		_, _ = s.Vault.Distribute(skillID, []string{targetID}, dist.Mode)
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{
		Snapshot: snap, Warnings: warnings,
		Message: fmt.Sprintf("已切换为 %s", mode),
	}, nil
}

func (s *AppService) OpenDirectoryEntry(targetID, name string) error {
	target, err := s.Store.GetAgent(targetID)
	if err != nil {
		return fmt.Errorf("目标不存在")
	}
	path, err := s.Vault.DirectoryEntryPath(target, name)
	if err != nil {
		return err
	}
	return openInFileManager(path)
}

func (s *AppService) SetDistribution(skillID, agentID string, enabled bool) (MutationResult, error) {
	if enabled {
		warnings, err := s.Vault.Distribute(skillID, []string{agentID}, s.defaultMode())
		if err != nil {
			return MutationResult{}, err
		}
		snap, err := s.buildSnapshot()
		if err != nil {
			return MutationResult{}, err
		}
		msg := "分发已建立"
		if len(warnings) > 0 {
			msg = warnings[0]
		}
		return MutationResult{Snapshot: snap, Warnings: warnings, Message: msg}, nil
	}
	if err := s.Vault.RemoveDistribution(skillID, agentID); err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: "分发已移除"}, nil
}

func (s *AppService) RemoveSkill(skillID string) (MutationResult, error) {
	skill, err := s.Store.GetSkill(skillID)
	if err != nil {
		return MutationResult{}, fmt.Errorf("技能不存在")
	}
	if err := s.Vault.RemoveSkill(skillID); err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: fmt.Sprintf("已移除「%s」，目录已移入回收区", skill.Name)}, nil
}

func (s *AppService) Repair() (MutationResult, error) {
	if err := s.Vault.CheckHealth(); err != nil {
		return MutationResult{}, err
	}
	snap, err := s.buildSnapshot()
	if err != nil {
		return MutationResult{}, err
	}
	return MutationResult{Snapshot: snap, Message: "健康检查完成"}, nil
}

func (s *AppService) SkillFiles(skillID string) ([]vault.FileEntry, error) {
	files, err := s.Vault.AllSkillFiles(skillID)
	if err != nil {
		return nil, err
	}
	if files == nil {
		files = []vault.FileEntry{}
	}
	return files, nil
}

func (s *AppService) OpenVault() error {
	return openInFileManager(s.Vault.Root)
}

func (s *AppService) OpenDataFolder() error {
	return openInFileManager(s.DataDir)
}

func (s *AppService) OpenSkillFolder(skillID string) error {
	skill, err := s.Store.GetSkill(skillID)
	if err != nil {
		return fmt.Errorf("技能不存在")
	}
	return openInFileManager(skill.VaultPath)
}

func (s *AppService) OpenAgentFolder(agentID string) error {
	agent, err := s.Store.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("Agent 不存在")
	}
	if err := os.MkdirAll(agent.SkillsPath, 0o755); err != nil {
		return err
	}
	return openInFileManager(agent.SkillsPath)
}

// ---------- snapshot ----------

func (s *AppService) buildSnapshot() (Snapshot, error) {
	skills, err := s.Store.ListSkills()
	if err != nil {
		return Snapshot{}, err
	}
	agents, err := s.Store.ListAgents()
	if err != nil {
		return Snapshot{}, err
	}
	sources, err := s.Store.ListSources()
	if err != nil {
		return Snapshot{}, err
	}
	dists, err := s.Store.ListDistributions()
	if err != nil {
		return Snapshot{}, err
	}
	groups, err := s.Store.ListSkillGroups()
	if err != nil {
		return Snapshot{}, err
	}
	tags := s.Store.AllSkillTags()
	activity, _ := s.Store.RecentActivity(14)
	settings := s.Store.AllSettings()
	if skills == nil {
		skills = []store.Skill{}
	}
	if agents == nil {
		agents = []store.Agent{}
	}
	if sources == nil {
		sources = []store.Source{}
	}
	if dists == nil {
		dists = []store.Distribution{}
	}
	if groups == nil {
		groups = []store.SkillGroup{}
	}
	if activity == nil {
		activity = []store.Activity{}
	}

	agentName := map[string]string{}
	agentInstalled := map[string]int{}
	agentViews := make([]AgentView, 0, len(agents))
	for _, a := range agents {
		agentName[a.ID] = a.Name
		agentViews = append(agentViews, AgentView{
			ID: a.ID, Name: a.Name, Kind: a.Kind, SkillsPath: a.SkillsPath, ConfigPath: a.ConfigPath,
			Detected: a.Detected, Enabled: a.Enabled,
		})
	}
	groupName := map[string]string{}
	groupCount := map[string]int{}
	for _, group := range groups {
		groupName[group.ID] = group.Name
	}
	sourceName := map[string]string{}
	sourceCount := map[string]int{}
	for _, src := range sources {
		sourceName[src.ID] = src.Name
	}
	skillDists := map[string][]store.Distribution{}
	for _, d := range dists {
		skillDists[d.SkillID] = append(skillDists[d.SkillID], d)
		agentInstalled[d.AgentID]++
	}
	for i := range agentViews {
		agentViews[i].Installed = agentInstalled[agentViews[i].ID]
	}

	skillViews := make([]SkillView, 0, len(skills))
	stats := Stats{Skills: len(skills)}
	for _, sk := range skills {
		sv := SkillView{
			ID: sk.ID, Name: sk.Name, Description: sk.Description,
			Notes: sk.Notes, Tags: tags[sk.ID], GroupID: sk.GroupID, GroupName: groupName[sk.GroupID],
			SourceID: sk.SourceID, SourceName: sourceName[sk.SourceID],
			VaultPath: sk.VaultPath, ContentHash: sk.ContentHash, Content: sk.Content,
			ContentFiles: sk.ContentFiles,
			Version:      sk.Version, SizeBytes: sk.SizeBytes, FileCount: sk.FileCount,
			InstalledAt: sk.InstalledAt, UpdatedAt: sk.UpdatedAt,
			UpdateAvailable: sk.UpdateAvailable, Agents: []string{},
		}
		if sv.Tags == nil {
			sv.Tags = []string{}
		}
		if sv.ContentFiles == nil {
			sv.ContentFiles = []string{}
		}
		if sk.GroupID != "" {
			groupCount[sk.GroupID]++
		}
		if sv.SourceName == "" {
			sv.SourceName = "本地导入"
		}
		for _, d := range skillDists[sk.ID] {
			sv.Agents = append(sv.Agents, d.AgentID)
			if d.Mode == "copy" {
				sv.CopyCount++
				stats.Copies++
			} else {
				sv.LinkCount++
				stats.Links++
			}
			if d.Status != "ok" {
				sv.BrokenCount++
				stats.Broken++
			}
		}
		if sk.UpdateAvailable {
			stats.Updates++
		}
		sourceCount[sk.SourceID]++
		stats.VaultSizeBytes += sk.SizeBytes
		skillViews = append(skillViews, sv)
	}

	sourceViews := make([]SourceView, 0, len(sources))
	for _, src := range sources {
		sourceViews = append(sourceViews, SourceView{
			ID: src.ID, Name: src.Name, Kind: src.Kind, URL: src.URL,
			Builtin: src.Builtin, CreatedAt: src.CreatedAt, SyncedAt: src.SyncedAt,
			SkillCount: sourceCount[src.ID],
		})
	}
	groupViews := make([]GroupView, 0, len(groups))
	for _, group := range groups {
		groupViews = append(groupViews, GroupView{
			ID: group.ID, Name: group.Name, SortOrder: group.SortOrder,
			SkillCount: groupCount[group.ID],
		})
	}
	for _, a := range agentViews {
		if a.Enabled {
			stats.EnabledAgents++
		}
		if a.Detected {
			stats.DetectedAgents++
		}
	}

	sort.Slice(skillViews, func(i, j int) bool {
		return strings.ToLower(skillViews[i].Name) < strings.ToLower(skillViews[j].Name)
	})
	info, _ := s.info(settings)
	return Snapshot{
		Skills: skillViews, Agents: agentViews, Sources: sourceViews, Groups: groupViews,
		Distributions: dists, Activity: activity, Settings: settings,
		Stats: stats, Vault: info, GeneratedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (s *AppService) info(settings map[string]string) (VaultInfo, error) {
	info := VaultInfo{
		Path: s.Vault.Root, DBPath: s.DBPath, AppVersion: s.Version,
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		SchemaVersion: s.Store.SchemaVersion(),
		SQLiteVersion: s.Store.SQLiteVersion(),
		LastScanAt:    settings[SettingLastScanAt],
	}
	if fi, err := os.Stat(s.DBPath); err == nil {
		info.DBSizeBytes = fi.Size()
	}
	return info, nil
}

func (s *AppService) defaultMode() string {
	if m, ok := s.Store.GetSetting(SettingDefaultMode); ok && m != "" {
		return m
	}
	if runtime.GOOS == "windows" {
		return "junction"
	}
	return "symlink"
}

func isAllowedSetting(key string) bool {
	switch key {
	case SettingSkin, SettingLayout, SettingLanguage, SettingFontFamily, SettingFontScale,
		SettingDefaultMode,
		SettingCheckOnStart,
		SettingAutoUpdateLinks, SettingCloseToTray, SettingCloseAction:
		return true
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if tag == "" {
			continue
		}
		runes := []rune(tag)
		if len(runes) > 32 {
			tag = string(runes[:32])
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
		if len(out) >= 30 {
			break
		}
	}
	return out
}

func slugifyLabel(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r > 127:
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '\\' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "item"
	}
	runes := []rune(out)
	if len(runes) > 48 {
		out = strings.Trim(string(runes[:48]), "-")
	}
	return out
}

func (s *AppService) uniqueGroupID(name string) string {
	base := slugifyLabel(name)
	groups, _ := s.Store.ListSkillGroups()
	used := map[string]bool{}
	for _, group := range groups {
		used[group.ID] = true
	}
	id := base
	for i := 2; used[id]; i++ {
		id = fmt.Sprintf("%s-%d", base, i)
	}
	return id
}

func (s *AppService) uniqueSourceID(name string) string {
	base := slugifyLabel(name)
	sources, _ := s.Store.ListSources()
	used := map[string]bool{}
	for _, source := range sources {
		used[source.ID] = true
	}
	id := base
	for i := 2; used[id]; i++ {
		id = fmt.Sprintf("%s-%d", base, i)
	}
	return id
}

func (s *AppService) uniqueAgentID(base string) string {
	agents, _ := s.Store.ListAgents()
	used := map[string]bool{}
	for _, agent := range agents {
		used[agent.ID] = true
	}
	if base == "" {
		base = "target"
	}
	id := base
	for i := 2; used[id]; i++ {
		id = fmt.Sprintf("%s-%d", base, i)
	}
	return id
}

func openInFileManager(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("目录不存在: %s", path)
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("无法打开目录: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
