package logx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// New creates a JSON logger that writes to stderr and to the vault log file.
func New(logDir string) (*slog.Logger, func() error, error) {
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log dir: %w", err)
	}
	path := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}
	handler := slog.NewJSONHandler(io.MultiWriter(os.Stderr, file), &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	closeFn := func() error {
		return file.Close()
	}
	return logger, closeFn, nil
}
