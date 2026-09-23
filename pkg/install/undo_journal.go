package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

type journalCandidate struct {
	file    installfs.Snapshot
	journal installfs.Journal
}

// selectInstallJournal loads the named install journal, or the latest committed one next to the config.
func selectInstallJournal(config, manifest installfs.Snapshot, id string) (installfs.Snapshot, installfs.Journal, error) {
	if id == "" {
		return latestInstallJournal(config, manifest)
	}
	file, journal, err := loadInstallJournal(installfs.JournalPath(config.Path, id), config.Path)
	if err != nil {
		return file, journal, err
	}
	if journal.PlanID != id || journal.Status != "committed" {
		return file, journal, fmt.Errorf("install journal %s is not a committed install", id)
	}
	return file, journal, nil
}

// latestInstallJournal prefers the newest committed journal whose installed hashes match the current
// config and manifest, so successive undos step back through successive installs. Without a match it
// returns the newest committed journal, whose drift the caller then reports.
func latestInstallJournal(config, manifest installfs.Snapshot) (installfs.Snapshot, installfs.Journal, error) {
	paths, err := installJournalPaths(config.Path)
	if err != nil {
		return installfs.Snapshot{}, installfs.Journal{}, err
	}
	var matching, newest []journalCandidate
	for _, path := range paths {
		file, journal, err := loadInstallJournal(path, config.Path)
		if err != nil {
			return file, journal, err
		}
		if journal.Status != "committed" || entryFor(journal, config.Path) == nil || entryFor(journal, manifest.Path) == nil {
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

func installJournalPaths(config string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Dir(config))
	if err != nil {
		return nil, fmt.Errorf("list install journals: %w", err)
	}
	prefix := filepath.Base(config) + ".profile-mango.journal."
	var paths []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".json") {
			paths = append(paths, filepath.Join(filepath.Dir(config), entry.Name()))
		}
	}
	return paths, nil
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

func journalMatches(journal installfs.Journal, config, manifest installfs.Snapshot) bool {
	return config.Exists && entryFor(journal, config.Path).AfterSHA256 == config.SHA256 &&
		manifest.Exists && entryFor(journal, manifest.Path).AfterSHA256 == manifest.SHA256
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
