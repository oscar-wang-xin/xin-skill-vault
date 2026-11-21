package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"xinskillvault/internal/agent"
	"xinskillvault/internal/config"
	"xinskillvault/internal/distribute"
	"xinskillvault/internal/scan"
	"xinskillvault/internal/skill"
	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
	"xinskillvault/internal/version"
)

var allowedThemes = map[string]bool{
	"mech-gray": true, "business-blue": true, "mint": true,
	"alibaba-orange": true, "grape": true, "mech-amber": true,
}

type Service struct {
	config *config.Manager
	store  *store.Store
	logger *slog.Logger
}

var hostApp *application.App

// SetApplication makes the Wails app available for native dialogs without binding it to the frontend.
func SetApplication(app *application.App) {
	hostApp = app
}

type AppInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Platform      string `json:"platform"`
	ConfigPath    string `json:"configPath"`
	DataDir       string `json:"dataDir"`
	VaultPath     string `json:"vaultPath"`
	PromptRoot    string `json:"promptRoot"`
	DatabasePath  string `json:"databasePath"`
	SchemaVersion int    `json:"schemaVersion"`
}

type VaultSummary struct {
	VaultPath       string `json:"vaultPath"`
	PromptRoot      string `json:"promptRoot"`
	DatabasePath    string `json:"databasePath"`
	DatabaseBytes   int64  `json:"databaseBytes"`
	SkillCount      int    `json:"skillCount"`
	TargetCount     int    `json:"targetCount"`
	AssignmentCount int    `json:"assignmentCount"`
	PromptCount     int    `json:"promptCount"`
	CategoryCount   int    `json:"categoryCount"`
	SchemaVersion   int    `json:"schemaVersion"`
}

type SaveConfigResult struct {
	RestartRequired bool `json:"restartRequired"`
}

func NewService(cfg *config.Manager, st *store.Store, logger *slog.Logger) *Service {
	return &Service{config: cfg, store: st, logger: logger}
}

func (s *Service) GetAppInfo() AppInfo {
	cfg := s.config.Current()
	schema, _ := s.store.SchemaVersion()
	return AppInfo{
		Name:          version.AppName,
		Version:       version.AppVersion,
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		ConfigPath:    s.config.Path(),
		DataDir:       filepath.Dir(s.config.Path()),
		VaultPath:     cfg.VaultPath,
		PromptRoot:    cfg.PromptRoot,
		DatabasePath:  s.store.Path(),
		SchemaVersion: schema,
	}
}

func (s *Service) GetConfig() config.Config {
	return s.config.Current()
}

func (s *Service) SaveConfig(next config.Config) (SaveConfigResult, error) {
	previous := s.config.Current()
	if !allowedThemes[next.Theme] {
		return SaveConfigResult{}, fmt.Errorf("unknown theme %q", next.Theme)
	}
	if err := EnsureLayout(next); err != nil {
		return SaveConfigResult{}, err
	}
	if err := s.config.Save(next); err != nil {
		return SaveConfigResult{}, err
	}
	result := SaveConfigResult{RestartRequired: previous.VaultPath != next.VaultPath}
	if s.logger != nil {
		s.logger.Info("config saved", "restart_required", result.RestartRequired)
	}
	return result, nil
}

func (s *Service) GetVaultSummary() (VaultSummary, error) {
	cfg := s.config.Current()
	skills, err := s.store.SkillCount()
	if err != nil {
		return VaultSummary{}, err
	}
	targets, err := s.store.TargetCount()
	if err != nil {
		return VaultSummary{}, err
	}
	assignments, err := s.store.AssignmentCount()
	if err != nil {
		return VaultSummary{}, err
	}
	var prompts, categories, schema int
	if err := s.store.DB().QueryRow("SELECT COUNT(*) FROM prompts").Scan(&prompts); err != nil {
		return VaultSummary{}, fmt.Errorf("count prompts: %w", err)
	}
	if err := s.store.DB().QueryRow("SELECT COUNT(*) FROM prompt_categories").Scan(&categories); err != nil {
		return VaultSummary{}, fmt.Errorf("count prompt categories: %w", err)
	}
	if err := s.store.DB().QueryRow("PRAGMA user_version").Scan(&schema); err != nil {
		return VaultSummary{}, fmt.Errorf("read schema version: %w", err)
	}
	var size int64
	if info, err := os.Stat(s.store.Path()); err == nil {
		size = info.Size()
	}
	return VaultSummary{
		VaultPath:       cfg.VaultPath,
		PromptRoot:      cfg.PromptRoot,
		DatabasePath:    s.store.Path(),
		DatabaseBytes:   size,
		SkillCount:      skills,
		TargetCount:     targets,
		AssignmentCount: assignments,
		PromptCount:     prompts,
		CategoryCount:   categories,
		SchemaVersion:   schema,
	}, nil
}

