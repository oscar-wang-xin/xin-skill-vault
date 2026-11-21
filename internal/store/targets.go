package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type TargetRecord struct {
	ID          string
	Kind        string
	AgentID     string
	Name        string
	Path        string
	ConfigPath  string
	Platform    string
	Enabled     bool
	Detected    bool
	Writable    bool
	DefaultMode string
	LastScanAt  int64
	CreatedAt   int64
}

type AssignmentRecord struct {
	ID            string
	SkillID       string
	TargetID      string
	Mode          string
	SourcePath    string
	SourceHash    string
	InstalledPath string
	InstalledHash string
	Status        string
	AssignedAt    int64
	UpdatedAt     int64
}

func (s *Store) UpsertTarget(record TargetRecord) error {
	result, err := s.db.Exec(`
		UPDATE targets SET
			kind = ?, agent_id = ?, name = ?, path = ?, config_path = ?, platform = ?,
			enabled = ?, detected = ?, writable = ?, default_mode = ?, last_scan_at = ?
		WHERE id = ?`,
		record.Kind, nullString(record.AgentID), record.Name, record.Path, nullString(record.ConfigPath),
		nullString(record.Platform), boolInt(record.Enabled), boolInt(record.Detected),
		boolInt(record.Writable), record.DefaultMode, record.LastScanAt, record.ID,
	)
	if err != nil {
		return fmt.Errorf("update target: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		return nil
	}
	_, err = s.db.Exec(`
		INSERT INTO targets (id, kind, agent_id, name, path, config_path, platform, enabled, detected, writable, default_mode, last_scan_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(kind, path) DO UPDATE SET
			agent_id = excluded.agent_id,
			name = excluded.name,
			config_path = excluded.config_path,
			platform = excluded.platform,
			enabled = excluded.enabled,
			detected = excluded.detected,
			writable = excluded.writable,
			default_mode = excluded.default_mode`,
		record.ID, record.Kind, nullString(record.AgentID), record.Name, record.Path,
		nullString(record.ConfigPath), nullString(record.Platform), boolInt(record.Enabled),
		boolInt(record.Detected), boolInt(record.Writable), record.DefaultMode,
		record.LastScanAt, record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert target: %w", err)
	}
	return nil
}

func (s *Store) FindTargetByAgent(agentID string) (*TargetRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, kind, agent_id, name, path, config_path, platform, enabled, detected, writable,
			default_mode, last_scan_at, created_at
		FROM targets WHERE kind = 'agent' AND agent_id = ?`, agentID)
	var record TargetRecord
	var agent, config, platform sql.NullString
	var enabled, detected, writable int
	err := row.Scan(&record.ID, &record.Kind, &agent, &record.Name, &record.Path, &config,
		&platform, &enabled, &detected, &writable, &record.DefaultMode, &record.LastScanAt, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find target: %w", err)
	}
	record.AgentID = agent.String
	record.ConfigPath = config.String
	record.Platform = platform.String
	record.Enabled = enabled != 0
	record.Detected = detected != 0
	record.Writable = writable != 0
	return &record, nil
}

func (s *Store) FindTargetByID(id string) (*TargetRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, kind, agent_id, name, path, config_path, platform, enabled, detected, writable,
			default_mode, last_scan_at, created_at
		FROM targets WHERE id = ?`, id)
	var record TargetRecord
	var agent, config, platform sql.NullString
	var enabled, detected, writable int
	err := row.Scan(&record.ID, &record.Kind, &agent, &record.Name, &record.Path, &config,
		&platform, &enabled, &detected, &writable, &record.DefaultMode, &record.LastScanAt, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find target by id: %w", err)
	}
	record.AgentID = agent.String
	record.ConfigPath = config.String
	record.Platform = platform.String
	record.Enabled = enabled != 0
	record.Detected = detected != 0
	record.Writable = writable != 0
	return &record, nil
}

func (s *Store) FindAssignment(skillID, targetID string) (*AssignmentRecord, error) {
	row := s.db.QueryRow(`
		SELECT id, skill_id, target_id, mode, source_path, source_hash, installed_path,
			installed_hash, status, assigned_at, updated_at
		FROM assignments WHERE skill_id = ? AND target_id = ?`, skillID, targetID)
	var record AssignmentRecord
	var installedHash sql.NullString
	err := row.Scan(&record.ID, &record.SkillID, &record.TargetID, &record.Mode, &record.SourcePath,
		&record.SourceHash, &record.InstalledPath, &installedHash, &record.Status, &record.AssignedAt, &record.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find assignment: %w", err)
	}
	record.InstalledHash = installedHash.String
	return &record, nil
}

func (s *Store) ListAssignments(skillID string) ([]AssignmentRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, skill_id, target_id, mode, source_path, source_hash, installed_path,
			installed_hash, status, assigned_at, updated_at
		FROM assignments WHERE skill_id = ? ORDER BY assigned_at DESC`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()
	var out []AssignmentRecord
	for rows.Next() {
		var record AssignmentRecord
		var installedHash sql.NullString
		if err := rows.Scan(&record.ID, &record.SkillID, &record.TargetID, &record.Mode, &record.SourcePath,
			&record.SourceHash, &record.InstalledPath, &installedHash, &record.Status, &record.AssignedAt, &record.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		record.InstalledHash = installedHash.String
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) UpsertAssignment(record AssignmentRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO assignments (id, skill_id, target_id, mode, source_path, source_hash, installed_path, installed_hash, status, assigned_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(skill_id, target_id) DO UPDATE SET
			mode = excluded.mode,
			source_path = excluded.source_path,
			source_hash = excluded.source_hash,
			installed_path = excluded.installed_path,
			installed_hash = excluded.installed_hash,
			status = excluded.status,
			updated_at = excluded.updated_at`,
		record.ID, record.SkillID, record.TargetID, record.Mode, record.SourcePath, record.SourceHash,
		record.InstalledPath, nullString(record.InstalledHash), record.Status, record.AssignedAt, record.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert assignment: %w", err)
	}
	return nil
}

func (s *Store) DeleteAssignment(id string) error {
	_, err := s.db.Exec(`DELETE FROM assignments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}
	return nil
}

func (s *Store) ListTargets() ([]TargetRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, kind, agent_id, name, path, config_path, platform, enabled, detected, writable,
			default_mode, last_scan_at, created_at
		FROM targets ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list targets: %w", err)
	}
	defer rows.Close()
	var out []TargetRecord
	for rows.Next() {
		var record TargetRecord
		var agent, config, platform sql.NullString
		var enabled, detected, writable int
		if err := rows.Scan(&record.ID, &record.Kind, &agent, &record.Name, &record.Path, &config,
			&platform, &enabled, &detected, &writable, &record.DefaultMode, &record.LastScanAt, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan target: %w", err)
		}
		record.AgentID = agent.String
		record.ConfigPath = config.String
		record.Platform = platform.String
		record.Enabled = enabled != 0
		record.Detected = detected != 0
		record.Writable = writable != 0
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Store) UpdateAssignmentState(id, status, installedHash string) error {
	_, err := s.db.Exec(`UPDATE assignments SET status = ?, installed_hash = ?, updated_at = ? WHERE id = ?`,
		status, nullString(installedHash), time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("update assignment state: %w", err)
	}
	return nil
}

func (s *Store) UpdateTargetName(id, name string) error {
	_, err := s.db.Exec(`UPDATE targets SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("update target name: %w", err)
	}
	return nil
}
