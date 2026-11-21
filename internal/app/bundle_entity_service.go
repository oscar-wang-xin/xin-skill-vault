package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"xinskillvault/internal/skillpack"
	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

type BundleItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SkillCount  int      `json:"skillCount"`
	Slugs       []string `json:"slugs"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
}

func (s *Service) bundleItem(record store.BundleRecord) (BundleItem, error) {
	skills, err := s.store.ListBundleSkills(record.ID)
	if err != nil {
		return BundleItem{}, err
	}
	slugs := make([]string, 0, len(skills))
	for _, item := range skills {
		slugs = append(slugs, item.Slug)
	}
	return BundleItem{
		ID: record.ID, Name: record.Name, Description: record.Description,
		SkillCount: len(skills), Slugs: slugs, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}, nil
}

func (s *Service) ListBundles() ([]BundleItem, error) {
	records, err := s.store.ListBundles()
	if err != nil {
		return nil, err
	}
	out := make([]BundleItem, 0, len(records))
	for _, record := range records {
		item, err := s.bundleItem(record)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) bundleSkillIDs(slugs []string) ([]string, error) {
	ids := make([]string, 0, len(slugs))
	seen := map[string]bool{}
	for _, slug := range slugs {
		if seen[slug] {
			continue
		}
		record, err := s.store.FindSkillBySlug(slug)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, fmt.Errorf("技能不存在：%s", slug)
		}
		seen[slug] = true
		ids = append(ids, record.ID)
	}
	return ids, nil
}

func (s *Service) CreateBundle(name, description string, slugs []string) (BundleItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return BundleItem{}, fmt.Errorf("Bundle 名称不能为空")
	}
	ids, err := s.bundleSkillIDs(slugs)
	if err != nil {
		return BundleItem{}, err
	}
	now := time.Now().Unix()
	record := store.BundleRecord{
		ID: newBundleID(), Name: name, Description: strings.TrimSpace(description),
		CreatedAt: now, UpdatedAt: now, SkillIDs: ids,
	}
	if err := s.store.CreateBundle(record); err != nil {
		return BundleItem{}, err
	}
	return s.bundleItem(record)
}

func (s *Service) UpdateBundle(id, name, description string, slugs []string) (BundleItem, error) {
	record, err := s.store.FindBundle(id)
	if err != nil {
		return BundleItem{}, err
	}
	if record == nil {
		return BundleItem{}, fmt.Errorf("Bundle 不存在")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return BundleItem{}, fmt.Errorf("Bundle 名称不能为空")
	}
	ids, err := s.bundleSkillIDs(slugs)
	if err != nil {
		return BundleItem{}, err
	}
	record.Name = name
	record.Description = strings.TrimSpace(description)
	record.SkillIDs = ids
	record.UpdatedAt = time.Now().Unix()
	if err := s.store.UpdateBundle(*record); err != nil {
		return BundleItem{}, err
	}
	return s.bundleItem(*record)
}

func (s *Service) DeleteBundle(id string) error {
	return s.store.DeleteBundle(id)
}

func (s *Service) ListBundleSkills(id string) ([]ComboSkillItem, error) {
	records, err := s.store.ListBundleSkills(id)
	if err != nil {
		return nil, err
	}
	out := make([]ComboSkillItem, 0, len(records))
	for _, item := range records {
		out = append(out, ComboSkillItem{
			Slug: item.Slug, Name: item.Name, Description: item.Description,
			Version: item.Version, Enabled: item.Enabled,
		})
	}
	return out, nil
}

func (s *Service) ExportBundle(id string) (string, error) {
	record, err := s.store.FindBundle(id)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("Bundle 不存在")
	}
	records, err := s.store.ListBundleSkills(id)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", fmt.Errorf("Bundle 里没有技能")
	}
	members := make([]skillpack.BundleSkill, 0, len(records))
	for _, skillRecord := range records {
		dir, err := vault.New(s.config.Current().VaultPath, s.store).SkillDir(skillRecord.Slug)
		if err != nil {
			return "", err
		}
		members = append(members, skillpack.BundleSkill{
			Slug: skillRecord.Slug, Name: skillRecord.Name, Version: skillRecord.Version,
			Description: skillRecord.Description, ContentHash: skillRecord.ContentHash, Dir: dir,
		})
	}
	if hostApp == nil {
		return "", fmt.Errorf("应用尚未初始化")
	}
	destination, err := hostApp.Dialog.SaveFile().
		SetFilename(skillpack.SafeFilename(record.Name)+".bundle.zip").
		AddFilter("Skill Bundle (*.zip)", "*.zip").
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
	if err := skillpack.ExportBundle(destination, record.Name, members); err != nil {
		return "", err
	}
	return destination, nil
}

func newBundleID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("bd_%d", time.Now().UnixNano())
	}
	return "bd_" + hex.EncodeToString(buf)
}