func (s *Service) OpenVaultFolder() error {
	return openFolder(s.config.Current().VaultPath)
}

func (s *Service) OpenPromptFolder() error {
	return openFolder(s.config.Current().PromptRoot)
}

func (s *Service) OpenConfigFolder() error {
	return openFolder(filepath.Dir(s.config.Path()))
}

func openFolder(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create folder: %w", err)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open folder: %w", err)
	}
	return nil
}

// Timestamp is exposed for frontend boot diagnostics.
func (s *Service) Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

type SkillItem struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Version       string   `json:"version"`
	Author        string   `json:"author"`
	License       string   `json:"license"`
	Icon          string   `json:"icon"`
	Path          string   `json:"path"`
	SizeBytes     int64    `json:"sizeBytes"`
	FileCount     int      `json:"fileCount"`
	Enabled       bool     `json:"enabled"`
	AssignedCount int      `json:"assignedCount"`
	SourceType    string   `json:"sourceType"`
	ContentHash   string   `json:"contentHash"`
	UpdatedAt     int64    `json:"updatedAt"`
	Tags          []string `json:"tags"`
}

type SkillDetail struct {
	SkillItem
	Body          string            `json:"body"`
	SourceRef     string            `json:"sourceRef"`
	Files         []vault.FileEntry `json:"files"`
	SnapshotCount int               `json:"snapshotCount"`
}

type SnapshotItem struct {
	ID          string `json:"id"`
	ContentHash string `json:"contentHash"`
	FilePath    string `json:"filePath"`
	Note        string `json:"note"`
	CreatedAt   int64  `json:"createdAt"`
}

func (s *Service) ListSkills() ([]SkillItem, error) {
	records, err := s.store.ListSkills()
	if err != nil {
		return nil, err
	}
	items := make([]SkillItem, 0, len(records))
	for _, record := range records {
		items = append(items, skillItem(record))
	}
	return items, nil
}

func (s *Service) GetSkillDetail(slug string) (*SkillDetail, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("技能不存在：%s", slug)
	}
	item := skillItem(*record)
	detail := &SkillDetail{SkillItem: item}
	parsed, err := skill.ParseDir(filepath.Join(s.config.Current().VaultPath, filepath.FromSlash(record.Path)))
	if err == nil {
		detail.Body = parsed.Body
		detail.Tags = parsed.Meta.Tags
	}
	files, err := vault.New(s.config.Current().VaultPath, s.store).SkillFiles(slug)
	if err != nil {
		return nil, err
	}
	detail.Files = files
	if snapshots, err := vault.New(s.config.Current().VaultPath, s.store).Snapshots(slug); err == nil {
		detail.SnapshotCount = len(snapshots)
	}
	return detail, nil
}

func (s *Service) ImportSkillDirectory(source string) (SkillItem, error) {
	cfg := s.config.Current()
	if err := EnsureLayout(cfg); err != nil {
		return SkillItem{}, err
	}
	record, err := vault.New(cfg.VaultPath, s.store).ImportDirectory(source)
	if err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("skill imported", "slug", record.Slug, "source", record.SourceRef)
	}
	return skillItem(record), nil
}

// PickSkillArchive opens the native file picker for .zip / .skill packages.
func (s *Service) PickSkillArchive() (string, error) {
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	path, err := hostApp.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Skill 包 (*.zip, *.skill)", "*.zip;*.skill").
		SetTitle("选择技能压缩包").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("选择压缩包失败：%w", err)
	}
	return path, nil
}

