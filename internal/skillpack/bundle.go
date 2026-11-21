package skillpack

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type BundleSkill struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	ContentHash string `json:"contentHash"`
	Dir         string `json:"-"`
}

type BundleManifest struct {
	Format     string        `json:"format"`
	Name       string        `json:"name"`
	ExportedAt int64         `json:"exportedAt"`
	Skills     []BundleSkill `json:"skills"`
}

// ExportBundle writes multiple skill directories under skills/<slug>/ plus bundle.json.
func ExportBundle(destination, name string, skills []BundleSkill) error {
	sort.Slice(skills, func(i, j int) bool { return skills[i].Slug < skills[j].Slug })
	temp, err := os.CreateTemp(filepath.Dir(destination), ".bundle-*.tmp")
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
	manifest := BundleManifest{Format: "xsv-bundle-v1", Name: name, ExportedAt: time.Now().Unix(), Skills: skills}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	if writer, err := archive.Create("bundle.json"); err == nil {
		writer.Write(data)
	}
	for _, skill := range skills {
		var files []string
		if err := filepath.WalkDir(skill.Dir, func(current string, entry os.DirEntry, walkErr error) error {
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
			rel, err := filepath.Rel(skill.Dir, current)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		}); err != nil {
			archive.Close()
			file.Close()
			return fmt.Errorf("walk %s: %w", skill.Slug, err)
		}
		sort.Strings(files)
		for _, rel := range files {
			writer, err := archive.Create("skills/" + skill.Slug + "/" + rel)
			if err != nil {
				archive.Close()
				file.Close()
				return err
			}
			input, err := os.Open(filepath.Join(skill.Dir, filepath.FromSlash(rel)))
			if err != nil {
				archive.Close()
				file.Close()
				return err
			}
			_, copyErr := io.Copy(writer, input)
			input.Close()
			if copyErr != nil {
				archive.Close()
				file.Close()
				return copyErr
			}
		}
	}
	if err := archive.Close(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, destination)
}
