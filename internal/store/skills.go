package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type SkillRecord struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Version     string
	Author      string
	License     string
	Icon        string
	Path        string
	ContentHash string
	SizeBytes   int64
	FileCount   int
	Enabled     bool
	SourceType  string
	SourceRef   string
	SourceRev   string
	SourceURL   string
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64
}

const skillColumns = `id, slug, name, description, version, author, license, icon, path,
	content_hash, size_bytes, file_count, enabled, source_type, source_ref, source_rev,
	source_url, created_at, updated_at, deleted_at`

func (s *Store) CreateSkill(record SkillRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO skills (
			id, slug, name, description, version, author, license, icon, path,
			content_hash, size_bytes, file_count, enabled, source_type, source_ref, source_rev,
			source_url, created_at, updated_at, deleted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.Slug, record.Name, record.Description, nullString(record.Version),
		nullString(record.Author), nullString(record.License), nullString(record.Icon),
		record.Path, record.ContentHash, record.SizeBytes, record.FileCount, boolInt(record.Enabled),
		record.SourceType, nullString(record.SourceRef), nullString(record.SourceRev),
		nullString(record.SourceURL), record.CreatedAt, record.UpdatedAt, record.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert skill: %w", err)
	}
	return nil
}

func (s *Store) FindSkillBySlug(slug string) (*SkillRecord, error) {
	row := s.db.QueryRow(`SELECT `+skillColumns+` FROM skills WHERE slug = ? AND deleted_at IS NULL`, slug)
	record, err := scanSkill(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find skill %q: %w", slug, err)
	}
	return &record, nil
}

func (s *Store) ListSkills() ([]SkillRecord, error) {
	rows, err := s.db.Query(`SELECT ` + skillColumns + ` FROM skills WHERE deleted_at IS NULL ORDER BY updated_at DESC, name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()
	var out []SkillRecord
	for rows.Next() {
		record, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate skills: %w", err)
	}
	return out, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSkill(row scanner) (SkillRecord, error) {
	var (
		record                          SkillRecord
		version, author, license, icon  sql.NullString
		sourceRef, sourceRev, sourceURL sql.NullString
		deletedAt                       sql.NullInt64
		enabled                         int
	)
	err := row.Scan(
		&record.ID, &record.Slug, &record.Name, &record.Description, &version, &author,
		&license, &icon, &record.Path, &record.ContentHash, &record.SizeBytes, &record.FileCount,
		&enabled, &record.SourceType, &sourceRef, &sourceRev, &sourceURL,
		&record.CreatedAt, &record.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return SkillRecord{}, err
	}
	record.Version = version.String
	record.Author = author.String
	record.License = license.String
	record.Icon = icon.String
	record.SourceRef = sourceRef.String
	record.SourceRev = sourceRev.String
	record.SourceURL = sourceURL.String
	record.Enabled = enabled != 0
	if deletedAt.Valid {
		value := deletedAt.Int64
		record.DeletedAt = &value
	}
	return record, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