func (s *Service) ImportSkillArchive(path string) (SkillItem, error) {
	cfg := s.config.Current()
	if err := EnsureLayout(cfg); err != nil {
		return SkillItem{}, err
	}
	record, err := vault.New(cfg.VaultPath, s.store).ImportArchive(path)
	if err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("skill archive imported", "slug", record.Slug, "source", record.SourceRef)
	}
	return skillItem(record), nil
}

func (s *Service) ReadSkillDocument(slug string) (string, error) {
	return vault.New(s.config.Current().VaultPath, s.store).ReadDocument(slug)
}

func (s *Service) SaveSkillDocument(slug, content, note string) (SkillItem, error) {
	if strings.TrimSpace(content) == "" {
		return SkillItem{}, fmt.Errorf("SKILL.md 不能为空")
	}
	record, err := vault.New(s.config.Current().VaultPath, s.store).SaveDocument(slug, content, note)
	if err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("skill document saved", "slug", slug)
	}
	return skillItem(record), nil
}

func (s *Service) ListSkillSnapshots(slug string) ([]SnapshotItem, error) {
	records, err := vault.New(s.config.Current().VaultPath, s.store).Snapshots(slug)
	if err != nil {
		return nil, err
	}
	items := make([]SnapshotItem, 0, len(records))
	for _, record := range records {
		items = append(items, SnapshotItem{
			ID: record.ID, ContentHash: record.ContentHash,
			FilePath: record.FilePath, Note: record.Note, CreatedAt: record.CreatedAt,
		})
	}
	return items, nil
}

// PickSkillDirectory opens the native folder picker; an empty string means cancelled.
func (s *Service) PickSkillDirectory() (string, error) {
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	path, err := hostApp.Dialog.OpenFile().
		CanChooseFiles(false).
		CanChooseDirectories(true).
		SetTitle("选择包含 SKILL.md 的技能目录").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("选择目录失败：%w", err)
	}
	return path, nil
}

func (s *Service) OpenSkillFolder(slug string) error {
	dir, err := vault.New(s.config.Current().VaultPath, s.store).SkillDir(slug)
	if err != nil {
		return err
	}
	return openFolder(dir)
}

func skillItem(record store.SkillRecord) SkillItem {
	return SkillItem{
		ID:            record.ID,
		Slug:          record.Slug,
		Name:          record.Name,
		Description:   record.Description,
		Version:       record.Version,
		Author:        record.Author,
		License:       record.License,
		Icon:          record.Icon,
		Path:          record.Path,
		SizeBytes:     record.SizeBytes,
		FileCount:     record.FileCount,
		Enabled:       record.Enabled,
		AssignedCount: record.AssignedCount,
		SourceType:    record.SourceType,
		ContentHash:   record.ContentHash,
		UpdatedAt:     record.UpdatedAt,
		Tags:          []string{},
	}
}

func (s *Service) SetSkillEnabled(slug string, enabled bool) (SkillItem, error) {
	record, err := vault.New(s.config.Current().VaultPath, s.store).SetEnabled(slug, enabled)
	if err != nil {
		return SkillItem{}, err
	}
	return skillItem(record), nil
}

func (s *Service) DeleteSkill(slug string) error {
	trashPath, err := vault.New(s.config.Current().VaultPath, s.store).Trash(slug)
	if err != nil {
		return err
	}
	if s.logger != nil {
		s.logger.Info("skill moved to trash", "slug", slug, "path", trashPath)
	}
	return nil
}

func (s *Service) RestoreSkillSnapshot(slug, snapshotID string) (SkillItem, error) {
	record, err := vault.New(s.config.Current().VaultPath, s.store).RestoreSnapshot(slug, snapshotID)
	if err != nil {
		return SkillItem{}, err
	}
	return skillItem(record), nil
}

func (s *Service) DeleteSkillSnapshot(slug, snapshotID string) error {
	return vault.New(s.config.Current().VaultPath, s.store).DeleteSnapshot(slug, snapshotID)
}

func (s *Service) ScanAgents() (scan.Result, error) {
	return scan.Scan(agent.Defaults(), s.store), nil
}

