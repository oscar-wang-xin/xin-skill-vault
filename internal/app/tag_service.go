package app

import (
	"fmt"
	"strings"
)

type TagItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Count int    `json:"count"`
}

func (s *Service) ListTags() ([]TagItem, error) {
	records, err := s.store.ListTags()
	if err != nil {
		return nil, err
	}
	out := make([]TagItem, 0, len(records))
	for _, record := range records {
		out = append(out, TagItem{ID: record.ID, Name: record.Name, Color: record.Color, Count: record.Count})
	}
	return out, nil
}

func (s *Service) SetSkillTags(slug string, tags []string) (SkillItem, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return SkillItem{}, err
	}
	if record == nil {
		return SkillItem{}, fmt.Errorf("技能不存在：%s", slug)
	}
	if err := s.store.SetSkillTags(record.ID, tags); err != nil {
		return SkillItem{}, err
	}
	item := skillItem(*record)
	if stored, err := s.store.SkillTags(record.ID); err == nil {
		item.Tags = stored
	} else {
		item.Tags = tags
	}
	return item, nil
}

func (s *Service) GetSkillNote(slug string) (string, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("技能不存在：%s", slug)
	}
	return s.store.GetSkillNote(record.ID)
}

func (s *Service) SaveSkillNote(slug, body string) (string, error) {
	record, err := s.store.FindSkillBySlug(slug)
	if err != nil {
		return "", err
	}
	if record == nil {
		return "", fmt.Errorf("技能不存在：%s", slug)
	}
	if err := s.store.SaveSkillNote(record.ID, body); err != nil {
		return "", err
	}
	return body, nil
}

func (s *Service) RenameTag(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	existing, err := s.store.FindTagByName(name)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != id {
		return fmt.Errorf("标签已存在")
	}
	return s.store.RenameTag(id, name)
}

func (s *Service) DeleteTag(id string) error {
	return s.store.DeleteTag(id)
}

func (s *Service) MergeTags(sourceID, targetID string) error {
	if sourceID == targetID {
		return fmt.Errorf("不能合并到自身")
	}
	return s.store.MergeTag(sourceID, targetID)
}
