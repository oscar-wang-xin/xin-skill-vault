//go:build !windows

package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateDirLink creates a relative symlink so a vault moved between disks keeps
// working as long as the agent and vault stay on the same volume.
func CreateDirLink(link, target string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	rel, err := filepath.Rel(filepath.Dir(link), target)
	if err != nil {
		rel = target
	}
	if err := os.Symlink(rel, link); err != nil {
		return fmt.Errorf("创建 Symlink 失败：%w", err)
	}
	return nil
}

func LinkMode() string { return "symlink" }

func IsLink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func LinkTarget(path string) (string, bool) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", false
	}
	return target, true
}

func RemoveLink(path string) error { return os.Remove(path) }

func SameLocation(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(ai, bi)
}
