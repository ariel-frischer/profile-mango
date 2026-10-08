package main

import (
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// testStateDir is the sandbox Mango state directory TestMain points every command at.
func testStateDir(t *testing.T) string {
	t.Helper()
	dir, err := selectedStateDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// historyTransaction is the state directory of the single-target install planID, anchored on config.
func historyTransaction(t *testing.T, config, planID string) string {
	t.Helper()
	return installfs.TransactionDir(filepath.Join(testStateDir(t), "transactions"), planID, installfs.LockPath(config))
}

// historyJournal is the state journal of the single-target install planID on config.
func historyJournal(t *testing.T, config, planID string) string {
	t.Helper()
	return filepath.Join(historyTransaction(t, config, planID), installfs.DefaultJournalName)
}

// historyBackup is the state backup of config kept by the single-target install planID.
func historyBackup(t *testing.T, config, planID string) string {
	t.Helper()
	return installfs.HistoryBackupPath(historyTransaction(t, config, planID), config)
}
