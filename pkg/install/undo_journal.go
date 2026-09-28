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

type journalCandidate struct {
	file    installfs.Snapshot
	journal installfs.Journal
}

// selectInstallJournal loads the named install journal, or the latest committed one next to the config.
// A multi-target install may anchor the journal next to another target's file; a reference locates it.
func selectInstallJournal(config, manifest installfs.Snapshot, id string) (installfs.Snapshot, installfs.Journal, error) {
	if id == "" {
		return latestInstallJournal(config, manifest)
	}
	location, err := namedJournalLocation(config.Path, id)
	if err != nil {
		return installfs.Snapshot{}, installfs.Journal{}, err
	}
	file, journal, err := loadJournalAt(location)
	if err != nil {
		return file, journal, err
	}
	if journal.PlanID != id || journal.Status != "committed" {
		return file, journal, fmt.Errorf("install journal %s is not a committed install", id)
	}
	return file, journal, nil
}

// namedJournalLocation prefers a journal next to the config and falls back to a reference naming a shared one.
func namedJournalLocation(config, id string) (journalLocation, error) {
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
	file, journal, err := loadInstallJournal(location.path, location.anchor)
	if err == nil && location.planID != "" && journal.PlanID != location.planID {
		err = fmt.Errorf("install journal %s does not record the referenced install", filepath.Base(location.path))
	}
	return file, journal, err
}

// latestInstallJournal prefers the newest committed journal whose installed hashes match the current
// config and manifest, so successive undos step back through successive installs. Without a match it
// returns the newest committed journal, whose drift the caller then reports.
func latestInstallJournal(config, manifest installfs.Snapshot) (installfs.Snapshot, installfs.Journal, error) {
	locations, err := installJournalLocations(config.Path)
	if err != nil {
		return installfs.Snapshot{}, installfs.Journal{}, err
	}
	var matching, newest []journalCandidate
	for _, location := range locations {
		file, journal, err := loadJournalAt(location)
		if err != nil {
			return file, journal, err
		}
		if journal.Status != "committed" || entryFor(journal, manifest.Path) == nil {
			continue
		}
		candidate := journalCandidate{file: file, journal: journal}
		newest = append(newest, candidate)
		if journalMatches(journal, config, manifest) {
			matching = append(matching, candidate)
		}
	}
	for _, candidates := range [][]journalCandidate{matching, newest} {
		if len(candidates) > 0 {
			latest := newestCandidate(candidates)
			return latest.file, latest.journal, nil
		}
	}
	return installfs.Snapshot{}, installfs.Journal{}, fmt.Errorf("no committed profile-mango install journal next to %s; nothing to undo", config.Path)
}

// installJournalLocations lists journals next to the config and those named by adjacent references.
func installJournalLocations(config string) ([]journalLocation, error) {
	entries, err := os.ReadDir(filepath.Dir(config))
	if err != nil {
		return nil, fmt.Errorf("list install journals: %w", err)
	}
	journalPrefix, refPrefix := filepath.Base(config)+".profile-mango.journal.", filepath.Base(config)+".profile-mango.journal-ref."
	var locations []journalLocation
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

// loadInstallJournal strictly decodes a journal and checks that it was written for this config's transaction.
func loadInstallJournal(path, config string) (installfs.Snapshot, installfs.Journal, error) {
	file, err := installfs.SnapshotFile(path)
	if err != nil {
		return file, installfs.Journal{}, fmt.Errorf("inspect install journal: %w", err)
	}
	if !file.Exists {
		return file, installfs.Journal{}, fmt.Errorf("committed install journal is missing")
	}
	journal, err := decodeInstallJournal(file)
	if err != nil {
		return file, journal, err
	}
	if journal.APIVersion != installfs.JournalVersion || !restoreID.MatchString(journal.PlanID) || journal.LockPath != config+".profile-mango.lock" ||
		file.Path != installfs.JournalPath(config, journal.PlanID) || !uniqueEntryPaths(journal) {
		return file, journal, fmt.Errorf("install journal %s does not describe a transaction for this config", filepath.Base(path))
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

// journalRef locates a transaction journal stored next to another path, so each target config of a
// multi-target install can find the one shared journal. It is a locator only; undo validates the journal itself.
type journalRef struct {
	APIVersion  string `json:"apiVersion"`
	PlanID      string `json:"planID"`
	JournalPath string `json:"journalPath"`
}

// journalLocation names a journal, the path it is anchored next to, and the plan a reference expects it to record.
type journalLocation struct {
	path, anchor, planID string
}

func journalRefPath(config, planID string) string {
	return filepath.Clean(config) + ".profile-mango.journal-ref." + planID[:16] + ".json"
}

// writeJournalRefs records, next to each changed target config, where the committed journal lives
// when the transaction anchored it next to a different file.
func writeJournalRefs(targets []TargetPlan, planID string, applied installfs.ApplyResult) error {
	data, err := json.MarshalIndent(journalRef{APIVersion: journalRefVersion, PlanID: planID, JournalPath: applied.JournalPath}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal reference: %w", err)
	}
	for _, target := range targets {
		config, err := filepath.Abs(target.ConfigPath)
		if err != nil {
			return fmt.Errorf("resolve %s config: %w", target.Target.String(), err)
		}
		if len(target.changes) == 0 || installfs.JournalPath(config, planID) == applied.JournalPath {
			continue
		}
		if err := installfs.WriteSidecar(journalRefPath(config, planID), append(data, '\n')); err != nil {
			return fmt.Errorf("write %s journal reference: %w", target.Target.String(), err)
		}
	}
	return nil
}

// loadJournalRef strictly decodes a reference written next to config and returns the journal it names.
func loadJournalRef(path, config string) (journalLocation, error) {
	file, err := installfs.SnapshotFile(path)
	if err != nil {
		return journalLocation{}, fmt.Errorf("inspect install journal reference: %w", err)
	}
	var ref journalRef
	decoder := json.NewDecoder(bytes.NewReader(file.Content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ref); err != nil {
		return journalLocation{}, fmt.Errorf("decode install journal reference: %w", err)
	}
	if err := requireGeneratedJSON(file.Content, ref, "install journal reference"); err != nil {
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
