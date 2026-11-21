package distribute

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"xinskillvault/internal/store"
	"xinskillvault/internal/vault"
)

type Distributor struct {
	Vault *vault.Vault
	Store *store.Store
}

type Manifest struct {
	Version     int    `json:"version"`
	Skill       string `json:"skill"`
	ContentHash string `json:"contentHash"`
	VaultPath   string `json:"vaultPath"`
	Mode        string `json:"mode"`
	InstalledAt int64  `json:"installedAt"`
}

func (d *Distributor) Install(target store.TargetRecord, slug, mode string) (store.AssignmentRecord, error) {
	var zero store.AssignmentRecord
	if mode == "symlink" {
		mode = "link"
	}
	if mode != "link" && mode != "copy" {
		mode = target.DefaultMode
	}
	if mode == "symlink" {
		mode = "link"
	}
	if mode != "link" && mode != "copy" {
		mode = "copy"
	}
	record, err := d.Store.FindSkillBySlug(slug)
	if err != nil {
		return zero, err
	}
	if record == nil {
		return zero, fmt.Errorf("技能不存在：%s", slug)
	}
	sourceDir, err := d.Vault.SkillDir(slug)
	if err != nil {
		return zero, err
	}
	if err := os.MkdirAll(target.Path, 0o755); err != nil {
		return zero, fmt.Errorf("create agent directory: %w", err)
	}
	dest := filepath.Join(target.Path, slug)
	existing, err := d.Store.FindAssignment(record.ID, target.ID)
	if err != nil {
		return zero, err
	}
	if _, err := os.Lstat(dest); err == nil {
		if existing == nil || existing.InstalledPath != dest {
			return zero, fmt.Errorf("目标目录已存在同名技能：%s", dest)
		}
		if err := removeInstalled(dest); err != nil {
			return zero, fmt.Errorf("remove previous install: %w", err)
		}
	}
	actualMode := mode
	if mode == "link" {
		if err := os.Symlink(sourceDir, dest); err != nil {
			actualMode = "copy"
		}
	}
	if actualMode == "copy" {
		temp, err := os.MkdirTemp(target.Path, ".xsv-install-*")
		if err != nil {
			return zero, fmt.Errorf("create install temp: %w", err)
		}
		cleanup := func() { _ = os.RemoveAll(temp) }
		if err := vault.CopyDir(sourceDir, temp); err != nil {
			cleanup()
			return zero, err
		}
		manifest := Manifest{
			Version: 1, Skill: record.Slug, ContentHash: record.ContentHash,
			VaultPath: d.Vault.Root, Mode: actualMode, InstalledAt: time.Now().Unix(),
		}
		data, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(filepath.Join(temp, ".skillrepo-manifest.json"), data, 0o644); err != nil {
			cleanup()
			return zero, fmt.Errorf("write manifest: %w", err)
		}
		if err := os.Rename(temp, dest); err != nil {
			cleanup()
			return zero, fmt.Errorf("move install into place: %w", err)
		}
	}
	now := time.Now().Unix()
	assignment := store.AssignmentRecord{
		ID: existingID(existing, record.ID, target.ID), SkillID: record.ID, TargetID: target.ID,
		Mode: actualMode, SourcePath: sourceDir, SourceHash: record.ContentHash,
		InstalledPath: dest, InstalledHash: record.ContentHash, Status: "ok",
		AssignedAt: now, UpdatedAt: now,
	}
	if existing != nil {
		assignment.AssignedAt = existing.AssignedAt
	}
	if err := d.Store.UpsertAssignment(assignment); err != nil {
		_ = removeInstalled(dest)
		return zero, err
	}
	return assignment, nil
}

func (d *Distributor) Uninstall(skillID, targetID string) error {
	assignment, err := d.Store.FindAssignment(skillID, targetID)
	if err != nil {
		return err
	}
	if assignment == nil {
		return fmt.Errorf("没有找到分发记录")
	}
	if err := removeInstalled(assignment.InstalledPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove installed skill: %w", err)
	}
	return d.Store.DeleteAssignment(assignment.ID)
}

func removeInstalled(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	return os.RemoveAll(path)
}

func existingID(existing *store.AssignmentRecord, skillID, targetID string) string {
	if existing != nil {
		return existing.ID
	}
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("as_%d", time.Now().UnixNano())
	}
	return "as_" + hex.EncodeToString(buf)
}

// RemoveTarget removes every assignment installed on a target and deletes the target row.
func (d *Distributor) RemoveTarget(targetID string) error {
	assignments, err := d.Store.ListAssignmentsByTarget(targetID)
	if err != nil {
		return err
	}
	for _, assignment := range assignments {
		if err := removeInstalled(assignment.InstalledPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove installed skill: %w", err)
		}
	}
	return d.Store.DeleteTarget(targetID)
}
