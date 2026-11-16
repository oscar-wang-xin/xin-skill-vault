package store

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

const schemaVersion = 4

type Store struct {
	db   *sql.DB
	Path string
}

type Agent struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"` // agent | project
	SkillsPath string `json:"skillsPath"`
	ConfigPath string `json:"configPath"`
	Detected   bool   `json:"detected"`
	Enabled    bool   `json:"enabled"`
	SortOrder  int    `json:"sortOrder"`
}

type Source struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	URL       string `json:"url"`
	Builtin   bool   `json:"builtin"`
	CreatedAt string `json:"createdAt"`
	SyncedAt  string `json:"syncedAt"`
}

type Skill struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Notes           string   `json:"notes"`
	SourceID        string   `json:"sourceId"`
	SourceRef       string   `json:"sourceRef"`
	GroupID         string   `json:"groupId"`
	VaultPath       string   `json:"vaultPath"`
	ContentHash     string   `json:"contentHash"`
	SizeBytes       int64    `json:"sizeBytes"`
	FileCount       int      `json:"fileCount"`
	Content         string   `json:"content"`
	ContentFiles    []string `json:"contentFiles"`
	Version         string   `json:"version"`
	InstalledAt     string   `json:"installedAt"`
	UpdatedAt       string   `json:"updatedAt"`
	LastCheckedAt   string   `json:"lastCheckedAt"`
	UpdateAvailable bool     `json:"updateAvailable"`
}

type SkillGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
}

type Distribution struct {
	SkillID     string `json:"skillId"`
	AgentID     string `json:"agentId"`
	Mode        string `json:"mode"`
	LinkPath    string `json:"linkPath"`
	Status      string `json:"status"`
	ContentHash string `json:"contentHash"`
	UpdatedAt   string `json:"updatedAt"`
}

