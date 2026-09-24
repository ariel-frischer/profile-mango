package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

var restoreID = regexp.MustCompile(`^[a-f0-9]{64}$`)

const manifestSuffix = ".profile-mango.manifest.json"

// UndoRequest selects one installed target config to return to its pre-install state.
type UndoRequest struct {
	Target     Target
	ConfigPath string
	// OriginalPlanID names the install to reverse; empty selects the latest committed install journal.
	OriginalPlanID string
	// Override discards edits made to target files after the install; the ownership manifest must still be unchanged.
	Override bool
	Registry *Registry
}

// RestorePlan is an inert, content-bound preview of one installation reversal.
type RestorePlan struct {
	APIVersion     string        `json:"apiVersion"`
	Kind           string        `json:"kind"`
	PlanID         string        `json:"planID"`
	OriginalPlanID string        `json:"originalPlanID"`
	Target         string        `json:"target"`
	ConfigPath     string        `json:"configPath"`
	Status         string        `json:"status"`
	Override       bool          `json:"override,omitempty"`
	Files          []RestoreFile `json:"files"`
	request        UndoRequest
	changes        []installfs.Change
	checks         []installfs.Change
}

type RestoreFile struct {
	Path         string `json:"path"`
	Action       string `json:"action"`
	BeforeSHA256 string `json:"beforeSHA256"`
	AfterSHA256  string `json:"afterSHA256,omitempty"`
	Drifted      bool   `json:"drifted,omitempty"`
	// Diff is a redacted human preview of the file effect; it never enters JSON or the plan ID.
	Diff string `json:"-"`
}

// DriftError reports target files edited after the install being undone.
type DriftError struct {
	Files []RestoreFile
}

func (err *DriftError) Error() string {
	var builder strings.Builder
	builder.WriteString("config changed after the install being undone; review the diff, then rerun with --override to discard these edits")
	for _, file := range err.Files {
		builder.WriteString("\n")
		builder.WriteString(file.Diff)
	}
	return builder.String()
}

// BuildRestorePlan keeps the original restore signature: an exact target@version and an optional install plan ID.
func BuildRestorePlan(target, configPath, originalID string) (RestorePlan, error) {
	parsed, err := ParseTarget(target)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("parse restore target: %w", err)
	}
	return BuildUndoPlan(UndoRequest{Target: parsed, ConfigPath: configPath, OriginalPlanID: originalID})
}

type undoState struct {
	config, manifest, journalFile installfs.Snapshot
	ownership                     Manifest
	journal                       installfs.Journal
}

// BuildUndoPlan previews reversing one committed install of a registered, installable target.
func BuildUndoPlan(request UndoRequest) (RestorePlan, error) {
	if err := validateUndoTarget(request); err != nil {
		return RestorePlan{}, err
	}
	state, err := loadUndoState(request)
	if err != nil {
		return RestorePlan{}, err
	}
	request.OriginalPlanID = state.journal.PlanID
	plan := RestorePlan{APIVersion: "profilemango.dev/restore-plan/v1alpha1", Kind: "RestorePlan", OriginalPlanID: state.journal.PlanID,
		Target: request.Target.String(), ConfigPath: state.config.Path, Status: "ready", Override: request.Override, request: request}
	plan.checks = append(plan.checks, installfs.Change{Path: state.journalFile.Path, Before: state.journalFile})
	if err := appendRestoreEntries(&plan, state); err != nil {
		return RestorePlan{}, err
	}
	if drifted := driftedFiles(plan.Files); len(drifted) > 0 && !request.Override {
		return RestorePlan{}, &DriftError{Files: drifted}
	}
	plan.PlanID, err = restorePlanID(plan, state.journalFile)
	if err != nil {
		return RestorePlan{}, fmt.Errorf("hash restore plan: %w", err)
	}
	return plan, nil
}

