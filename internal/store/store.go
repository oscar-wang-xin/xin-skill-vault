package store

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Store struct {
	db   *sql.DB
	path string
}

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("database path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create database dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db, path: path}
	if err := s.configure(); err != nil {
		db.Close()
		return nil, err
	}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) configure() error {
	for _, statement := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA synchronous = NORMAL",
	} {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("configure sqlite (%s): %w", statement, err)
		}
	}
	return nil
}

func (s *Store) migrate() error {
	var current int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	names, err := migrationNames()
	if err != nil {
		return err
	}
	for index, name := range names {
		target := index + 1
		if target <= current {
			continue
		}
		if current > 0 {
			if _, err := s.db.Exec("PRAGMA wal_checkpoint(FULL)"); err != nil {
				return fmt.Errorf("checkpoint before backup: %w", err)
			}
			if err := backupDatabase(s.path); err != nil {
				return err
			}
		}
		body, err := migrationFiles.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", target)); err != nil {
			tx.Rollback()
			return fmt.Errorf("set schema version %d: %w", target, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
		current = target
	}
	return nil
}

func migrationNames() ([]string, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func backupDatabase(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat database for backup: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}
	backup := fmt.Sprintf("%s.bak.%s", path, time.Now().UTC().Format("20060102T150405Z"))
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open database for backup: %w", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(backup, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create database backup: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(backup)
		return fmt.Errorf("copy database backup: %w", err)
	}
	if err := dst.Sync(); err != nil {
		dst.Close()
		os.Remove(backup)
		return fmt.Errorf("sync database backup: %w", err)
	}
	return dst.Close()
}

func (s *Store) DB() *sql.DB  { return s.db }
func (s *Store) Path() string { return s.path }

func (s *Store) SchemaVersion() (int, error) {
	var version int
	err := s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func (s *Store) SkillCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM skills WHERE deleted_at IS NULL").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count skills: %w", err)
	}
	return count, nil
}

func (s *Store) TargetCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM targets WHERE enabled = 1").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count targets: %w", err)
	}
	return count, nil
}

func (s *Store) AssignmentCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM assignments WHERE status != 'missing'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count assignments: %w", err)
	}
	return count, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}
