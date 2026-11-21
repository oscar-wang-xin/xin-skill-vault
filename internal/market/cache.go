package market

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	Dir string
}

func (c Cache) Get(key string, maxAge time.Duration, target any) bool {
	if c.Dir == "" {
		return false
	}
	path := c.path(key)
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > maxAge {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(data, target) == nil
}

func (c Cache) Put(key string, value any) error {
	if c.Dir == "" {
		return nil
	}
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	path := c.path(key)
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}

func (c Cache) path(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.Dir, hex.EncodeToString(sum[:])+".json")
}
