package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xinskillvault/internal/market"
	"xinskillvault/internal/vault"
)

func (s *Service) ListMarketSources() []market.Source {
	return market.Sources()
}

func (s *Service) SearchMarket(source, query string, page int) (market.Result, error) {
	if page < 1 {
		page = 1
	}
	providers := market.DefaultProviders()
	source = strings.TrimSpace(source)
	if source == "" || source == "all" {
		result := market.Result{Items: []market.Skill{}, Page: page}
		for _, provider := range providers {
			items, total, err := provider.Search(context.Background(), query, page)
			if err != nil {
				continue
			}
			result.Items = append(result.Items, items...)
			result.Total += total
		}
		s.markInstalled(result.Items)
		if len(result.Items) == 0 {
			return result, fmt.Errorf("市场源暂时不可用")
		}
		return result, nil
	}
	provider, ok := providers[source]
	if !ok {
		return market.Result{Items: []market.Skill{}, Page: page}, fmt.Errorf("该市场源暂未接入")
	}
	items, total, err := provider.Search(context.Background(), query, page)
	if err != nil {
		return market.Result{Items: []market.Skill{}, Page: page}, err
	}
	s.markInstalled(items)
	return market.Result{Items: items, Total: total, Page: page}, nil
}

func (s *Service) markInstalled(items []market.Skill) {
	for index := range items {
		if record, err := s.store.FindSkillBySlug(items[index].Slug); err == nil && record != nil {
			items[index].Installed = true
		}
	}
}

func (s *Service) InstallMarketSkill(source, slug, namespace string) (SkillItem, error) {
	providers := market.DefaultProviders()
	provider, ok := providers[source]
	if !ok {
		return SkillItem{}, fmt.Errorf("该市场源暂未接入")
	}
	skill := market.Skill{
		Source: source, SourceName: source, Slug: slug, Namespace: namespace,
		URL: "https://www.skillhub.cn/" + namespace + "/" + slug,
	}
	if existing, err := s.store.FindSkillBySlug(slug); err != nil {
		return SkillItem{}, err
	} else if existing != nil {
		return SkillItem{}, fmt.Errorf("技能 %q 已在仓库中", slug)
	}
	tempDir, err := os.MkdirTemp("", "xsv-market-*")
	if err != nil {
		return SkillItem{}, err
	}
	defer os.RemoveAll(tempDir)
	tempPath := filepath.Join(tempDir, "skill.zip")
	if err := provider.Download(context.Background(), skill, tempPath); err != nil {
		return SkillItem{}, err
	}
	cfg := s.config.Current()
	record, err := vault.New(cfg.VaultPath, s.store).ImportArchive(tempPath)
	if err != nil {
		return SkillItem{}, err
	}
	record.SourceType = "market"
	record.SourceRef = skill.URL
	record.SourceURL = skill.URL
	record.SourceRev = time.Now().UTC().Format("2006-01-02")
	if err := s.store.UpdateSkill(record); err != nil {
		return SkillItem{}, err
	}
	if s.logger != nil {
		s.logger.Info("market skill imported", "source", source, "slug", slug)
	}
	return skillItem(record), nil
}
