CREATE TABLE IF NOT EXISTS bundles (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS bundle_members (
  bundle_id  TEXT NOT NULL REFERENCES bundles(id) ON DELETE CASCADE,
  skill_id   TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY (bundle_id, skill_id)
);
