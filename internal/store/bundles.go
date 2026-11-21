package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type BundleRecord struct {
	ID          string
	Name        string
	Description string
	CreatedAt   int64
	UpdatedAt   int64
	SkillIDs    []string
}

func (s *Store) CreateBundle(record BundleRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO bundles (id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		record.ID, record.Name, nullString(record.Description), record.CreatedAt, record.UpdatedAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("create bundle: %w", err)
	}
	for _, skillID := range record.SkillIDs {
		if _, err := tx.Exec(`INSERT INTO bundle_members (bundle_id, skill_id, created_at) VALUES (?, ?, ?)`,
			record.ID, skillID, record.CreatedAt); err != nil {
			tx.Rollback()
			return fmt.Errorf("add bundle member: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) UpdateBundle(record BundleRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE bundles SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		record.Name, nullString(record.Description), record.UpdatedAt, record.ID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM bundle_members WHERE bundle_id = ?`, record.ID); err != nil {
		tx.Rollback()
		return err
	}
	for _, skillID := range record.SkillIDs {
		if _, err := tx.Exec(`INSERT INTO bundle_members (bundle_id, skill_id, created_at) VALUES (?, ?, ?)`,
			record.ID, skillID, record.CreatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteBundle(id string) error {
	_, err := s.db.Exec(`DELETE FROM bundles WHERE id = ?`, id)
	return err
}

func (s *Store) FindBundle(id string) (*BundleRecord, error) {
	row := s.db.QueryRow(`SELECT id, name, description, created_at, updated_at FROM bundles WHERE id = ?`, id)
	var record BundleRecord
	var description sql.NullString
	if err := row.Scan(&record.ID, &record.Name, &description, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	record.Description = description.String
	ids, err := s.bundleSkillIDs(id)
	if err != nil {
		return nil, err
	}
	record.SkillIDs = ids
	return &record, nil
}

func (s *Store) ListBundles() ([]BundleRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, description, created_at, updated_at FROM bundles ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BundleRecord
	for rows.Next() {
		var record BundleRecord
		var description sql.NullString
		if err := rows.Scan(&record.ID, &record.Name, &description, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		record.Description = description.String
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ids, err := s.bundleSkillIDs(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].SkillIDs = ids
	}
	return out, nil
}

func (s *Store) ListBundleSkills(id string) ([]SkillRecord, error) {
	rows, err := s.db.Query(`SELECT `+skillColumns+` FROM skills
		WHERE id IN (SELECT skill_id FROM bundle_members WHERE bundle_id = ?) AND deleted_at IS NULL
		ORDER BY name COLLATE NOCASE`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillRecord
	for rows.Next() {
		record, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) bundleSkillIDs(bundleID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT skill_id FROM bundle_members WHERE bundle_id = ?`, bundleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
