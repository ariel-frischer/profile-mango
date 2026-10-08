package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// errJournalMissing reports a referenced or named install journal that does not exist.
var errJournalMissing = errors.New("install journal is missing")

type journalCandidate struct {
	file     installfs.Snapshot
	journal  installfs.Journal
	location journalLocation
}

// selectInstallJournal loads the named install journal, or the latest committed one for the config,
// from the state history or, for installs made by earlier versions, beside the config.
func selectInstallJournal(config, manifest installfs.Snapshot, id, stateDir string) (journalCandidate, error) {
	if id == "" {
		return latestInstallJournal(config, manifest, stateDir)
	}
	location, err := namedJournalLocation(config.Path, id, stateDir)
	if err != nil {
		return journalCandidate{}, err
	}
	file, journal, err := loadJournalAt(location)
	if err != nil {
		return journalCandidate{}, err
	}
	if journal.PlanID != id || journal.Status != "committed" {
		return journalCandidate{}, fmt.Errorf("install journal %s is not a committed install", id)
	}
	return journalCandidate{file: file, journal: journal, location: location}, nil
}

// namedJournalLocation prefers a state reference, then a legacy journal next to the config,
// then a legacy reference naming a shared one.
func namedJournalLocation(config, id, stateDir string) (journalLocation, error) {
	if stateDir != "" {
		path := historyRefPath(stateDir, config, id)
		file, err := installfs.SnapshotFile(path)
		if err != nil {
			return journalLocation{}, fmt.Errorf("inspect install journal reference: %w", err)
		}
		if file.Exists {
			return loadHistoryRef(path, config, stateDir)
		}
	}
	local := journalLocation{path: installfs.JournalPath(config, id), anchor: config}
	for _, path := range []string{local.path, journalRefPath(config, id)} {
		file, err := installfs.SnapshotFile(path)
		if err != nil {
			return local, fmt.Errorf("inspect install journal: %w", err)
		}
		if file.Exists && path != local.path {
			return loadJournalRef(path, config)
		}
		if file.Exists {
			return local, nil
		}
	}
	return local, nil
}

func loadJournalAt(location journalLocation) (installfs.Snapshot, installfs.Journal, error) {
	file, journal, err := loadInstallJournal(location)
	if err == nil && location.planID != "" && journal.PlanID != location.planID {
		err = fmt.Errorf("install journal %s does not record the referenced install", filepath.Base(location.path))
	}
	return file, journal, err
}

// latestInstallJournal prefers the newest committed journal whose installed hashes match the current
// config and manifest, so successive undos step back through successive installs. Without a match it
// returns the newest committed journal, whose drift the caller then reports.
func latestInstallJournal(config, manifest installfs.Snapshot, stateDir string) (journalCandidate, error) {
	locations, err := installJournalLocations(config.Path, stateDir)
	if err != nil {
		return journalCandidate{}, err
	}
	var matching, newest []journalCandidate
	for _, location := range locations {
		file, journal, err := loadJournalAt(location)
		if location.history != "" && errors.Is(err, errJournalMissing) {
			// The reference was written before a transaction that never committed.
			continue
		}
		if err != nil {
			return journalCandidate{}, err
		}
		if journal.Status != "committed" || entryFor(journal, manifest.Path) == nil {
			continue
		}
		candidate := journalCandidate{file: file, journal: journal, location: location}
		newest = append(newest, candidate)
		if journalMatches(journal, config, manifest) {
			matching = append(matching, candidate)
		}
	}
	for _, candidates := range [][]journalCandidate{matching, newest} {
		if len(candidates) > 0 {
			return newestCandidate(candidates), nil
		}
	}
	return journalCandidate{}, fmt.Errorf("no committed profile-mango install journal for %s; nothing to undo", config.Path)
}

// installJournalLocations lists the state journals referenced for the config, then legacy
// journals next to the config and those named by legacy adjacent references.
func installJournalLocations(config, stateDir string) ([]journalLocation, error) {
	locations, err := historyJournalLocations(config, stateDir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(config))
	if err != nil {
		return nil, fmt.Errorf("list install journals: %w", err)
	}
	journalPrefix, refPrefix := filepath.Base(config)+".profile-mango.journal.", filepath.Base(config)+".profile-mango.journal-ref."
	for _, entry := range entries {
		path := filepath.Join(filepath.Dir(config), entry.Name())
		switch {
		case !strings.HasSuffix(entry.Name(), ".json"):
		case strings.HasPrefix(entry.Name(), journalPrefix):
			locations = append(locations, journalLocation{path: path, anchor: config})
		case strings.HasPrefix(entry.Name(), refPrefix):
			location, err := loadJournalRef(path, config)
			if err != nil {
				return nil, err
			}
			locations = append(locations, location)
		}
	}
	return locations, nil
}