type Activity struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Subject   string `json:"subject"`
	Detail    string `json:"detail"`
	CreatedAt string `json:"createdAt"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	// A single connection keeps writes serialized and avoids SQLITE_BUSY on
	// Windows where the database may sit on a slow or filtered volume.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	s := &Store{db: db, Path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func filepathDir(path string) string {
	return filepath.Dir(path)
}

func (s *Store) migrate() error {
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("读取数据库版本失败: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("数据库版本 %d 高于应用支持的版本 %d，请升级 Xin Skill Vault", version, schemaVersion)
	}
	if version < 1 {
		if _, err := s.db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
		if _, err := s.db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
			return fmt.Errorf("写入数据库版本失败: %w", err)
		}
		return nil
	}
	if version == 1 {
		if _, err := s.db.Exec(migrationV2SQL); err != nil {
			return fmt.Errorf("升级数据库到 v2 失败: %w", err)
		}
		if _, err := s.db.Exec(`PRAGMA user_version = 2`); err != nil {
			return fmt.Errorf("写入数据库版本失败: %w", err)
		}
		version = 2
	}
	if version == 2 {
		if _, err := s.db.Exec(migrationV3SQL); err != nil {
			return fmt.Errorf("升级数据库到 v3 失败: %w", err)
		}
		if _, err := s.db.Exec(`PRAGMA user_version = 3`); err != nil {
			return fmt.Errorf("写入数据库版本失败: %w", err)
		}
		version = 3
	}
	if version == 3 {
		if _, err := s.db.Exec(migrationV4SQL); err != nil {
			return fmt.Errorf("升级数据库到 v4 失败: %w", err)
		}
		if _, err := s.db.Exec(`PRAGMA user_version = 4`); err != nil {
			return fmt.Errorf("写入数据库版本失败: %w", err)
		}
	}
	return nil
}

const migrationV2SQL = `
CREATE TABLE IF NOT EXISTS skill_groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
ALTER TABLE skills ADD COLUMN notes TEXT NOT NULL DEFAULT '';
ALTER TABLE skills ADD COLUMN group_id TEXT REFERENCES skill_groups(id) ON DELETE SET NULL;
CREATE TABLE IF NOT EXISTS skill_tags (
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY(skill_id, tag)
);
ALTER TABLE agents ADD COLUMN kind TEXT NOT NULL DEFAULT 'agent';
CREATE INDEX IF NOT EXISTS idx_skills_group ON skills(group_id);
CREATE INDEX IF NOT EXISTS idx_skill_tags_tag ON skill_tags(tag);
`

const migrationV3SQL = `
ALTER TABLE skills ADD COLUMN risk_reasons TEXT NOT NULL DEFAULT '[]';
`

const migrationV4SQL = `
ALTER TABLE skills ADD COLUMN content TEXT NOT NULL DEFAULT 'none';
ALTER TABLE skills ADD COLUMN content_files TEXT NOT NULL DEFAULT '[]';
ALTER TABLE skills DROP COLUMN risk;
ALTER TABLE skills DROP COLUMN risk_reasons;
DELETE FROM settings WHERE key IN ('security.risk', 'security.blockHigh');
`

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) SQLiteVersion() string {
	var v string
	if err := s.db.QueryRow(`SELECT sqlite_version()`).Scan(&v); err != nil {
		return "unknown"
	}
	return v
}

func (s *Store) SchemaVersion() int { return schemaVersion }

// ---------- settings ----------

func (s *Store) GetSetting(key string) (string, bool) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return "", false
	}
	return v, true
}

func (s *Store) GetSettingBool(key string, fallback bool) bool {
	v, ok := s.GetSetting(key)
	if !ok {
		return fallback
	}
	return v != "false" && v != "0"
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings(key, value, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now())
	return err
}

func (s *Store) AllSettings() map[string]string {
	out := map[string]string{}
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			out[k] = v
		}
	}
	return out
}

// ---------- sources ----------

func (s *Store) SeedSources(sources []Source) error {
	for _, src := range sources {
		_, err := s.db.Exec(`INSERT INTO sources(id, name, kind, url, builtin, created_at)
			VALUES(?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name = excluded.name, kind = excluded.kind, url = excluded.url, builtin = excluded.builtin`,
			src.ID, src.Name, src.Kind, src.URL, boolInt(src.Builtin), firstNonEmpty(src.CreatedAt, now()))
		if err != nil {
			return fmt.Errorf("写入来源 %s 失败: %w", src.ID, err)
		}
	}
	return nil
}

func (s *Store) ListSources() ([]Source, error) {
	rows, err := s.db.Query(`SELECT id, name, kind, url, builtin, created_at, COALESCE(synced_at, '')
		FROM sources ORDER BY builtin DESC, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		var src Source
		var builtin int
		if err := rows.Scan(&src.ID, &src.Name, &src.Kind, &src.URL, &builtin, &src.CreatedAt, &src.SyncedAt); err != nil {
			return nil, err
		}
		src.Builtin = builtin == 1
		out = append(out, src)
	}
	return out, rows.Err()
}

func (s *Store) GetSource(id string) (Source, error) {
	var src Source
	var builtin int
	err := s.db.QueryRow(`SELECT id, name, kind, url, builtin, created_at, COALESCE(synced_at, '')
		FROM sources WHERE id = ?`, id).
		Scan(&src.ID, &src.Name, &src.Kind, &src.URL, &builtin, &src.CreatedAt, &src.SyncedAt)
	src.Builtin = builtin == 1
	return src, err
}

func (s *Store) SourceExists(id string) bool {
	var one int
	return s.db.QueryRow(`SELECT 1 FROM sources WHERE id = ?`, id).Scan(&one) == nil
}

func (s *Store) AddSource(src Source) error {
	_, err := s.db.Exec(`INSERT INTO sources(id, name, kind, url, builtin, created_at) VALUES(?, ?, ?, ?, 0, ?)`,
		src.ID, src.Name, src.Kind, src.URL, firstNonEmpty(src.CreatedAt, now()))
	if err != nil {
		return fmt.Errorf("添加来源失败: %w", err)
	}
	return nil
}

func (s *Store) DeleteSource(id string) error {
	_, err := s.db.Exec(`DELETE FROM sources WHERE id = ? AND builtin = 0`, id)
	return err
}

