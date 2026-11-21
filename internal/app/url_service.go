package app

import (
	"context"
	"fmt"
	"os"
	"strings"

	"xinskillvault/internal/netimport"
	"xinskillvault/internal/skill"
	"xinskillvault/internal/vault"
)

type URLImportResult struct {
	Status string    `json:"status"`
	Skill  SkillItem `json:"skill"`
}

func (s *Service) ImportSkillURL(rawURL, conflict string) (URLImportResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return URLImportResult{}, fmt.Errorf("URL 不能为空")
	}
	switch conflict {
	case "", "skip":
		conflict = "skip"
	case "overwrite":
	default:
		return URLImportResult{}, fmt.Errorf("不支持的冲突策略")
	}
	tempDir, err := os.MkdirTemp("", "xsv-url-*")
	if err != nil {
		return URLImportResult{}, err
	}
	defer os.RemoveAll(tempDir)
	root, err := netimport.NewFetcher().Fetch(context.Background(), rawURL, tempDir)
	if err != nil {
		return URLImportResult{}, err
	}
	parsed, err := skill.ParseDir(root)
	if err != nil {
		return URLImportResult{}, err
	}
	existing, err := s.store.FindSkillBySlug(parsed.Slug)
	if err != nil {
		return URLImportResult{}, err
	}
	if existing != nil {
		if conflict == "skip" {
			item := skillItem(*existing)
			if tags, err := s.store.SkillTags(existing.ID); err == nil {
				item.Tags = tags
			}
			return URLImportResult{Status: "skipped", Skill: item}, nil
		}
		record, err := vault.New(s.config.Current().VaultPath, s.store).ReplaceFromDirectory(
			parsed.Slug, root, "url", rawURL, "URL 覆盖前",
		)
		if err != nil {
			return URLImportResult{}, err
		}
		return URLImportResult{Status: "overwritten", Skill: skillItem(record)}, nil
	}
	record, err := vault.New(s.config.Current().VaultPath, s.store).ImportDirectory(root)
	if err != nil {
		return URLImportResult{}, err
	}
	record.SourceType = "url"
	record.SourceRef = rawURL
	record.SourceURL = rawURL
	if err := s.store.UpdateSkill(record); err != nil {
		return URLImportResult{}, err
	}
	return URLImportResult{Status: "imported", Skill: skillItem(record)}, nil
}
