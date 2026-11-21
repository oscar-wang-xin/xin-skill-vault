package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	dirName      = "XinSkillVault"
	configName   = "config.json"
	defaultTheme = "mech-gray"
)

// Config is the persisted application configuration. Paths are absolute after Normalize.
type Config struct {
	VaultPath  string `json:"vaultPath"`
	PromptRoot string `json:"promptRoot"`
	Theme      string `json:"theme"`
	Neumorph   bool   `json:"neumorph"`
	Language   string `json:"language"`
}

// Manager owns loading and atomic saving of the config file.
type Manager struct {
	path string
	mu   sync.RWMutex
	cfg  Config
}

func UserConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, dirName), nil
}

func Default() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolve home dir: %w", err)
	}
	return Config{
		VaultPath:  filepath.Join(home, ".skill-repo"),
		PromptRoot: filepath.Join(home, ".xin-skill-vault", "prompts"),
		Theme:      defaultTheme,
		Neumorph:   true,
		Language:   "zh-CN",
	}, nil
}

// NewManager loads the config at configPath, creating defaults when it is missing.
func NewManager(configPath string) (*Manager, error) {
	if strings.TrimSpace(configPath) == "" {
		return nil, errors.New("config path is empty")
	}
	cfg, err := Default()
	if err != nil {
		return nil, err
	}
	manager := &Manager{path: configPath, cfg: cfg}
	if err := manager.load(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config: %w", err)
		}
		return m.saveLocked(m.cfg)
	}
	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	m.cfg = normalize(loaded, m.cfg)
	return nil
}

func (m *Manager) Current() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Path() string { return m.path }

// Save validates and atomically writes the configuration.
func (m *Manager) Save(next Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	next = normalize(next, m.cfg)
	if next.VaultPath == next.PromptRoot {
		return errors.New("vault path and prompt root must differ")
	}
	if err := m.saveLocked(next); err != nil {
		return err
	}
	m.cfg = next
	return nil
}

func (m *Manager) saveLocked(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(m.path), ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}
	if err := os.Rename(tmpName, m.path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func normalize(cfg, fallback Config) Config {
	cfg.VaultPath = cleanPath(cfg.VaultPath, fallback.VaultPath)
	cfg.PromptRoot = cleanPath(cfg.PromptRoot, fallback.PromptRoot)
	if strings.TrimSpace(cfg.Theme) == "" {
		cfg.Theme = fallback.Theme
	}
	if strings.TrimSpace(cfg.Language) == "" {
		cfg.Language = fallback.Language
	}
	return cfg
}

func cleanPath(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if value == "~" || strings.HasPrefix(value, "~"+string(filepath.Separator)) || strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			value = filepath.Join(home, strings.TrimLeft(value[1:], "/\\"))
		}
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return filepath.Clean(abs)
}
