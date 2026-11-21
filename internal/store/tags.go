package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type TagRecord struct {
	ID    string
	Name  string
	Color string
	Count int
}

func (s *Store) ListTags() ([]TagRecord, error) {
	rows, err := s.db.Query(`SELECT t.id, t.name, COALESCE(t.color, ''), COUNT(st.skill_id)
		FROM tags t LEFT JOIN skill_tags st ON st.tag_id = t.id
		GROUP BY t.id, t.name, t.color ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	var out []TagRecord
	for rows.Next() {
		var record TagRecord
		if err := rows.Scan(&record.ID, &record.Name, &record.Color, &record.Count); err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) SkillTags(skillID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT t.name FROM tags t JOIN skill_tags st ON st.tag_id = t.id
		WHERE st.skill_id = ? ORDER BY t.name COLLATE NOCASE`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func (s *Store) SetSkillTags(skillID string, names []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM skill_tags WHERE skill_id = ?`, skillID); err != nil {
		tx.Rollback()
		return err
	}
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		if _, err := tx.Exec(`INSERT INTO tags (id, name, created_at) VALUES (?, ?, 0)
			ON CONFLICT(name) DO NOTHING`, "tag_"+name, name); err != nil {
			tx.Rollback()
			return fmt.Errorf("create tag: %w", err)
		}
		var tagID string
		if err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO skill_tags (skill_id, tag_id) VALUES (?, ?)`, skillID, tagID); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetSkillNote(skillID string) (string, error) {
	row := s.db.QueryRow(`SELECT body FROM notes WHERE skill_id = ? ORDER BY updated_at DESC LIMIT 1`, skillID)
	var body string
	if err := row.Scan(&body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return body, nil
}

func (s *Store) SaveSkillNote(skillID, body string) error {
	var id string
	err := s.db.QueryRow(`SELECT id FROM notes WHERE skill_id = ? ORDER BY updated_at DESC LIMIT 1`, skillID).Scan(&id)
	now := timeNow()
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.Exec(`INSERT INTO notes (id, skill_id, body, updated_at) VALUES (?, ?, ?, ?)`, "note_"+skillID, skillID, body, now)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE notes SET body = ?, updated_at = ? WHERE id = ?`, body, now, id)
	return err
}

func timeNow() int64 {
	return time.Now().Unix()
}

func (s *Store) FindTagByName(name string) (*TagRecord, error) {
	row := s.db.QueryRow(`SELECT id, name, COALESCE(color, ''), 0 FROM tags WHERE name = ?`, name)
	var record TagRecord
	if err := row.Scan(&record.ID, &record.Name, &record.Color, &record.Count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (s *Store) RenameTag(id, name string) error {
	_, err := s.db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("rename tag: %w", err)
	}
	return nil
}

func (s *Store) DeleteTag(id string) error {
	_, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

func (s *Store) MergeTag(sourceID, targetID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO skill_tags (skill_id, tag_id)
		SELECT skill_id, ? FROM skill_tags WHERE tag_id = ?`, targetID, sourceID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, sourceID); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