func validateUndoTarget(request UndoRequest) error {
	registry := request.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	adapter, found := registry.Lookup(request.Target)
	if !found || !adapter.Metadata().Installable {
		return fmt.Errorf("undo requires a registered installable target@version, got %q", request.Target.String())
	}
	if request.OriginalPlanID != "" && !restoreID.MatchString(request.OriginalPlanID) {
		return fmt.Errorf("original install plan ID must be 64 lowercase hex characters")
	}
	return nil
}

func loadUndoState(request UndoRequest) (undoState, error) {
	var state undoState
	var err error
	if state.config, err = installfs.SnapshotFile(request.ConfigPath); err != nil {
		return state, fmt.Errorf("inspect undo config: %w", err)
	}
	if state.manifest, err = installfs.SnapshotFile(state.config.Path + manifestSuffix); err != nil {
		return state, fmt.Errorf("inspect ownership manifest: %w", err)
	}
	if state.ownership, err = validateRestoreManifest(state.manifest, request.Target); err != nil {
		return state, err
	}
	state.journalFile, state.journal, err = selectInstallJournal(state.config, state.manifest, request.OriginalPlanID)
	if err != nil {
		return state, err
	}
	state.journal.Entries, err = targetJournalEntries(state.journal, state.manifest.Path, state.ownership)
	if err != nil {
		return state, err
	}
	return state, requireChangedFilesJournaled(state)
}

// requireChangedFilesJournaled checks that the journal covers every file whose owned hash the
// install changed, so a journal that dropped or renamed an entry cannot undo only part of an install.
func requireChangedFilesJournaled(state undoState) error {
	manifestEntry := entryFor(state.journal, state.manifest.Path)
	var before Manifest
	if manifestEntry.BeforeExists {
		backup, err := installfs.SnapshotFile(manifestEntry.BackupPath)
		if err != nil || !backup.Exists {
			return fmt.Errorf("original ownership manifest backup is missing or unreadable")
		}
		if before, err = decodeManifest(backup.Content, state.ownership.Target); err != nil {
			return fmt.Errorf("decode original ownership manifest: %w", err)
		}
	}
	for _, file := range state.ownership.Files {
		priorHash, _ := ownershipHash(before, file.Path)
		if priorHash != file.SHA256 && entryFor(state.journal, file.Path) == nil {
			return fmt.Errorf("install journal %s does not describe every file the install changed", state.journal.PlanID)
		}
	}
	return nil
}

func validateRestoreManifest(snapshot installfs.Snapshot, target Target) (Manifest, error) {
	if !snapshot.Exists {
		return Manifest{}, fmt.Errorf("no profile-mango ownership manifest at %s; nothing to undo", snapshot.Path)
	}
	manifest, err := decodeManifest(snapshot.Content, target)
	if err != nil {
		return Manifest{}, fmt.Errorf("validate ownership manifest: %w", err)
	}
	if err := requireGeneratedJSON(snapshot.Content, manifest, "ownership manifest"); err != nil {
		return Manifest{}, err
	}
	if manifest.Owner != "profile-mango" || manifest.Target != target || manifest.PlanID != "" {
		return Manifest{}, fmt.Errorf("ownership manifest does not belong to a profile-mango %s install", target.String())
	}
	return manifest, nil
}

// targetJournalEntries keeps the entries this target owns: its manifest and manifest-listed files,
// which include the config unless a named-profile install left it unchanged.
// A multi-target transaction may also journal other targets' files, which this undo leaves alone.
func targetJournalEntries(journal installfs.Journal, manifestPath string, ownership Manifest) ([]installfs.JournalEntry, error) {
	var result []installfs.JournalEntry
	hasOwned, hasManifest := false, false
	for _, entry := range journal.Entries {
		ownedHash, owned := ownershipHash(ownership, entry.Path)
		switch {
		case entry.Path == manifestPath:
			hasManifest = true
		case owned:
			hasOwned = true
			if ownedHash != entry.AfterSHA256 {
				return nil, fmt.Errorf("ownership manifest does not record the installed content of %s", entry.Path)
			}
		default:
			continue
		}
		result = append(result, entry)
	}
	if !hasOwned || !hasManifest {
		return nil, fmt.Errorf("install journal %s does not describe this config and its ownership manifest", journal.PlanID)
	}
	return result, nil
}

