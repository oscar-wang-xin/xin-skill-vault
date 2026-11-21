package diagnostics

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Report struct {
	GeneratedAt   int64  `json:"generatedAt"`
	AppName       string `json:"appName"`
	AppVersion    string `json:"appVersion"`
	Platform      string `json:"platform"`
	ConfigPath    string `json:"configPath"`
	VaultPath     string `json:"vaultPath"`
	PromptRoot    string `json:"promptRoot"`
	DatabasePath  string `json:"databasePath"`
	SchemaVersion int    `json:"schemaVersion"`
	LogPath       string `json:"logPath"`
}

func Export(destination string, report Report, config any, logPath string) error {
	temp, err := os.CreateTemp(filepath.Dir(destination), ".diagnostics-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	temp.Close()
	defer os.Remove(tempPath)
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(file)
	write := func(name string, data []byte) error {
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		_, err = writer.Write(data)
		return err
	}
	report.GeneratedAt = time.Now().Unix()
	data, _ := json.MarshalIndent(report, "", "  ")
	if err := write("diagnostics.json", data); err != nil {
		archive.Close()
		file.Close()
		return err
	}
	if config != nil {
		if configData, err := json.MarshalIndent(config, "", "  "); err == nil {
			_ = write("config.json", configData)
		}
	}
	if logPath != "" {
		if input, err := os.Open(logPath); err == nil {
			writer, err := archive.Create("app.log")
			if err == nil {
				_, _ = io.Copy(writer, io.LimitReader(input, 2<<20))
			}
			input.Close()
		}
	}
	if err := write("README.txt", []byte("Xin Skill Vault diagnostics\nGenerated locally; no skill content is included.\n")); err != nil {
		archive.Close()
		file.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("write diagnostics: %w", err)
	}
	return nil
}
