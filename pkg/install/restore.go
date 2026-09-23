package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

var restoreID = regexp.MustCompile(`^[a-f0-9]{64}$`)

// RestorePlan is an inert, content-bound preview of one Codex installation reversal.
type RestorePlan struct {
	APIVersion     string        `json:"apiVersion"`
	Kind           string        `json:"kind"`
	PlanID         string        `json:"planID"`
	OriginalPlanID string        `json:"originalPlanID"`
	Target         string        `json:"target"`
	ConfigPath     string        `json:"configPath"`
	Status         string        `json:"status"`
	Files          []RestoreFile `json:"files"`
	changes        []installfs.Change
	checks         []installfs.Change
}

type RestoreFile struct {
	Path         string `json:"path"`
	Action       string `json:"action"`
	BeforeSHA256 string `json:"beforeSHA256"`
	AfterSHA256  string `json:"afterSHA256,omitempty"`
}

func BuildRestorePlan(target, configPath, originalID string) (RestorePlan, error) {
	if target != "codex@0.154.0" || !restoreID.MatchString(originalID) {
		return RestorePlan{}, fmt.Errorf("restore requires codex@0.154.0 and an original 64-hex install plan ID")
	}
	config, err := installfs.SnapshotFile(configPath)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("inspect restore config: %w", err)
	}
	manifestPath := config.Path + ".profile-mango.manifest.json"
	manifest, err := installfs.SnapshotFile(manifestPath)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("inspect restore manifest: %w", err)
	}
	journalPath := installfs.JournalPath(config.Path, originalID)
	journalFile, err := installfs.SnapshotFile(journalPath)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("inspect install journal: %w", err)
	}
	journal, err := parseRestoreJournal(journalFile, originalID, config.Path, manifestPath)
	if err != nil {
		return RestorePlan{}, err
	}
	if err := validateRestoreManifest(manifest, config, originalID); err != nil {
		return RestorePlan{}, err
	}
	plan := RestorePlan{APIVersion: "profilemango.dev/restore-plan/v1alpha1", Kind: "RestorePlan", OriginalPlanID: originalID, Target: target, ConfigPath: config.Path, Status: "ready"}
	plan.checks = append(plan.checks, installfs.Change{Path: journalPath, Before: journalFile})
	if err := appendRestoreEntries(&plan, journal, config, manifest); err != nil {
		return RestorePlan{}, err
	}
	plan.PlanID, err = restorePlanID(plan, journalFile, config, manifest)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("hash restore plan: %w", err)
	}
	return plan, nil
}

func appendRestoreEntries(plan *RestorePlan, journal installfs.Journal, config, manifest installfs.Snapshot) error {
	for _, entry := range journal.Entries {
		current := config
		if entry.Path == manifest.Path {
			current = manifest
		}
		change, backupCheck, err := restoreChange(entry, current, plan.OriginalPlanID)
		if err != nil {
			return err
		}
		plan.changes = append(plan.changes, change)
		if backupCheck.Before.Path != "" {
			plan.checks = append(plan.checks, backupCheck)
		}
		action, afterHash := "update", installfs.Hash(change.Content)
		if change.Delete {
			action, afterHash = "delete", ""
		}
		plan.Files = append(plan.Files, RestoreFile{Path: entry.Path, Action: action, BeforeSHA256: current.SHA256, AfterSHA256: afterHash})
	}
	return nil
}

