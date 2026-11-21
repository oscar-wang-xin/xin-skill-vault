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

func (s *Service) SearchMarket(source, query, category string, page int) (market.Result, error) {
	if page < 1 {
		page = 1
	}
	source = strings.TrimSpace(source)
	cache := s.marketCache()
	cacheKey := fmt.Sprintf("search:%s:%s:%s:%d", source, query, category, page)
	var cached market.Result
	if cache.Get(cacheKey, 5*time.Minute, &cached) {
		s.markInstalled(cached.Items)
		return cached, nil
	}
	providers := market.DefaultProviders()
	result := market.Result{Items: []market.Skill{}, Page: page}
	if source == "" || source == "all" {
		for _, provider := range providers {
			if category != "" && provider.ID() == "skills-sh" {
				continue
			}
			items, total, err := provider.Search(context.Background(), query, category, page)
			if err != nil {
				continue
			}
			result.Items = append(result.Items, items...)
			result.Total += total
		}
		if len(result.Items) == 0 {
			return result, fmt.Errorf("市场源暂时不可用")
		}
	} else {
		provider, ok := providers[source]
		if !ok {
			return result, fmt.Errorf("该市场源暂未接入")
		}
		items, total, err := provider.Search(context.Background(), query, category, page)
		if err != nil {
			return result, err
		}
		result.Items = items
		result.Total = total
	}
	s.markInstalled(result.Items)
	_ = cache.Put(cacheKey, result)
	return result, nil
}

func (s *Service) ListMarketCategories() ([]market.Category, error) {
	cache := s.marketCache()
	var categories []market.Category
	if cache.Get("categories", 24*time.Hour, &categories) {
		return categories, nil
	}
	provider := market.NewSkillHub()
	categories, err := provider.Categories(context.Background())
	if err != nil {
		return nil, err
	}
	_ = cache.Put("categories", categories)
	return categories, nil
}

func (s *Service) marketCache() market.Cache {
	return market.Cache{Dir: filepath.Join(s.config.Current().VaultPath, ".skillrepo", "market", "cache")}
}

func (s *Service) markInstalled(items []market.Skill) {
	for index := range items {
		if record, err := s.store.FindSkillBySlug(items[index].Slug); err == nil && record != nil {
			items[index].Installed = true
		}
	}
}

func (s *Service) AuditMarketSkill(source, slug, namespace string) (market.Audit, error) {
	provider, ok := market.DefaultProviders()[source]
	if !ok {
		return market.Audit{}, fmt.Errorf("该市场源暂未接入")
	}
	tempDir, err := os.MkdirTemp("", "xsv-audit-*")
	if err != nil {
		return market.Audit{}, err
	}
	defer os.RemoveAll(tempDir)
	item := market.Skill{Source: source, Slug: slug, Namespace: namespace}
	tempPath := filepath.Join(tempDir, "skill.zip")
	if err := provider.Download(context.Background(), item, tempPath); err != nil {
		return market.Audit{}, err
	}
	return market.AuditArchive(tempPath)
}

func (s *Service) InstallMarketSkill(source, slug, namespace string, force bool) (SkillItem, error) {
	providers := market.DefaultProviders()
	provider, ok := providers[source]
	if !ok {
		return SkillItem{}, fmt.Errorf("该市场源暂未接入")
	}
	itemURL := ""
	switch source {
	case "skillhub":
		itemURL = "https://www.skillhub.cn/" + namespace + "/" + slug
	case "skills-sh":
		itemURL = "https://www.skills.sh/" + namespace + "/" + slug
	}
	skill := market.Skill{
		Source: source, SourceName: source, Slug: slug, Namespace: namespace, URL: itemURL,
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
	audit, err := market.AuditArchive(tempPath)
	if err != nil {
		return SkillItem{}, err
	}
	if audit.Level == "high" && !force {
		return SkillItem{}, fmt.Errorf("安装包触发高风险审计：%s", strings.Join(audit.Findings, "；"))
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