func (s *Store) CountSkillsBySource(id string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM skills WHERE source_id = ?`, id).Scan(&count)
	return count, err
}

// ---------- agents ----------

func (s *Store) SeedAgents(agents []Agent) error {
	for _, a := range agents {
		kind := a.Kind
		if kind == "" {
			kind = "agent"
		}
		enabled := a.Enabled
		if !a.Detected {
			// Keep undetected agents visible, but out of the default fan-out.
			enabled = false
		}
		_, err := s.db.Exec(`INSERT INTO agents(id, name, kind, skills_path, config_path, detected, enabled, sort_order)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name = excluded.name, kind = excluded.kind,
				skills_path = excluded.skills_path, config_path = excluded.config_path,
				detected = excluded.detected, sort_order = excluded.sort_order`,
			a.ID, a.Name, kind, a.SkillsPath, a.ConfigPath, boolInt(a.Detected), boolInt(enabled), a.SortOrder)
		if err != nil {
			return fmt.Errorf("写入 Agent %s 失败: %w", a.ID, err)
		}
	}
	return nil
}

func (s *Store) AddAgent(a Agent) error {
	kind := a.Kind
	if kind == "" {
		kind = "agent"
	}
	_, err := s.db.Exec(`INSERT INTO agents(id, name, kind, skills_path, config_path, detected, enabled, sort_order)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, kind, a.SkillsPath, a.ConfigPath, boolInt(a.Detected), boolInt(a.Enabled), a.SortOrder)
	return err
}

func (s *Store) DeleteAgent(id string) error {
	_, err := s.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	return err
}

