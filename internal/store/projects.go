package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type ProjectRecord struct {
	ID          string
	Name        string
	Path        string
	Description string
	CreatedAt   int64
	UpdatedAt   int64
}

func (s *Store) CreateProject(record ProjectRecord) error {
	_, err := s.db.Exec(`INSERT INTO projects (id, name, path, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		record.ID, record.Name, record.Path, nullString(record.Description), record.CreatedAt, record.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

func (s *Store) ListProjects() ([]ProjectRecord, error) {
	rows, err := s.db.Query(`SELECT id, name, path, description, created_at, updated_at
		FROM projects ORDER BY updated_at DESC, name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	var out []ProjectRecord
	for rows.Next() {
		var record ProjectRecord
		var description sql.NullString
		if err := rows.Scan(&record.ID, &record.Name, &record.Path, &description, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		record.Description = description.String
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) FindProject(id string) (*ProjectRecord, error) {
	row := s.db.QueryRow(`SELECT id, name, path, description, created_at, updated_at FROM projects WHERE id = ?`, id)
	var record ProjectRecord
	var description sql.NullString
	if err := row.Scan(&record.ID, &record.Name, &record.Path, &description, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find project: %w", err)
	}
	record.Description = description.String
	return &record, nil
}

func (s *Store) UpdateProject(record ProjectRecord) error {
	_, err := s.db.Exec(`UPDATE projects SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		record.Name, nullString(record.Description), record.UpdatedAt, record.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	return nil
}

func (s *Store) DeleteProject(id string) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

func (s *Store) SetProjectAgents(projectID string, agentIDs []string, now int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project_agents WHERE project_id = ?`, projectID); err != nil {
		tx.Rollback()
		return fmt.Errorf("clear project agents: %w", err)
	}
	for _, agentID := range agentIDs {
		if _, err := tx.Exec(`INSERT INTO project_agents (project_id, agent_id, created_at) VALUES (?, ?, ?)`,
			projectID, agentID, now); err != nil {
			tx.Rollback()
			return fmt.Errorf("add project agent: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) ListProjectAgents(projectID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT agent_id FROM project_agents WHERE project_id = ? ORDER BY created_at`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project agents: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) ListAssignmentsByTarget(targetID string) ([]AssignmentRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, skill_id, target_id, mode, source_path, source_hash, installed_path,
			installed_hash, status, assigned_at, updated_at
		FROM assignments WHERE target_id = ?`, targetID)
	if err != nil {
		return nil, fmt.Errorf("list target assignments: %w", err)
	}
	defer rows.Close()
	var out []AssignmentRecord
	for rows.Next() {
		var record AssignmentRecord
		var installedHash sql.NullString
		if err := rows.Scan(&record.ID, &record.SkillID, &record.TargetID, &record.Mode, &record.SourcePath,
			&record.SourceHash, &record.InstalledPath, &installedHash, &record.Status, &record.AssignedAt, &record.UpdatedAt); err != nil {
			return nil, err
		}
		record.InstalledHash = installedHash.String
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) DeleteTarget(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM assignments WHERE target_id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("delete target assignments: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM targets WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("delete target: %w", err)
	}
	return tx.Commit()
}