// loadInstallJournal strictly decodes a journal and checks that it was written where its
// location requires: in its own state transaction directory, or beside the legacy anchor.
func loadInstallJournal(location journalLocation) (installfs.Snapshot, installfs.Journal, error) {
	file, err := installfs.SnapshotFile(location.path)
	if err != nil {
		return file, installfs.Journal{}, fmt.Errorf("inspect install journal: %w", err)
	}
	if !file.Exists {
		return file, installfs.Journal{}, fmt.Errorf("%w: %s; that install never committed, or its history was removed", errJournalMissing, location.path)
	}
	journal, err := decodeInstallJournal(file)
	if err != nil {
		return file, journal, err
	}
	placed := journal.LockPath == installfs.LockPath(location.anchor) && file.Path == installfs.JournalPath(location.anchor, journal.PlanID)
	if location.history != "" {
		placed = historyJournalValid(file.Path, location.history, journal)
	}
	if journal.APIVersion != installfs.JournalVersion || !restoreID.MatchString(journal.PlanID) || !placed || !uniqueEntryPaths(journal) {
		return file, journal, fmt.Errorf("install journal %s does not describe a transaction for this config", filepath.Base(location.path))
	}
	return file, journal, nil
}

func uniqueEntryPaths(journal installfs.Journal) bool {
	seen := make(map[string]struct{}, len(journal.Entries))
	for _, entry := range journal.Entries {
		if _, found := seen[entry.Path]; found {
			return false
		}
		seen[entry.Path] = struct{}{}
	}
	return true
}

func entryFor(journal installfs.Journal, path string) *installfs.JournalEntry {
	for index := range journal.Entries {
		if journal.Entries[index].Path == path {
			return &journal.Entries[index]
		}
	}
	return nil
}

// journalMatches compares the installed hashes; a named-profile install may leave the main config out of the journal.
func journalMatches(journal installfs.Journal, config, manifest installfs.Snapshot) bool {
	configEntry := entryFor(journal, config.Path)
	configMatches := configEntry == nil || (config.Exists && configEntry.AfterSHA256 == config.SHA256)
	return configMatches && manifest.Exists && entryFor(journal, manifest.Path).AfterSHA256 == manifest.SHA256
}

func newestCandidate(candidates []journalCandidate) journalCandidate {
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i].file, candidates[j].file
		if left.Identity.ModTime != right.Identity.ModTime {
			return left.Identity.ModTime < right.Identity.ModTime
		}
		return left.Path < right.Path
	})
	return candidates[len(candidates)-1]
}

const journalRefVersion = "profilemango.dev/install-journal-ref/v1alpha1"

// journalRef locates a shared transaction journal, so each target config of a multi-target
// install can find it. It is a locator only; undo validates the journal itself.
type journalRef struct {
	APIVersion  string `json:"apiVersion"`
	PlanID      string `json:"planID"`
	JournalPath string `json:"journalPath"`
}

// journalLocation names a journal and the plan a reference expects it to record. A legacy
// journal is anchored next to anchor; a state journal lives under the history transactions dir.
type journalLocation struct {
	path, anchor, planID, history string
}

// journalRefPath is where earlier versions referenced a shared journal, next to the config.
func journalRefPath(config, planID string) string {
	return filepath.Clean(config) + ".profile-mango.journal-ref." + planID[:16] + ".json"
}

// loadJournalRef strictly decodes a legacy reference written next to config and returns the journal it names.
func loadJournalRef(path, config string) (journalLocation, error) {
	file, ref, err := decodeJournalRef(path)
	if err != nil {
		return journalLocation{}, err
	}
	if ref.APIVersion != journalRefVersion || !restoreID.MatchString(ref.PlanID) || file.Path != journalRefPath(config, ref.PlanID) {
		return journalLocation{}, fmt.Errorf("install journal reference %s does not belong to this config", filepath.Base(path))
	}
	anchor := strings.TrimSuffix(ref.JournalPath, ".profile-mango.journal."+ref.PlanID[:16]+".json")
	if !filepath.IsAbs(anchor) || installfs.JournalPath(anchor, ref.PlanID) != ref.JournalPath {
		return journalLocation{}, fmt.Errorf("install journal reference %s names an invalid journal path", filepath.Base(path))
	}
	return journalLocation{path: ref.JournalPath, anchor: anchor, planID: ref.PlanID}, nil
}