func (s *Service) ImportScannedSkill(agentID, source string) (SkillItem, error) {
	adapters := agent.Defaults()
	var found *agent.Adapter
	for i := range adapters {
		if adapters[i].ID == agentID {
			found = &adapters[i]
			break
		}
	}
	if found == nil || !found.Detected {
		return SkillItem{}, fmt.Errorf("Agent 不存在或未检测到")
	}
	abs, err := filepath.Abs(source)
	if err != nil {
		return SkillItem{}, err
	}
	if !withinPath(found.Path, abs) {
		return SkillItem{}, fmt.Errorf("技能路径不在该 Agent 目录内")
	}
	cfg := s.config.Current()
	record, err := vault.New(cfg.VaultPath, s.store).ImportDirectory(abs)
	if err != nil {
		return SkillItem{}, err
	}
	record.SourceType = "scan"
	record.SourceRef = abs
	if err := s.store.UpdateSkill(record); err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("scanned skill imported", "slug", record.Slug, "agent", agentID)
	}
	return skillItem(record), nil
}

func (s *Service) OpenAgentFolder(agentID string) error {
	for _, adapter := range agent.Defaults() {
		if adapter.ID == agentID {
			if !adapter.Detected {
				return fmt.Errorf("Agent 目录不存在")
			}
			return openFolder(adapter.Path)
		}
	}
	return fmt.Errorf("Agent 不存在")
}

func withinPath(root, target string) bool {
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

type DistributionTarget struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Short         string `json:"short"`
	Color         string `json:"color"`
	Path          string `json:"path"`
	DefaultMode   string `json:"defaultMode"`
	Detected      bool   `json:"detected"`
	Installed     bool   `json:"installed"`
	InstalledMode string `json:"installedMode"`
	InstalledPath string `json:"installedPath"`
}

type SkillAssignment struct {
	ID            string `json:"id"`
	AgentID       string `json:"agentId"`
	TargetName    string `json:"targetName"`
	Short         string `json:"short"`
	Color         string `json:"color"`
	Mode          string `json:"mode"`
	InstalledPath string `json:"installedPath"`
	Status        string `json:"status"`
	UpdatedAt     int64  `json:"updatedAt"`
}

func (s *Service) ensureTargets() (map[string]store.TargetRecord, error) {
	out := map[string]store.TargetRecord{}
	for _, adapter := range agent.Defaults() {
		record, err := s.store.FindTargetByAgent(adapter.ID)
		if err != nil {
			return nil, err
		}
		target := store.TargetRecord{
			ID: "agent:" + adapter.ID, Kind: "agent", AgentID: adapter.ID, Name: adapter.Name,
			Path: adapter.Path, ConfigPath: adapter.ConfigPath, Platform: runtime.GOOS,
			Enabled: true, Detected: adapter.Detected, Writable: true,
			DefaultMode: adapter.Mode, CreatedAt: time.Now().Unix(),
		}
		if record != nil {
			target.ID = record.ID
			target.CreatedAt = record.CreatedAt
		}
		if err := s.store.UpsertTarget(target); err != nil {
			return nil, err
		}
		out[adapter.ID] = target
	}
	return out, nil
}

func (s *Service) ListDistributionTargets(slug string) ([]DistributionTarget, error) {
	targets, err := s.ensureTargets()
	if err != nil {
		return nil, err
	}
	var record *store.SkillRecord
	if slug != "" {
		record, err = s.store.FindSkillBySlug(slug)
		if err != nil {
			return nil, err
		}
	}
	assignments := map[string]store.AssignmentRecord{}
	if record != nil {
		list, err := s.store.ListAssignments(record.ID)
		if err != nil {
			return nil, err
		}
		for _, assignment := range list {
			assignments[assignment.TargetID] = assignment
		}
	}
	adapters := agent.Defaults()
	out := make([]DistributionTarget, 0, len(adapters))
	for _, adapter := range adapters {
		target, ok := targets[adapter.ID]
		if !ok {
			continue
		}
		item := DistributionTarget{
			ID: adapter.ID, Name: adapter.Name, Short: adapter.Short, Color: adapter.Color,
			Path: adapter.Path, DefaultMode: adapter.Mode, Detected: adapter.Detected,
		}
		if assignment, ok := assignments[target.ID]; ok {
			item.Installed = true
			item.InstalledMode = assignment.Mode
			item.InstalledPath = assignment.InstalledPath
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) DistributeSkill(slug, agentID, mode string) (SkillAssignment, error) {
	targets, err := s.ensureTargets()
	if err != nil {
		return SkillAssignment{}, err
	}
	target, ok := targets[agentID]
	if !ok {
		return SkillAssignment{}, fmt.Errorf("Agent 不存在")
	}
	if !target.Detected {
		return SkillAssignment{}, fmt.Errorf("Agent 目录未检测到，无法分发")
	}
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return SkillAssignment{}, err
	}
	if record == nil {
		return SkillAssignment{}, fmt.Errorf("技能不存在：%s", slug)
	}
	assignment, err := (&distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}).Install(target, slug, mode)
	if err != nil {
		return SkillAssignment{}, err
	}
	return s.assignmentDTO(record, target, assignment), nil
}

