package app

import (
	"fmt"
	"os"
	"path/filepath"

	"xinskillvault/internal/config"
)

// EnsureLayout creates the directories that must exist before the UI can work.
func EnsureLayout(cfg config.Config) error {
	dirs := []string{
		filepath.Join(cfg.VaultPath, "skills"),
		filepath.Join(cfg.VaultPath, "bundles"),
		filepath.Join(cfg.VaultPath, ".skillrepo", "cache"),
		filepath.Join(cfg.VaultPath, ".skillrepo", "trash"),
		filepath.Join(cfg.VaultPath, ".skillrepo", "logs"),
		cfg.PromptRoot,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}