func (s *Store) CountDistributionsForAgent(id string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM distributions WHERE agent_id = ?`, id).Scan(&count)
	return count, err
}

func (s *Store) ListAgents() ([]Agent, error) {
	rows, err := s.db.Query(`SELECT id, name, kind, skills_path, config_path, detected, enabled, sort_order
		FROM agents ORDER BY CASE kind WHEN 'agent' THEN 0 ELSE 1 END,
			detected DESC, enabled DESC, sort_order, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Agent
	for rows.Next() {
		var a Agent
		var detected, enabled int
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &a.SkillsPath, &a.ConfigPath, &detected, &enabled, &a.SortOrder); err != nil {
			return nil, err
		}
		a.Detected = detected == 1
		a.Enabled = enabled == 1
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAgent(id string) (Agent, error) {
	var a Agent
	var detected, enabled int
	err := s.db.QueryRow(`SELECT id, name, kind, skills_path, config_path, detected, enabled, sort_order
		FROM agents WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.Kind, &a.SkillsPath, &a.ConfigPath, &detected, &enabled, &a.SortOrder)
	if err != nil {
		return a, err
	}
	a.Detected = detected == 1
	a.Enabled = enabled == 1
	return a, nil
}

func (s *Store) SetAgentEnabled(id string, enabled bool) error {
	_, err := s.db.Exec(`UPDATE agents SET enabled = ? WHERE id = ?`, boolInt(enabled), id)
	return err
}

func (s *Store) MarkAgentDetected(id string, detected bool) error {
	_, err := s.db.Exec(`UPDATE agents SET detected = ? WHERE id = ?`, boolInt(detected), id)
	return err
}

// ---------- skills ----------

func (s *Store) UpsertSkill(skill Skill) error {
	nowValue := firstNonEmpty(skill.UpdatedAt, now())
	_, err := s.db.Exec(`INSERT INTO skills(
			id, name, description, notes, source_id, source_ref, group_id, vault_path, content_hash,
			size_bytes, file_count, content, content_files, version, installed_at, updated_at,
			last_checked_at, update_available)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			vault_path = excluded.vault_path,
			content_hash = excluded.content_hash,
			size_bytes = excluded.size_bytes,
			file_count = excluded.file_count,
			content = excluded.content,
			content_files = excluded.content_files,
			version = excluded.version,
			updated_at = excluded.updated_at,
			notes = CASE WHEN excluded.notes <> '' THEN excluded.notes ELSE skills.notes END,
			group_id = CASE WHEN excluded.group_id <> '' THEN excluded.group_id ELSE skills.group_id END,
			source_id = CASE WHEN excluded.source_id <> '' THEN excluded.source_id ELSE skills.source_id END,
			source_ref = CASE WHEN excluded.source_ref <> '' THEN excluded.source_ref ELSE skills.source_ref END`,
		skill.ID, skill.Name, skill.Description, skill.Notes, skill.SourceID, skill.SourceRef, nullIfEmpty(skill.GroupID), skill.VaultPath, skill.ContentHash,
		skill.SizeBytes, skill.FileCount, skill.Content, encodeStringList(skill.ContentFiles), skill.Version, firstNonEmpty(skill.InstalledAt, nowValue), nowValue,
		nullIfEmpty(skill.LastCheckedAt), boolInt(skill.UpdateAvailable))
	if err != nil {
		return fmt.Errorf("写入技能 %s 失败: %w", skill.ID, err)
	}
	return nil
}

func (s *Store) ListSkills() ([]Skill, error) {
	rows, err := s.db.Query(`SELECT id, name, description, notes, COALESCE(source_id, ''), source_ref,
		COALESCE(group_id, ''), vault_path, content_hash, size_bytes, file_count, content, content_files, version,
		installed_at, updated_at, COALESCE(last_checked_at, ''), update_available
		FROM skills ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Skill
	for rows.Next() {
		var sk Skill
		var update int
		var contentFiles string
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Notes, &sk.SourceID, &sk.SourceRef,
			&sk.GroupID, &sk.VaultPath, &sk.ContentHash, &sk.SizeBytes, &sk.FileCount, &sk.Content, &contentFiles,
			&sk.Version, &sk.InstalledAt, &sk.UpdatedAt, &sk.LastCheckedAt, &update); err != nil {
			return nil, err
		}
		sk.ContentFiles = decodeStringList(contentFiles)
		sk.UpdateAvailable = update == 1
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *Store) GetSkill(id string) (Skill, error) {
	var sk Skill
	var update int
	var contentFiles string
	err := s.db.QueryRow(`SELECT id, name, description, notes, COALESCE(source_id, ''), source_ref,
		COALESCE(group_id, ''), vault_path, content_hash, size_bytes, file_count, content, content_files, version,
		installed_at, updated_at, COALESCE(last_checked_at, ''), update_available
		FROM skills WHERE id = ?`, id).
		Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Notes, &sk.SourceID, &sk.SourceRef,
			&sk.GroupID, &sk.VaultPath, &sk.ContentHash, &sk.SizeBytes, &sk.FileCount, &sk.Content, &contentFiles,
			&sk.Version, &sk.InstalledAt, &sk.UpdatedAt, &sk.LastCheckedAt, &update)
	if err != nil {
		return sk, err
	}
	sk.ContentFiles = decodeStringList(contentFiles)
	sk.UpdateAvailable = update == 1
	return sk, nil
}

func (s *Store) DeleteSkill(id string) error {
	_, err := s.db.Exec(`DELETE FROM skills WHERE id = ?`, id)
	return err
}

func (s *Store) UpdateSkillMeta(skillID, notes, groupID string, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE skills SET notes = ?, group_id = ? WHERE id = ?`,
		notes, nullIfEmpty(groupID), skillID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("技能不存在: %s", skillID)
	}
	if _, err := tx.Exec(`DELETE FROM skill_tags WHERE skill_id = ?`, skillID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO skill_tags(skill_id, tag) VALUES(?, ?)`, skillID, tag); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) AllSkillTags() map[string][]string {
	out := map[string][]string{}
	rows, err := s.db.Query(`SELECT skill_id, tag FROM skill_tags ORDER BY tag COLLATE NOCASE`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var skillID, tag string
		if rows.Scan(&skillID, &tag) == nil {
			out[skillID] = append(out[skillID], tag)
		}
	}
	return out
}

// ---------- skill groups ----------

