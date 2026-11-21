-- Xin Skill Vault initial index schema.
-- Filesystem remains the source of truth; this database is rebuildable.

CREATE TABLE IF NOT EXISTS skills (
  id            TEXT PRIMARY KEY,
  slug          TEXT NOT NULL UNIQUE,
  name          TEXT NOT NULL,
  description   TEXT NOT NULL DEFAULT '',
  version       TEXT,
  author        TEXT,
  license       TEXT,
  icon          TEXT,
  path          TEXT NOT NULL,
  content_hash  TEXT NOT NULL,
  size_bytes    INTEGER NOT NULL DEFAULT 0,
  file_count    INTEGER NOT NULL DEFAULT 0,
  enabled       INTEGER NOT NULL DEFAULT 1,
  source_type   TEXT NOT NULL,
  source_ref    TEXT,
  source_rev    TEXT,
  source_url    TEXT,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL,
  deleted_at    INTEGER
);
CREATE INDEX IF NOT EXISTS idx_skills_hash ON skills(content_hash);
CREATE INDEX IF NOT EXISTS idx_skills_updated ON skills(updated_at DESC);

CREATE TABLE IF NOT EXISTS tags (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  color      TEXT,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS skill_tags (
  skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  tag_id   TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (skill_id, tag_id)
);

CREATE TABLE IF NOT EXISTS groups (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  description TEXT,
  color       TEXT,
  icon        TEXT,
  kind        TEXT NOT NULL DEFAULT 'manual',
  query       TEXT,
  sort_order  INTEGER NOT NULL DEFAULT 0,
  created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS skill_groups (
  skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  PRIMARY KEY (skill_id, group_id)
);

CREATE TABLE IF NOT EXISTS targets (
  id           TEXT PRIMARY KEY,
  kind         TEXT NOT NULL,
  agent_id     TEXT,
  name         TEXT NOT NULL,
  path         TEXT NOT NULL,
  config_path  TEXT,
  platform     TEXT,
  enabled      INTEGER NOT NULL DEFAULT 1,
  detected     INTEGER NOT NULL DEFAULT 0,
  writable     INTEGER NOT NULL DEFAULT 1,
  default_mode TEXT NOT NULL DEFAULT 'symlink',
  last_scan_at INTEGER,
  created_at   INTEGER NOT NULL,
  UNIQUE(kind, path)
);

CREATE TABLE IF NOT EXISTS assignments (
  id             TEXT PRIMARY KEY,
  skill_id       TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  target_id      TEXT NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
  mode           TEXT NOT NULL,
  source_path    TEXT NOT NULL,
  source_hash    TEXT NOT NULL,
  origin_url     TEXT,
  installed_path TEXT NOT NULL,
  installed_hash TEXT,
  status         TEXT NOT NULL DEFAULT 'ok',
  assigned_at    INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL,
  UNIQUE(skill_id, target_id)
);
CREATE INDEX IF NOT EXISTS idx_assignments_target ON assignments(target_id, status);

CREATE TABLE IF NOT EXISTS prompt_categories (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  parent_id  TEXT REFERENCES prompt_categories(id) ON DELETE SET NULL,
  dir_path   TEXT NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS prompts (
  id              TEXT PRIMARY KEY,
  title           TEXT NOT NULL,
  category_id     TEXT REFERENCES prompt_categories(id) ON DELETE SET NULL,
  tags_json       TEXT NOT NULL DEFAULT '[]',
  uses            INTEGER NOT NULL DEFAULT 0,
  current_version INTEGER NOT NULL DEFAULT 1,
  file_path       TEXT NOT NULL,
  current_hash    TEXT NOT NULL,
  created_at      INTEGER NOT NULL,
  updated_at      INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_prompts_category ON prompts(category_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS prompt_versions (
  id           TEXT PRIMARY KEY,
  prompt_id    TEXT NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
  version      INTEGER NOT NULL,
  title        TEXT NOT NULL,
  category_id  TEXT,
  tags_json    TEXT NOT NULL DEFAULT '[]',
  file_path    TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  note         TEXT,
  created_at   INTEGER NOT NULL,
  UNIQUE(prompt_id, version)
);
CREATE INDEX IF NOT EXISTS idx_prompt_versions ON prompt_versions(prompt_id, version DESC);

CREATE TABLE IF NOT EXISTS notes (
  id         TEXT PRIMARY KEY,
  skill_id   TEXT REFERENCES skills(id) ON DELETE CASCADE,
  title      TEXT,
  body       TEXT NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_log (
  id          TEXT PRIMARY KEY,
  action      TEXT NOT NULL,
  entity      TEXT,
  entity_id   TEXT,
  target_id   TEXT,
  result      TEXT NOT NULL,
  detail_json TEXT,
  actor       TEXT NOT NULL DEFAULT 'ui',
  duration_ms INTEGER,
  created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_log(created_at DESC);

CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS skills_fts USING fts5(
  name, description, body, content='', tokenize='unicode61'
);
