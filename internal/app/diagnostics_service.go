package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"xinskillvault/internal/diagnostics"
)

func (s *Service) ExportDiagnostics() (string, error) {
	info := s.GetAppInfo()
	cfg := s.config.Current()
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	destination, err := hostApp.Dialog.SaveFile().
		SetFilename("xin-skill-vault-diagnostics.zip").
		AddFilter("Diagnostics (*.zip)", "*.zip").
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(destination) == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(destination), ".zip") {
		destination += ".zip"
	}
	report := diagnostics.Report{
		AppName: info.Name, AppVersion: info.Version, Platform: info.Platform,
		ConfigPath: info.ConfigPath, VaultPath: info.VaultPath, PromptRoot: info.PromptRoot,
		DatabasePath: info.DatabasePath, SchemaVersion: info.SchemaVersion,
		LogPath: filepath.Join(cfg.VaultPath, ".skillrepo", "logs", "app.log"),
	}
	if err := diagnostics.Export(destination, report, cfg, report.LogPath); err != nil {
		return "", err
	}
	return destination, nil
}
