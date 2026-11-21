CREATE TABLE IF NOT EXISTS snapshots (
  id           TEXT PRIMARY KEY,
  skill_id     TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  content_hash TEXT NOT NULL,
  file_path    TEXT NOT NULL,
  note         TEXT,
  created_at   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_skill ON snapshots(skill_id, created_at DESC);
