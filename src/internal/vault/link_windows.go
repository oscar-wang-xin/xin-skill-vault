//go:build windows

package vault

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CreateDirLink creates a Windows Junction. Junctions do not require
// Developer Mode or SeCreateSymbolicLinkPrivilege, so every v1 install works
// without elevation.
func CreateDirLink(link, target string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("创建 Junction 失败：%s", msg)
	}
	return nil
}

func LinkMode() string { return "junction" }

func IsLink(path string) bool {
	_, err := os.Readlink(path)
	return err == nil
}

func LinkTarget(path string) (string, bool) {
	target, err := os.Readlink(path)
	if err != nil {
		return "", false
	}
	return target, true
}

func RemoveLink(path string) error { return os.Remove(path) }

// SameLocation compares two paths after following links. os.SameFile handles
// junctions that report short (8.3) paths from Readlink.
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