func (s *Store) ListSkillGroups() ([]SkillGroup, error) {
	rows, err := s.db.Query(`SELECT id, name, sort_order, created_at FROM skill_groups
		ORDER BY sort_order, name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillGroup
	for rows.Next() {
		var g SkillGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GetSkillGroup(id string) (SkillGroup, error) {
	var g SkillGroup
	err := s.db.QueryRow(`SELECT id, name, sort_order, created_at FROM skill_groups WHERE id = ?`, id).
		Scan(&g.ID, &g.Name, &g.SortOrder, &g.CreatedAt)
	return g, err
}

func (s *Store) CreateSkillGroup(g SkillGroup) error {
	_, err := s.db.Exec(`INSERT INTO skill_groups(id, name, sort_order, created_at) VALUES(?, ?, ?, ?)`,
		g.ID, g.Name, g.SortOrder, firstNonEmpty(g.CreatedAt, now()))
	if err != nil {
		return fmt.Errorf("创建分组失败: %w", err)
	}
	return nil
}

func (s *Store) RenameSkillGroup(id, name string) error {
	res, err := s.db.Exec(`UPDATE skill_groups SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("重命名分组失败: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("分组不存在")
	}
	return nil
}

func (s *Store) DeleteSkillGroup(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE skills SET group_id = NULL WHERE group_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM skill_groups WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) CountSkillsInGroup(id string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM skills WHERE group_id = ?`, id).Scan(&count)
	return count, err
}

func (s *Store) PruneSkills(keep map[string]bool) error {
	rows, err := s.db.Query(`SELECT id FROM skills`)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && !keep[id] {
			stale = append(stale, id)
		}
	}
	rows.Close()
	for _, id := range stale {
		if err := s.DeleteSkill(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TouchChecked(ids []string) error {
	stamp := now()
	for _, id := range ids {
		if _, err := s.db.Exec(`UPDATE skills SET last_checked_at = ? WHERE id = ?`, stamp, id); err != nil {
			return err
		}
	}
	return nil
}

// ---------- distributions ----------

func (s *Store) UpsertDistribution(d Distribution) error {
	stamp := firstNonEmpty(d.UpdatedAt, now())
	_, err := s.db.Exec(`INSERT INTO distributions(skill_id, agent_id, mode, link_path, status, content_hash, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(skill_id, agent_id) DO UPDATE SET
			mode = excluded.mode, link_path = excluded.link_path, status = excluded.status,
			content_hash = excluded.content_hash, updated_at = excluded.updated_at`,
		d.SkillID, d.AgentID, d.Mode, d.LinkPath, d.Status, d.ContentHash, stamp, stamp)
	if err != nil {
		return fmt.Errorf("写入分发失败: %w", err)
	}
	return nil
}

func (s *Store) ListDistributions() ([]Distribution, error) {
	rows, err := s.db.Query(`SELECT skill_id, agent_id, mode, link_path, status, content_hash, updated_at
		FROM distributions ORDER BY skill_id, agent_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Distribution
	for rows.Next() {
		var d Distribution
		if err := rows.Scan(&d.SkillID, &d.AgentID, &d.Mode, &d.LinkPath, &d.Status, &d.ContentHash, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetDistribution(skillID, agentID string) (Distribution, error) {
	var d Distribution
	err := s.db.QueryRow(`SELECT skill_id, agent_id, mode, link_path, status, content_hash, updated_at
		FROM distributions WHERE skill_id = ? AND agent_id = ?`, skillID, agentID).
		Scan(&d.SkillID, &d.AgentID, &d.Mode, &d.LinkPath, &d.Status, &d.ContentHash, &d.UpdatedAt)
	return d, err
}

func (s *Store) UpdateDistributionStatus(skillID, agentID, status, contentHash string) error {
	_, err := s.db.Exec(`UPDATE distributions SET status = ?, content_hash = CASE WHEN ? <> '' THEN ? ELSE content_hash END, updated_at = ? WHERE skill_id = ? AND agent_id = ?`,
		status, contentHash, contentHash, now(), skillID, agentID)
	return err
}

func (s *Store) DeleteDistribution(skillID, agentID string) error {
	_, err := s.db.Exec(`DELETE FROM distributions WHERE skill_id = ? AND agent_id = ?`, skillID, agentID)
	return err
}

func (s *Store) DeleteDistributionsForSkill(skillID string) error {
	_, err := s.db.Exec(`DELETE FROM distributions WHERE skill_id = ?`, skillID)
	return err
}

// ---------- activity ----------

func (s *Store) AddActivity(action, subject, detail string) {
	_, _ = s.db.Exec(`INSERT INTO activity(action, subject, detail, created_at) VALUES(?, ?, ?, ?)`,
		action, subject, detail, now())
}

func (s *Store) RecentActivity(limit int) ([]Activity, error) {
	if limit <= 0 {
		limit = 12
	}
	rows, err := s.db.Query(`SELECT id, action, subject, detail, created_at FROM activity
		ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Activity
	for rows.Next() {
		var a Activity
		if err := rows.Scan(&a.ID, &a.Action, &a.Subject, &a.Detail, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------- helpers ----------

func now() string { return time.Now().Format(time.RFC3339) }

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func encodeStringList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeStringList(raw string) []string {
	if raw == "" || raw == "[]" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}
