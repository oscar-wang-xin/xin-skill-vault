package skillpack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ManifestName = ".skillrepo-manifest.json"

type Manifest struct {
	Format      string   `json:"format"`
	ExportedAt  int64    `json:"exportedAt"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	ContentHash string   `json:"contentHash"`
	Files       []string `json:"files"`
}

var ignored = map[string]bool{
	".git": true, "node_modules": true, ".skillrepo": true,
	".DS_Store": true, "Thumbs.db": true, ManifestName: true,
}

func Export(sourceDir, destination, slug, name, version, description, contentHash string) error {
	var files []string
	if err := filepath.WalkDir(sourceDir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ignored[entry.Name()] {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(files)
	manifest := Manifest{
		Format: "xsv-skill-v1", ExportedAt: time.Now().Unix(), Slug: slug, Name: name,
		Version: version, Description: description, ContentHash: contentHash, Files: files,
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".skillpack-*.tmp")
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
	writeFile := func(rel, source string) error {
		writer, err := archive.Create(rel)
		if err != nil {
			return err
		}
		input, err := os.Open(source)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		input.Close()
		return copyErr
	}
	for _, rel := range files {
		if err := writeFile(rel, filepath.Join(sourceDir, filepath.FromSlash(rel))); err != nil {
			archive.Close()
			file.Close()
			return fmt.Errorf("pack %s: %w", rel, err)
		}
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if writer, err := archive.Create(ManifestName); err == nil {
		writer.Write(manifestData)
	}
	if err := archive.Close(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, destination); err != nil {
		return err
	}
	return nil
}

func SafeFilename(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	value = replacer.Replace(value)
	if value == "" {
		return "skill"
	}
	return value
}
