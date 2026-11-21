package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type ComboRecord struct {
	ID          string
	Name        string
	Description string
	Color       string
	Icon        string
	CreatedAt   int64
	SkillIDs    []string
}

func (s *Store) CreateCombo(record ComboRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO groups (id, name, description, color, icon, kind, created_at)
		VALUES (?, ?, ?, ?, ?, 'manual', ?)`,
		record.ID, record.Name, nullString(record.Description), nullString(record.Color), nullString(record.Icon), record.CreatedAt); err != nil {
		tx.Rollback()
		return fmt.Errorf("create combo: %w", err)
	}
	for _, skillID := range record.SkillIDs {
		if _, err := tx.Exec(`INSERT INTO skill_groups (skill_id, group_id) VALUES (?, ?)`, skillID, record.ID); err != nil {
			tx.Rollback()
			return fmt.Errorf("add combo skill: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) UpdateCombo(record ComboRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE groups SET name = ?, description = ?, color = ?, icon = ? WHERE id = ?`,
		record.Name, nullString(record.Description), nullString(record.Color), nullString(record.Icon), record.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("update combo: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM skill_groups WHERE group_id = ?`, record.ID); err != nil {
		tx.Rollback()
		return err
	}
	for _, skillID := range record.SkillIDs {
		if _, err := tx.Exec(`INSERT INTO skill_groups (skill_id, group_id) VALUES (?, ?)`, skillID, record.ID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) DeleteCombo(id string) error {
	_, err := s.db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete combo: %w", err)
	}
	return nil
}

func (s *Store) FindCombo(id string) (*ComboRecord, error) {
	row := s.db.QueryRow(`SELECT id, name, description, color, icon, created_at FROM groups WHERE id = ? AND kind = 'manual'`, id)
	var record ComboRecord
	var description, color, icon sql.NullString
	if err := row.Scan(&record.ID, &record.Name, &description, &color, &icon, &record.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	record.Description = description.String
	record.Color = color.String
	record.Icon = icon.String
	ids, err := s.comboSkillIDs(id)
	if err != nil {
		return nil, err
	}
	record.SkillIDs = ids
	return &record, nil
}

func (s *Store) ListCombos() ([]ComboRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, description, color, icon, created_at
		FROM groups WHERE kind = 'manual' ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list combos: %w", err)
	}
	defer rows.Close()
	var out []ComboRecord
	for rows.Next() {
		var record ComboRecord
		var description, color, icon sql.NullString
		if err := rows.Scan(&record.ID, &record.Name, &description, &color, &icon, &record.CreatedAt); err != nil {
			return nil, err
		}
		record.Description = description.String
		record.Color = color.String
		record.Icon = icon.String
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		ids, err := s.comboSkillIDs(out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].SkillIDs = ids
	}
	return out, nil
}

func (s *Store) ListComboSkills(id string) ([]SkillRecord, error) {
	rows, err := s.db.Query(`SELECT `+skillColumns+` FROM skills
		JOIN skill_groups sg ON sg.skill_id = skills.id
		WHERE sg.group_id = ? AND skills.deleted_at IS NULL
		ORDER BY skills.name COLLATE NOCASE`, id)
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

func (s *Store) comboSkillIDs(groupID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT skill_id FROM skill_groups WHERE group_id = ?`, groupID)
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
