package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

const v1Schema = `
CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE sources (id TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL, url TEXT NOT NULL DEFAULT '', builtin INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, synced_at TEXT);
CREATE TABLE skills (id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', source_id TEXT REFERENCES sources(id) ON DELETE SET NULL, source_ref TEXT NOT NULL DEFAULT '', vault_path TEXT NOT NULL, content_hash TEXT NOT NULL DEFAULT '', size_bytes INTEGER NOT NULL DEFAULT 0, file_count INTEGER NOT NULL DEFAULT 0, risk TEXT NOT NULL DEFAULT 'low', version TEXT NOT NULL DEFAULT '', installed_at TEXT NOT NULL, updated_at TEXT NOT NULL, last_checked_at TEXT, update_available INTEGER NOT NULL DEFAULT 0);
CREATE TABLE agents (id TEXT PRIMARY KEY, name TEXT NOT NULL, skills_path TEXT NOT NULL, config_path TEXT NOT NULL DEFAULT '', detected INTEGER NOT NULL DEFAULT 0, enabled INTEGER NOT NULL DEFAULT 1, sort_order INTEGER NOT NULL DEFAULT 0);
CREATE TABLE distributions (id INTEGER PRIMARY KEY AUTOINCREMENT, skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE, agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE, mode TEXT NOT NULL, link_path TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'ok', content_hash TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(skill_id, agent_id));
CREATE TABLE activity (id INTEGER PRIMARY KEY AUTOINCREMENT, action TEXT NOT NULL, subject TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL);
PRAGMA user_version = 1;
`

func TestMigrateV1ToV4(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(v1Schema); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO sources(id,name,kind,created_at) VALUES('local','本地仓库','local','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO skills(id,name,vault_path,installed_at,updated_at) VALUES('old','old','C:\\vault\\old','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO agents(id,name,skills_path) VALUES('test','Test','C:\\agent\\skills')`); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(`INSERT INTO settings(key,value,updated_at) VALUES('security.blockHigh','true','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	st, err := Open(path)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	defer st.Close()
	if st.SchemaVersion() != 4 {
		t.Fatalf("schema version = %d", st.SchemaVersion())
	}
	if _, ok := st.GetSetting("security.blockHigh"); ok {
		t.Fatalf("legacy security setting survived migration")
	}
	skill, err := st.GetSkill("old")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Notes != "" || skill.Content != "none" {
		t.Fatalf("unexpected skill defaults: %+v", skill)
	}
	agent, err := st.GetAgent("test")
	if err != nil {
		t.Fatal(err)
	}
	if agent.Kind != "agent" {
		t.Fatalf("kind = %q", agent.Kind)
	}
	if err := st.CreateSkillGroup(SkillGroup{ID: "g", Name: "Group", CreatedAt: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpdateSkillMeta("old", "note", "g", []string{"tag"}); err != nil {
		t.Fatal(err)
	}
	skill, _ = st.GetSkill("old")
	if skill.Notes != "note" || skill.GroupID != "g" {
		t.Fatalf("metadata not saved: %+v", skill)
	}
	tags := st.AllSkillTags()
	if len(tags["old"]) != 1 || tags["old"][0] != "tag" {
		t.Fatalf("tags = %+v", tags)
	}

	skill.Content = "executable"
	skill.ContentFiles = []string{"scripts/run.ps1（脚本）", "bin/tool.exe（可执行文件）"}
	if err := st.UpsertSkill(skill); err != nil {
		t.Fatal(err)
	}
	skill, err = st.GetSkill("old")
	if err != nil {
		t.Fatal(err)
	}
	if skill.Content != "executable" || len(skill.ContentFiles) != 2 || skill.ContentFiles[1] != "bin/tool.exe（可执行文件）" {
		t.Fatalf("content metadata not persisted: %+v", skill)
	}
}
