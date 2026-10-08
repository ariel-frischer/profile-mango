package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// The Mango state directory keeps install and undo history away from target config folders:
//
//	<state>/transactions/<plan16>-<lock16>/journal.json        install journal
//	<state>/transactions/<plan16>-<lock16>/undo-journal.json   undo journal
//	<state>/transactions/<plan16>-<lock16>/backups/<path16>-<name>
//	<state>/targets/<config16>/journals/<plan16>.json          journal reference per target config
const undoJournalName = "undo-journal.json"

// requireStateDir checks that a write was given the absolute Mango state directory.
func requireStateDir(stateDir string) error {
	if stateDir == "" || !filepath.IsAbs(stateDir) || filepath.Clean(stateDir) != stateDir {
		return fmt.Errorf("install history requires an absolute, clean Mango state directory, got %q", stateDir)
	}
	return nil
}

func historyTransactions(stateDir string) string {
	return filepath.Join(stateDir, "transactions")
}

func historyRefDir(stateDir, config string) string {
	return filepath.Join(stateDir, "targets", installfs.Hash([]byte(filepath.Clean(config)))[:16], "journals")
}

func historyRefPath(stateDir, config, planID string) string {
	return filepath.Join(historyRefDir(stateDir, config), planID[:16]+".json")
}

// historyJournalValid reports whether a state journal sits exactly where its own plan and
// lock place it, and whether that lock was taken beside one of the files it journals.
func historyJournalValid(path, transactions string, journal installfs.Journal) bool {
	anchor, found := installfs.LockAnchor(journal.LockPath)
	if !found || !filepath.IsAbs(anchor) || entryFor(journal, anchor) == nil {
		return false
	}
	return path == filepath.Join(installfs.TransactionDir(transactions, journal.PlanID, journal.LockPath), installfs.DefaultJournalName)
}

// writeJournalRefs records, in the state directory, the journal of every changed target config.
// It runs before the transaction, so a reference can name a journal that never committed.
func writeJournalRefs(targets []TargetPlan, planID, stateDir, journalPath string) error {
	data, err := json.MarshalIndent(journalRef{APIVersion: journalRefVersion, PlanID: planID, JournalPath: journalPath}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal reference: %w", err)
	}
	for _, target := range targets {
		config, err := filepath.Abs(target.ConfigPath)
		if err != nil {
			return fmt.Errorf("resolve %s config: %w", target.Target.String(), err)
		}
		if len(target.changes) == 0 {
			continue
		}
		if err := installfs.WriteHistoryFile(historyRefPath(stateDir, config, planID), append(data, '\n')); err != nil {
			return fmt.Errorf("write %s journal reference: %w", target.Target.String(), err)
		}
	}
	return nil
}

// historyJournalLocations lists the state journals referenced for config.
func historyJournalLocations(config, stateDir string) ([]journalLocation, error) {
	if stateDir == "" {
		return nil, nil
	}
	dir := historyRefDir(stateDir, config)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list install history: %w", err)
	}
	var locations []journalLocation
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		location, err := loadHistoryRef(filepath.Join(dir, entry.Name()), config, stateDir)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}
	return locations, nil
}

// loadHistoryRef strictly decodes a state reference for config and returns the state journal it names.
func loadHistoryRef(path, config, stateDir string) (journalLocation, error) {
	file, ref, err := decodeJournalRef(path)
	if err != nil {
		return journalLocation{}, err
	}
	if ref.APIVersion != journalRefVersion || !restoreID.MatchString(ref.PlanID) || file.Path != historyRefPath(stateDir, config, ref.PlanID) {
		return journalLocation{}, fmt.Errorf("install journal reference %s does not belong to this config", file.Path)
	}
	transactions := historyTransactions(stateDir)
	transaction := filepath.Dir(ref.JournalPath)
	if filepath.Dir(transaction) != transactions || filepath.Base(ref.JournalPath) != installfs.DefaultJournalName ||
		!strings.HasPrefix(filepath.Base(transaction), ref.PlanID[:16]+"-") {
		return journalLocation{}, fmt.Errorf("install journal reference %s names an invalid journal path", file.Path)
	}
	return journalLocation{path: ref.JournalPath, planID: ref.PlanID, history: transactions}, nil
}

func decodeJournalRef(path string) (installfs.Snapshot, journalRef, error) {
	var ref journalRef
	file, err := installfs.SnapshotFile(path)
	if err != nil {
		return file, ref, fmt.Errorf("inspect install journal reference: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(file.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return file, ref, fmt.Errorf("decode install journal reference: %w", err)
	}
	return file, ref, requireGeneratedJSON(file.Content, ref, "install journal reference")
}

// findBackup locates a create-only backup of path whose bytes hash to sha256, in the
// state history or beside the file where earlier versions kept it.
func findBackup(path, sha256, stateDir string) (installfs.Snapshot, error) {
	candidates, err := backupCandidates(path, stateDir)
	if err != nil {
		return installfs.Snapshot{}, err
	}
	for _, candidate := range candidates {
		snapshot, err := installfs.SnapshotFile(candidate)
		if err == nil && snapshot.Exists && snapshot.SHA256 == sha256 {
			return snapshot, nil
		}
	}
	return installfs.Snapshot{}, fmt.Errorf("no backup of %s holds its pre-install bytes; restore it manually or run mango undo", filepath.Base(path))
}

func backupCandidates(path, stateDir string) ([]string, error) {
	var candidates []string
	if stateDir != "" {
		transactions := historyTransactions(stateDir)
		entries, err := os.ReadDir(transactions)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("list install history: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				candidates = append(candidates, installfs.HistoryBackupPath(filepath.Join(transactions, entry.Name()), path))
			}
		}
	}
	legacy, err := filepath.Glob(filepath.Clean(path) + ".profile-mango.bak.*")
	if err != nil {
		return nil, fmt.Errorf("list backups of %s: %w", filepath.Base(path), err)
	}
	sort.Strings(legacy)
	return append(candidates, legacy...), nil
}
