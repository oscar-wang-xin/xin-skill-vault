package vault

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDir copies a skill directory into place. Symlinks are dereferenced so a
// copy distribution never depends on the original source path.
func CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return copyDirContents(src, dst, 0)
}

func copyDirContents(src, dst string, depth int) error {
	if depth > 32 {
		return fmt.Errorf("目录层级过深，疑似存在循环链接")
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if SkipName(name) {
			continue
		}
		srcPath := filepath.Join(src, name)
		dstPath := filepath.Join(dst, name)

		info, err := os.Lstat(srcPath)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			real, err := filepath.EvalSymlinks(srcPath)
			if err != nil {
				return fmt.Errorf("解析链接 %s 失败: %w", name, err)
			}
			realInfo, err := os.Stat(real)
			if err != nil {
				return err
			}
			if realInfo.IsDir() {
				if err := os.MkdirAll(dstPath, 0o755); err != nil {
					return err
				}
				if err := copyDirContents(real, dstPath, depth+1); err != nil {
					return err
				}
			} else {
				if err := copyFile(real, dstPath, realInfo.Mode()); err != nil {
					return err
				}
			}
			continue
		}
		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			if err := copyDirContents(srcPath, dstPath, depth+1); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if mode&0o111 != 0 {
		perm = 0o755
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func DirSize(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && SkipName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}
