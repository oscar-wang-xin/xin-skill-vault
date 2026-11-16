-- Xin Skill Vault — SQLite schema v4
-- SKILL.md stays the portable, human-readable metadata inside each skill
-- directory. This database is the local index, settings and distribution state.

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sources (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL,              -- git | registry | local | url
    url        TEXT NOT NULL DEFAULT '',
    builtin    INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    synced_at  TEXT
);

CREATE TABLE IF NOT EXISTS skill_groups (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS skills (
    id               TEXT PRIMARY KEY,     -- stable directory slug
    name             TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    notes            TEXT NOT NULL DEFAULT '',
    source_id        TEXT REFERENCES sources(id) ON DELETE SET NULL,
    source_ref       TEXT NOT NULL DEFAULT '',
    group_id         TEXT REFERENCES skill_groups(id) ON DELETE SET NULL,
    vault_path       TEXT NOT NULL,
    content_hash     TEXT NOT NULL DEFAULT '',
    size_bytes       INTEGER NOT NULL DEFAULT 0,
    file_count       INTEGER NOT NULL DEFAULT 0,
    content          TEXT NOT NULL DEFAULT 'none',
    content_files    TEXT NOT NULL DEFAULT '[]',
    version          TEXT NOT NULL DEFAULT '',
    installed_at     TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    last_checked_at  TEXT,
    update_available INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS skill_tags (
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    tag      TEXT NOT NULL,
    PRIMARY KEY(skill_id, tag)
);

CREATE TABLE IF NOT EXISTS agents (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'agent', -- agent | project
    skills_path TEXT NOT NULL,
    config_path TEXT NOT NULL DEFAULT '',
    detected    INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS distributions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id     TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    agent_id     TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    mode         TEXT NOT NULL,            -- junction | symlink | copy
    link_path    TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'ok', -- ok | broken | drifted | missing
    content_hash TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    UNIQUE(skill_id, agent_id)
);

CREATE TABLE IF NOT EXISTS activity (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    action     TEXT NOT NULL,
    subject    TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_skills_source ON skills(source_id);
CREATE INDEX IF NOT EXISTS idx_skills_group ON skills(group_id);
CREATE INDEX IF NOT EXISTS idx_skill_tags_tag ON skill_tags(tag);
CREATE INDEX IF NOT EXISTS idx_distributions_skill ON distributions(skill_id);
CREATE INDEX IF NOT EXISTS idx_distributions_agent ON distributions(agent_id);
CREATE INDEX IF NOT EXISTS idx_activity_created ON activity(created_at DESC);
