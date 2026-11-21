package app

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"xinskillvault/internal/store"
)

type ComboItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Color       string   `json:"color"`
	SkillCount  int      `json:"skillCount"`
	Slugs       []string `json:"slugs"`
	CreatedAt   int64    `json:"createdAt"`
}

type ComboSkillItem struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
}

func (s *Service) comboItem(record store.ComboRecord) (ComboItem, error) {
	skills, err := s.store.ListComboSkills(record.ID)
	if err != nil {
		return ComboItem{}, err
	}
	slugs := make([]string, 0, len(skills))
	for _, item := range skills {
		slugs = append(slugs, item.Slug)
	}
	return ComboItem{
		ID: record.ID, Name: record.Name, Description: record.Description,
		Icon: record.Icon, Color: record.Color, SkillCount: len(skills),
		Slugs: slugs, CreatedAt: record.CreatedAt,
	}, nil
}

func (s *Service) ListCombos() ([]ComboItem, error) {
	records, err := s.store.ListCombos()
	if err != nil {
		return nil, err
	}
	out := make([]ComboItem, 0, len(records))
	for _, record := range records {
		item, err := s.comboItem(record)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) comboSkillIDs(slugs []string) ([]string, error) {
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

func (s *Service) CreateCombo(name, description, icon string, slugs []string) (ComboItem, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ComboItem{}, fmt.Errorf("组合名称不能为空")
	}
	ids, err := s.comboSkillIDs(slugs)
	if err != nil {
		return ComboItem{}, err
	}
	record := store.ComboRecord{
		ID: newComboID(), Name: name, Description: strings.TrimSpace(description),
		Icon: strings.TrimSpace(icon), CreatedAt: time.Now().Unix(), SkillIDs: ids,
	}
	if err := s.store.CreateCombo(record); err != nil {
		return ComboItem{}, err
	}
	return s.comboItem(record)
}

func (s *Service) UpdateCombo(id, name, description, icon string, slugs []string) (ComboItem, error) {
	record, err := s.store.FindCombo(id)
	if err != nil {
		return ComboItem{}, err
	}
	if record == nil {
		return ComboItem{}, fmt.Errorf("组合不存在")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return ComboItem{}, fmt.Errorf("组合名称不能为空")
	}
	ids, err := s.comboSkillIDs(slugs)
	if err != nil {
		return ComboItem{}, err
	}
	record.Name = name
	record.Description = strings.TrimSpace(description)
	record.Icon = strings.TrimSpace(icon)
	record.SkillIDs = ids
	if err := s.store.UpdateCombo(*record); err != nil {
		return ComboItem{}, err
	}
	return s.comboItem(*record)
}

func (s *Service) DeleteCombo(id string) error {
	return s.store.DeleteCombo(id)
}

func (s *Service) ListComboSkills(id string) ([]ComboSkillItem, error) {
	record, err := s.store.FindCombo(id)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("组合不存在")
	}
	records, err := s.store.ListComboSkills(id)
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

func newComboID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("cb_%d", time.Now().UnixNano())
	}
	return "cb_" + hex.EncodeToString(buf)
}