func parseRestoreJournal(snapshot installfs.Snapshot, id, config, manifest string) (installfs.Journal, error) {
	if !snapshot.Exists {
		return installfs.Journal{}, fmt.Errorf("committed install journal is missing")
	}
	var journal installfs.Journal
	decoder := json.NewDecoder(bytes.NewReader(snapshot.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return journal, fmt.Errorf("decode install journal: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return journal, fmt.Errorf("install journal contains trailing data")
	}
	if journal.APIVersion != installfs.JournalVersion || journal.PlanID != id || journal.Status != "committed" || journal.LockPath != config+".profile-mango.lock" || len(journal.Entries) != 2 {
		return journal, fmt.Errorf("install journal does not describe a committed Codex config and manifest transaction")
	}
	if journal.Entries[0].Path != config || journal.Entries[1].Path != manifest {
		return journal, fmt.Errorf("install journal contains unexpected destinations")
	}
	return journal, nil
}

func validateRestoreManifest(snapshot, config installfs.Snapshot, id string) error {
	if !snapshot.Exists || !config.Exists {
		return fmt.Errorf("installed config or ownership manifest is missing")
	}
	manifest, err := decodeManifest(snapshot.Content, Target{Name: "codex", Version: "0.154.0"})
	if err != nil {
		return fmt.Errorf("validate ownership manifest: %w", err)
	}
	if manifest.Owner != "profile-mango" || manifest.Target.String() != "codex@0.154.0" || manifest.PlanID != "" || len(manifest.Files) != 1 || manifest.Files[0].Path != config.Path || manifest.Files[0].SHA256 != config.SHA256 || len(manifest.Fields) != 0 {
		return fmt.Errorf("ownership manifest does not exclusively own the current Codex config for install %s", id)
	}
	return nil
}

func restoreChange(entry installfs.JournalEntry, current installfs.Snapshot, id string) (installfs.Change, installfs.Change, error) {
	if !entry.Applied || entry.Delete || entry.Path != current.Path || !restoreID.MatchString(entry.AfterSHA256) || !current.Exists || current.SHA256 != entry.AfterSHA256 {
		return installfs.Change{}, installfs.Change{}, fmt.Errorf("installed file %s is edited or journal entry invalid", entry.Path)
	}
	expectedMode := uint32(0o600)
	if entry.BeforeExists {
		expectedMode = entry.BeforeMode
	}
	if uint32(current.Mode) != expectedMode {
		return installfs.Change{}, installfs.Change{}, fmt.Errorf("installed file %s mode differs from recorded transaction", entry.Path)
	}
	change := installfs.Change{Path: entry.Path, Before: current, Delete: !entry.BeforeExists}
	if !entry.BeforeExists {
		if entry.BackupPath != "" || entry.BeforeSHA256 != "" || entry.BeforeMode != 0 {
			return change, installfs.Change{}, fmt.Errorf("unexpected backup for created file")
		}
		return change, installfs.Change{}, nil
	}
	if !restoreID.MatchString(entry.BeforeSHA256) || entry.BackupPath != installfs.BackupPath(entry.Path, id) || entry.BeforeMode == 0 || entry.BeforeMode&^0o777 != 0 {
		return change, installfs.Change{}, fmt.Errorf("original file lacks a valid adjacent backup")
	}
	backup, err := installfs.SnapshotFile(entry.BackupPath)
	if err != nil {
		return change, installfs.Change{}, fmt.Errorf("inspect restore backup: %w", err)
	}
	if !backup.Exists || backup.SHA256 != entry.BeforeSHA256 || uint32(backup.Mode) != entry.BeforeMode {
		return change, installfs.Change{}, fmt.Errorf("restore backup hash or mode differs from journal")
	}
	change.Content = backup.Content
	// Transaction engine preserves the current mode by default; record the original mode explicitly.
	change.Mode = backup.Mode
	return change, installfs.Change{Path: backup.Path, Before: backup}, nil
}

func restorePlanID(plan RestorePlan, journal, config, manifest installfs.Snapshot) (string, error) {
	identity := struct {
		OriginalID, Target, Config, JournalHash, ConfigHash, ManifestHash string
		Files                                                             []RestoreFile
		Modes                                                             []uint32
	}{
		plan.OriginalPlanID, plan.Target, filepath.Clean(plan.ConfigPath), journal.SHA256, config.SHA256, manifest.SHA256, plan.Files,
		[]uint32{uint32(config.Mode), uint32(manifest.Mode)},
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	return installfs.Hash(data), nil
}

func ApplyRestorePlan(plan RestorePlan, expected string) error {
	if expected == "" || expected != plan.PlanID {
		return fmt.Errorf("restore requires matching --expect-plan")
	}
	fresh, err := BuildRestorePlan(plan.Target, plan.ConfigPath, plan.OriginalPlanID)
	if err != nil {
		return fmt.Errorf("revalidate restore plan: %w", err)
	}
	if fresh.PlanID != plan.PlanID {
		return fmt.Errorf("restore source changed since planning: %w", installfs.ErrStale)
	}
	if err := installfs.Preflight(fresh.checks); err != nil {
		return fmt.Errorf("revalidate restore source: %w", err)
	}
	_, err = installfs.Apply(fresh.changes, installfs.ApplyOptions{PlanID: fresh.PlanID, Backup: true, Checks: fresh.checks})
	if err != nil {
		return fmt.Errorf("apply restore transaction: %w", err)
	}
	return nil
}
