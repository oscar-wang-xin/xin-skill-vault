package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SkillRecord struct {
	ID            string
	Slug          string
	Name          string
	Description   string
	Version       string
	Author        string
	License       string
	Icon          string
	Path          string
	ContentHash   string
	SizeBytes     int64
	FileCount     int
	Enabled       bool
	AssignedCount int
	SourceType    string
	SourceRef     string
	SourceRev     string
	SourceURL     string
	CreatedAt     int64
	UpdatedAt     int64
	DeletedAt     *int64
}

const skillColumns = `id, slug, name, description, version, author, license, icon, path,
	content_hash, size_bytes, file_count, enabled, source_type, source_ref, source_rev,
	source_url, created_at, updated_at, deleted_at,
	(SELECT COUNT(*) FROM assignments a WHERE a.skill_id = skills.id) AS assigned_count`

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
		assignedCount                   int
	)
	err := row.Scan(
		&record.ID, &record.Slug, &record.Name, &record.Description, &version, &author,
		&license, &icon, &record.Path, &record.ContentHash, &record.SizeBytes, &record.FileCount,
		&enabled, &record.SourceType, &sourceRef, &sourceRev, &sourceURL,
		&record.CreatedAt, &record.UpdatedAt, &deletedAt, &assignedCount,
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
	record.AssignedCount = assignedCount
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

type SnapshotRecord struct {
	ID          string
	SkillID     string
	ContentHash string
	FilePath    string
	Note        string
	CreatedAt   int64
}

func (s *Store) UpdateSkill(record SkillRecord) error {
	_, err := s.db.Exec(`
		UPDATE skills SET
			name = ?, description = ?, version = ?, author = ?, license = ?, icon = ?,
			content_hash = ?, size_bytes = ?, file_count = ?, source_type = ?, source_ref = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		record.Name, record.Description, nullString(record.Version), nullString(record.Author),
		nullString(record.License), nullString(record.Icon), record.ContentHash,
		record.SizeBytes, record.FileCount, record.SourceType, nullString(record.SourceRef),
		record.UpdatedAt, record.ID,
	)
	if err != nil {
		return fmt.Errorf("update skill: %w", err)
	}
	return nil
}

func (s *Store) CreateSnapshot(record SnapshotRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO snapshots (id, skill_id, content_hash, file_path, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		record.ID, record.SkillID, record.ContentHash, record.FilePath, nullString(record.Note), record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

func (s *Store) ListSnapshots(skillID string) ([]SnapshotRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, skill_id, content_hash, file_path, note, created_at
		FROM snapshots WHERE skill_id = ? ORDER BY created_at DESC`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()
	var out []SnapshotRecord
	for rows.Next() {
		var record SnapshotRecord
		var note sql.NullString
		if err := rows.Scan(&record.ID, &record.SkillID, &record.ContentHash, &record.FilePath, &note, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		record.Note = note.String
		out = append(out, record)
	}
	return out, rows.Err()
}

// UpdateSkillWithSnapshot updates a skill and records its previous snapshot in one transaction.
func (s *Store) UpdateSkillWithSnapshot(record SkillRecord, snapshot SnapshotRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin skill update: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE skills SET
			name = ?, description = ?, version = ?, author = ?, license = ?, icon = ?,
			content_hash = ?, size_bytes = ?, file_count = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		record.Name, record.Description, nullString(record.Version), nullString(record.Author),
		nullString(record.License), nullString(record.Icon), record.ContentHash,
		record.SizeBytes, record.FileCount, record.UpdatedAt, record.ID,
	); err != nil {
		tx.Rollback()
		return fmt.Errorf("update skill: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO snapshots (id, skill_id, content_hash, file_path, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		snapshot.ID, snapshot.SkillID, snapshot.ContentHash, snapshot.FilePath, nullString(snapshot.Note), snapshot.CreatedAt,
	); err != nil {
		tx.Rollback()
		return fmt.Errorf("insert snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit skill update: %w", err)
	}
	return nil
}

func (s *Store) SetSkillEnabled(id string, enabled bool) error {
	_, err := s.db.Exec(`UPDATE skills SET enabled = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		boolInt(enabled), time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("set skill enabled: %w", err)
	}
	return nil
}

func (s *Store) SoftDeleteSkill(id string, at int64) error {
	_, err := s.db.Exec(`UPDATE skills SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`, at, at, id)
	if err != nil {
		return fmt.Errorf("soft delete skill: %w", err)
	}
	return nil
}

func (s *Store) FindSnapshot(skillID, id string) (*SnapshotRecord, error) {
	row := s.db.QueryRow(`SELECT id, skill_id, content_hash, file_path, note, created_at
		FROM snapshots WHERE id = ? AND skill_id = ?`, id, skillID)
	var record SnapshotRecord
	var note sql.NullString
	if err := row.Scan(&record.ID, &record.SkillID, &record.ContentHash, &record.FilePath, &note, &record.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find snapshot: %w", err)
	}
	record.Note = note.String
	return &record, nil
}

func (s *Store) DeleteSnapshot(id string) error {
	_, err := s.db.Exec(`DELETE FROM snapshots WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}
	return nil
}
