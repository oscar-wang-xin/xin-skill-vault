package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type PromptCategoryRecord struct {
	ID        string
	Name      string
	ParentID  string
	DirPath   string
	SortOrder int
	CreatedAt int64
}

type PromptRecord struct {
	ID             string
	Title          string
	CategoryID     string
	Tags           []string
	Uses           int
	CurrentVersion int
	FilePath       string
	CurrentHash    string
	CreatedAt      int64
	UpdatedAt      int64
}

type PromptVersionRecord struct {
	ID          string
	PromptID    string
	Version     int
	Title       string
	CategoryID  string
	Tags        []string
	FilePath    string
	ContentHash string
	Note        string
	CreatedAt   int64
}

func (s *Store) ListPromptCategories() ([]PromptCategoryRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, COALESCE(parent_id, ''), dir_path, sort_order, created_at
		FROM prompt_categories ORDER BY sort_order, name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list prompt categories: %w", err)
	}
	defer rows.Close()
	var out []PromptCategoryRecord
	for rows.Next() {
		var record PromptCategoryRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.ParentID, &record.DirPath, &record.SortOrder, &record.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) FindPromptCategory(id string) (*PromptCategoryRecord, error) {
	row := s.db.QueryRow(`SELECT id, name, COALESCE(parent_id, ''), dir_path, sort_order, created_at
		FROM prompt_categories WHERE id = ?`, id)
	var record PromptCategoryRecord
	err := row.Scan(&record.ID, &record.Name, &record.ParentID, &record.DirPath, &record.SortOrder, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *Store) CreatePromptCategory(record PromptCategoryRecord) error {
	_, err := s.db.Exec(`INSERT INTO prompt_categories (id, name, parent_id, dir_path, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		record.ID, record.Name, nullString(record.ParentID), record.DirPath, record.SortOrder, record.CreatedAt)
	if err != nil {
		return fmt.Errorf("create prompt category: %w", err)
	}
	return nil
}

func (s *Store) UpdatePromptCategory(id, name string) error {
	_, err := s.db.Exec(`UPDATE prompt_categories SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("update prompt category: %w", err)
	}
	return nil
}

func (s *Store) DeletePromptCategory(id string) error {
	_, err := s.db.Exec(`DELETE FROM prompt_categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete prompt category: %w", err)
	}
	return nil
}

func (s *Store) CreatePromptWithVersion(record PromptRecord, version PromptVersionRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	tags, _ := json.Marshal(record.Tags)
	if _, err := tx.Exec(`INSERT INTO prompts (id, title, category_id, tags_json, uses, current_version, file_path, current_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?)`,
		record.ID, record.Title, nullString(record.CategoryID), string(tags), record.CurrentVersion,
		record.FilePath, record.CurrentHash, record.CreatedAt, record.UpdatedAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("create prompt: %w", err)
	}
	versionTags, _ := json.Marshal(version.Tags)
	if _, err := tx.Exec(`INSERT INTO prompt_versions (id, prompt_id, version, title, category_id, tags_json, file_path, content_hash, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		version.ID, version.PromptID, version.Version, version.Title, nullString(version.CategoryID),
		string(versionTags), version.FilePath, version.ContentHash, nullString(version.Note), version.CreatedAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("create prompt version: %w", err)
	}
	return tx.Commit()
}

func (s *Store) AddPromptVersion(record PromptRecord, version PromptVersionRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	versionTags, _ := json.Marshal(version.Tags)
	if _, err := tx.Exec(`INSERT INTO prompt_versions (id, prompt_id, version, title, category_id, tags_json, file_path, content_hash, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		version.ID, version.PromptID, version.Version, version.Title, nullString(version.CategoryID),
		string(versionTags), version.FilePath, version.ContentHash, nullString(version.Note), version.CreatedAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("add prompt version: %w", err)
	}
	tags, _ := json.Marshal(record.Tags)
	if _, err := tx.Exec(`UPDATE prompts SET title = ?, category_id = ?, tags_json = ?, current_version = ?, file_path = ?, current_hash = ?, updated_at = ? WHERE id = ?`,
		record.Title, nullString(record.CategoryID), string(tags), record.CurrentVersion,
		record.FilePath, record.CurrentHash, record.UpdatedAt, record.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("update prompt: %w", err)
	}
	return tx.Commit()
}

func (s *Store) FindPrompt(id string) (*PromptRecord, error) {
	row := s.db.QueryRow(`SELECT id, title, COALESCE(category_id, ''), tags_json, uses, current_version, file_path, current_hash, created_at, updated_at
		FROM prompts WHERE id = ?`, id)
	var record PromptRecord
	var tags string
	if err := row.Scan(&record.ID, &record.Title, &record.CategoryID, &tags, &record.Uses, &record.CurrentVersion,
		&record.FilePath, &record.CurrentHash, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	json.Unmarshal([]byte(tags), &record.Tags)
	if record.Tags == nil {
		record.Tags = []string{}
	}
	return &record, nil
}

func (s *Store) ListPrompts(category string) ([]PromptRecord, error) {
	query := `SELECT id, title, COALESCE(category_id, ''), tags_json, uses, current_version, file_path, current_hash, created_at, updated_at FROM prompts`
	args := []any{}
	switch category {
	case "", "all":
	case "none":
		query += ` WHERE category_id IS NULL`
	default:
		query += ` WHERE category_id = ?`
		args = append(args, category)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PromptRecord
	for rows.Next() {
		var record PromptRecord
		var tags string
		if err := rows.Scan(&record.ID, &record.Title, &record.CategoryID, &tags, &record.Uses, &record.CurrentVersion,
			&record.FilePath, &record.CurrentHash, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tags), &record.Tags)
		if record.Tags == nil {
			record.Tags = []string{}
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) ListPromptVersions(promptID string) ([]PromptVersionRecord, error) {
	rows, err := s.db.Query(`SELECT id, prompt_id, version, title, COALESCE(category_id, ''), tags_json, file_path, content_hash, COALESCE(note, ''), created_at
		FROM prompt_versions WHERE prompt_id = ? ORDER BY version DESC`, promptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PromptVersionRecord
	for rows.Next() {
		var record PromptVersionRecord
		var tags string
		if err := rows.Scan(&record.ID, &record.PromptID, &record.Version, &record.Title, &record.CategoryID,
			&tags, &record.FilePath, &record.ContentHash, &record.Note, &record.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tags), &record.Tags)
		if record.Tags == nil {
			record.Tags = []string{}
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) FindPromptVersion(promptID string, version int) (*PromptVersionRecord, error) {
	row := s.db.QueryRow(`SELECT id, prompt_id, version, title, COALESCE(category_id, ''), tags_json, file_path, content_hash, COALESCE(note, ''), created_at
		FROM prompt_versions WHERE prompt_id = ? AND version = ?`, promptID, version)
	var record PromptVersionRecord
	var tags string
	if err := row.Scan(&record.ID, &record.PromptID, &record.Version, &record.Title, &record.CategoryID,
		&tags, &record.FilePath, &record.ContentHash, &record.Note, &record.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	json.Unmarshal([]byte(tags), &record.Tags)
	return &record, nil
}

func (s *Store) SetPromptCurrent(promptID string, version PromptVersionRecord, updatedAt int64) error {
	_, err := s.db.Exec(`UPDATE prompts SET current_version = ?, file_path = ?, current_hash = ?, title = ?, category_id = ?, tags_json = ?, updated_at = ? WHERE id = ?`,
		version.Version, version.FilePath, version.ContentHash, version.Title, nullString(version.CategoryID),
		mustJSON(version.Tags), updatedAt, promptID)
	if err != nil {
		return fmt.Errorf("set prompt current: %w", err)
	}
	return nil
}

func (s *Store) DeletePromptVersion(promptID string, version int) error {
	_, err := s.db.Exec(`DELETE FROM prompt_versions WHERE prompt_id = ? AND version = ?`, promptID, version)
	if err != nil {
		return fmt.Errorf("delete prompt version: %w", err)
	}
	return nil
}

func (s *Store) DeletePrompt(id string) error {
	_, err := s.db.Exec(`DELETE FROM prompts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete prompt: %w", err)
	}
	return nil
}

func mustJSON(value []string) string {
	if value == nil {
		value = []string{}
	}
	data, _ := json.Marshal(value)
	return string(data)
}
