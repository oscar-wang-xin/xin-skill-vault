package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"xinskillvault/internal/config"
	"xinskillvault/internal/store"
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

func (s *Service) Close() error {
	return s.store.Close()
}

// Timestamp is exposed for frontend boot diagnostics.
func (s *Service) Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