func appendRestoreEntries(plan *RestorePlan, state undoState) error {
	for _, entry := range state.journal.Entries {
		current, err := currentSnapshot(entry.Path, state)
		if err != nil {
			return err
		}
		if err := planRestoreEntry(plan, entry, current, entry.Path == state.manifest.Path); err != nil {
			return err
		}
	}
	return nil
}

func currentSnapshot(path string, state undoState) (installfs.Snapshot, error) {
	switch path {
	case state.config.Path:
		return state.config, nil
	case state.manifest.Path:
		return state.manifest, nil
	}
	snapshot, err := installfs.SnapshotFile(path)
	if err != nil {
		return snapshot, fmt.Errorf("inspect installed file: %w", err)
	}
	return snapshot, nil
}

func planRestoreEntry(plan *RestorePlan, entry installfs.JournalEntry, current installfs.Snapshot, isManifest bool) error {
	drifted, err := installedDrift(entry, current, isManifest)
	if err != nil {
		return err
	}
	original, err := originalContent(entry, plan.OriginalPlanID, plan.request.Target, isManifest)
	if err != nil {
		return err
	}
	file := RestoreFile{Path: entry.Path, BeforeSHA256: current.SHA256, Drifted: drifted}
	change := installfs.Change{Path: entry.Path, Before: current}
	switch {
	case !entry.BeforeExists && !current.Exists:
		file.Action = ActionNoop
	case !entry.BeforeExists:
		file.Action, change.Delete = ActionDelete, true
	default:
		file.Action, file.AfterSHA256 = ActionUpdate, original.SHA256
		// The transaction engine preserves the current mode by default; restore the original mode explicitly.
		change.Content, change.Mode = original.Content, original.Mode
	}
	if !isManifest {
		file.Diff = unifiedDiff(entry.Path, current.Content, change.Content)
	}
	if file.Action != ActionNoop {
		plan.changes = append(plan.changes, change)
	}
	if original.Exists {
		plan.checks = append(plan.checks, installfs.Change{Path: original.Path, Before: original})
	}
	plan.Files = append(plan.Files, file)
	return nil
}

// installedDrift reports whether a file differs from what the install wrote; the manifest must never drift.
func installedDrift(entry installfs.JournalEntry, current installfs.Snapshot, isManifest bool) (bool, error) {
	if !entry.Applied || entry.Delete || !restoreID.MatchString(entry.AfterSHA256) {
		return false, fmt.Errorf("install journal entry for %q is invalid or cannot be undone", entry.Path)
	}
	expectedMode := uint32(0o600)
	if entry.BeforeExists {
		expectedMode = entry.BeforeMode
	}
	unchanged := current.Exists && current.SHA256 == entry.AfterSHA256 && uint32(current.Mode) == expectedMode
	if !unchanged && isManifest {
		return false, fmt.Errorf("ownership manifest %q changed after install; undo will not touch it", entry.Path)
	}
	return !unchanged, nil
}

// originalContent returns the verified pre-install backup, or an absent snapshot when the install created the file.
func originalContent(entry installfs.JournalEntry, id string, target Target, isManifest bool) (installfs.Snapshot, error) {
	if !entry.BeforeExists {
		if entry.BackupPath != "" || entry.BeforeSHA256 != "" || entry.BeforeMode != 0 {
			return installfs.Snapshot{}, fmt.Errorf("unexpected backup for created file")
		}
		return installfs.Snapshot{}, nil
	}
	if !restoreID.MatchString(entry.BeforeSHA256) || entry.BackupPath != installfs.BackupPath(entry.Path, id) || entry.BeforeMode == 0 || entry.BeforeMode&^0o777 != 0 {
		return installfs.Snapshot{}, fmt.Errorf("original %s lacks a valid adjacent backup (was it installed with --no-backup?)", entry.Path)
	}
	backup, err := installfs.SnapshotFile(entry.BackupPath)
	if err != nil {
		return backup, fmt.Errorf("inspect restore backup: %w", err)
	}
	if !backup.Exists || backup.SHA256 != entry.BeforeSHA256 || uint32(backup.Mode) != entry.BeforeMode {
		return backup, fmt.Errorf("restore backup hash or mode differs from journal")
	}
	if isManifest {
		return backup, validateOriginalManifest(backup, target)
	}
	return backup, nil
}

