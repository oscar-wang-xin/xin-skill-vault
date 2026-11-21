package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"xinskillvault/internal/skillpack"
	"xinskillvault/internal/vault"
)

func (s *Service) ExportSkillPackage(slug string) (string, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("技能不存在：%s", slug)
	}
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	defaultName := skillpack.SafeFilename(record.Slug) + ".skill"
	destination, err := hostApp.Dialog.SaveFile().
		SetFilename(defaultName).
		AddFilter("Skill 包 (*.skill)", "*.skill").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("选择导出位置失败：%w", err)
	}
	if strings.TrimSpace(destination) == "" {
		return "", nil
	}
	if !strings.HasSuffix(strings.ToLower(destination), ".skill") {
		destination += ".skill"
	}
	sourceDir, err := vault.New(s.config.Current().VaultPath, s.store).SkillDir(slug)
	if err != nil {
		return "", err
	}
	if _, err := filepath.Abs(destination); err != nil {
		return "", err
	}
	if err := skillpack.Export(sourceDir, destination, record.Slug, record.Name, record.Version, record.Description, record.ContentHash); err != nil {
		return "", err
	}
	if s.logger != nil {
		s.logger.Info("skill package exported", "slug", slug, "path", destination)
	}
	return destination, nil
}
