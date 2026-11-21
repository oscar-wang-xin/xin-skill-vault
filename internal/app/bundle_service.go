package app

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"xinskillvault/internal/skill"
	"xinskillvault/internal/skillpack"
	"xinskillvault/internal/vault"
)

type BundleImportResult struct {
	Imported []string `json:"imported"`
	Skipped  []string `json:"skipped"`
	Errors   []string `json:"errors"`
	Bundle   string   `json:"bundle"`
}

func (s *Service) PickBundleArchive() (string, error) {
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	value, err := hostApp.Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		AddFilter("Skill Bundle (*.zip)", "*.zip").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("选择 Bundle 失败：%w", err)
	}
	return value, nil
}

func (s *Service) ImportBundleArchive(archivePath string) (BundleImportResult, error) {
	result := BundleImportResult{Imported: []string{}, Skipped: []string{}, Errors: []string{}}
	abs, err := filepath.Abs(archivePath)
	if err != nil {
		return result, err
	}
	reader, err := zip.OpenReader(abs)
	if err != nil {
		return result, fmt.Errorf("打开 Bundle: %w", err)
	}
	defer reader.Close()
	tempDir, err := os.MkdirTemp("", "xsv-bundle-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(tempDir)
	for _, entry := range reader.File {
		name := strings.ReplaceAll(entry.Name, "\\", "/")
		clean := path.Clean(strings.TrimPrefix(name, "/"))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
			return result, fmt.Errorf("Bundle 包含不安全路径：%s", entry.Name)
		}
		target := filepath.Join(tempDir, filepath.FromSlash(clean))
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return result, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return result, err
		}
		input, err := entry.Open()
		if err != nil {
			return result, err
		}
		output, err := os.Create(target)
		if err != nil {
			input.Close()
			return result, err
		}
		_, copyErr := io.Copy(output, input)
		input.Close()
		output.Close()
		if copyErr != nil {
			return result, copyErr
		}
	}
	manifestPath := filepath.Join(tempDir, "bundle.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return result, fmt.Errorf("Bundle 缺少 bundle.json")
	}
	var manifest skillpack.BundleManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return result, fmt.Errorf("解析 bundle.json: %w", err)
	}
	result.Bundle = manifest.Name
	skillsRoot := filepath.Join(tempDir, "skills")
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return result, fmt.Errorf("Bundle 没有 skills 目录")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(skillsRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, skill.DocName)); err != nil {
			result.Skipped = append(result.Skipped, entry.Name())
			continue
		}
		record, err := vault.New(s.config.Current().VaultPath, s.store).ImportDirectory(dir)
		if err != nil {
			result.Skipped = append(result.Skipped, entry.Name())
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		record.SourceType = "bundle"
		record.SourceRef = abs
		record.SourceRev = manifest.Name
		if err := s.store.UpdateSkill(record); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		result.Imported = append(result.Imported, record.Slug)
	}
	if len(result.Imported) == 0 && len(result.Skipped) == 0 {
		return result, fmt.Errorf("Bundle 里没有可导入的技能")
	}
	return result, nil
}