func validateOriginalManifest(backup installfs.Snapshot, target Target) error {
	original, err := decodeManifest(backup.Content, target)
	if err != nil {
		return fmt.Errorf("decode original ownership manifest: %w", err)
	}
	if original.Owner != "profile-mango" {
		return fmt.Errorf("original ownership manifest has another owner")
	}
	return requireGeneratedJSON(backup.Content, original, "original ownership manifest")
}

func driftedFiles(files []RestoreFile) []RestoreFile {
	var result []RestoreFile
	for _, file := range files {
		if file.Drifted {
			result = append(result, file)
		}
	}
	return result
}

func requireGeneratedJSON(data []byte, value any, label string) error {
	canonical, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", label, err)
	}
	if !bytes.Equal(data, append(canonical, '\n')) {
		return fmt.Errorf("%s is not a canonical installer-generated JSON artifact", label)
	}
	return nil
}

func decodeInstallJournal(snapshot installfs.Snapshot) (installfs.Journal, error) {
	var journal installfs.Journal
	decoder := json.NewDecoder(bytes.NewReader(snapshot.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return journal, fmt.Errorf("decode install journal: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return journal, fmt.Errorf("install journal contains trailing data")
	}
	if err := requireGeneratedJSON(snapshot.Content, journal, "install journal"); err != nil {
		return journal, err
	}
	return journal, nil
}

func restorePlanID(plan RestorePlan, journal installfs.Snapshot) (string, error) {
	identity := struct {
		OriginalID, Target, Config, JournalHash string
		Override                                bool
		Files                                   []RestoreFile
		Sources                                 []installfs.Identity
	}{
		plan.OriginalPlanID, plan.Target, filepath.Clean(plan.ConfigPath), journal.SHA256, plan.Override, plan.Files, restoreSourceIdentities(plan),
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	return installfs.Hash(data), nil
}

func restoreSourceIdentities(plan RestorePlan) []installfs.Identity {
	identities := make([]installfs.Identity, 0, len(plan.changes)+len(plan.checks))
	for _, change := range plan.changes {
		identities = append(identities, change.Before.Identity)
	}
	for _, check := range plan.checks {
		identities = append(identities, check.Before.Identity)
	}
	return identities
}

// undoJournalPath keeps undo journals apart from install journals so undo never selects its own transaction.
func undoJournalPath(config, planID string) string {
	return filepath.Clean(config) + ".profile-mango.undo-journal." + planID[:16] + ".json"
}

func ApplyRestorePlan(plan RestorePlan, expected string) error {
	if expected == "" || expected != plan.PlanID {
		return fmt.Errorf("restore requires matching --expect-plan")
	}
	fresh, err := BuildUndoPlan(plan.request)
	if err != nil {
		return fmt.Errorf("revalidate restore plan: %w", err)
	}
	if fresh.PlanID != plan.PlanID {
		return fmt.Errorf("restore source changed since planning: %w", installfs.ErrStale)
	}
	if err := installfs.Preflight(fresh.checks); err != nil {
		return fmt.Errorf("revalidate restore source: %w", err)
	}
	options := installfs.ApplyOptions{PlanID: fresh.PlanID, Backup: true, Checks: fresh.checks, JournalPath: undoJournalPath(fresh.ConfigPath, fresh.PlanID)}
	if _, err := installfs.Apply(fresh.changes, options); err != nil {
		return fmt.Errorf("apply restore transaction: %w", err)
	}
	return nil
}