func (s *Service) UninstallSkill(slug, agentID string) error {
	targets, err := s.ensureTargets()
	if err != nil {
		return err
	}
	target, ok := targets[agentID]
	if !ok {
		return fmt.Errorf("Agent 不存在")
	}
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return err
	}
	if record == nil {
		return fmt.Errorf("技能不存在：%s", slug)
	}
	return (&distribute.Distributor{Vault: vault.New(s.config.Current().VaultPath, s.store), Store: s.store}).Uninstall(record.ID, target.ID)
}

func (s *Service) ListSkillAssignments(slug string) ([]SkillAssignment, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("技能不存在：%s", slug)
	}
	targets, err := s.ensureTargets()
	if err != nil {
		return nil, err
	}
	byTarget := map[string]store.TargetRecord{}
	adapters := agent.Defaults()
	for _, adapter := range adapters {
		if target, ok := targets[adapter.ID]; ok {
			byTarget[target.ID] = target
		}
	}
	list, err := s.store.ListAssignments(record.ID)
	if err != nil {
		return nil, err
	}
	out := make([]SkillAssignment, 0, len(list))
	for _, assignment := range list {
		target, ok := byTarget[assignment.TargetID]
		if !ok {
			continue
		}
		out = append(out, s.assignmentDTO(record, target, assignment))
	}
	return out, nil
}

func (s *Service) assignmentDTO(record *store.SkillRecord, target store.TargetRecord, assignment store.AssignmentRecord) SkillAssignment {
	adapter := agent.Adapter{ID: target.AgentID, Name: target.Name}
	for _, item := range agent.Defaults() {
		if item.ID == target.AgentID {
			adapter = item
			break
		}
	}
	return SkillAssignment{
		ID: assignment.ID, AgentID: adapter.ID, TargetName: adapter.Name,
		Short: adapter.Short, Color: adapter.Color, Mode: assignment.Mode,
		InstalledPath: assignment.InstalledPath, Status: assignment.Status, UpdatedAt: assignment.UpdatedAt,
	}
}

func (s *Service) UpdateScannedSkill(agentID, source string) (SkillItem, error) {
	adapters := agent.Defaults()
	var found *agent.Adapter
	for i := range adapters {
		if adapters[i].ID == agentID {
			found = &adapters[i]
			break
		}
	}
	if found == nil || !found.Detected {
		return SkillItem{}, fmt.Errorf("Agent 不存在或未检测到")
	}
	abs, err := filepath.Abs(source)
	if err != nil {
		return SkillItem{}, err
	}
	if !withinPath(found.Path, abs) {
		return SkillItem{}, fmt.Errorf("技能路径不在该 Agent 目录内")
	}
	parsed, err := skill.ParseDir(abs)
	if err != nil {
		return SkillItem{}, err
	}
	if record, err := s.store.FindSkillBySlug(parsed.Slug); err != nil {
		return SkillItem{}, err
	} else if record == nil {
		return SkillItem{}, fmt.Errorf("技能尚未导入，请先导入")
	}
	record, err := vault.New(s.config.Current().VaultPath, s.store).ReplaceFromDirectory(
		parsed.Slug, abs, "scan", abs, "扫描更新前",
	)
	if err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("scanned skill updated", "slug", record.Slug, "agent", agentID)
	}
	return skillItem(record), nil
}
