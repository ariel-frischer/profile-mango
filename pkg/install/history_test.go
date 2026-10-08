package install

import (
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// installedTransaction returns the state history directory of the one transaction planID committed.
func installedTransaction(t *testing.T, planID string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(historyTransactions(testStateDir), planID[:16]+"-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("transactions for %s = %v, err = %v", planID[:16], matches, err)
	}
	return matches[0]
}

// installedBackup is where the transaction planID kept its backup of path.
func installedBackup(t *testing.T, planID, path string) string {
	t.Helper()
	return installfs.HistoryBackupPath(installedTransaction(t, planID), path)
}

// plannedBackup is where applying planID, anchored on anchor, will back up path.
func plannedBackup(planID, anchor, path string) string {
	transaction := installfs.TransactionDir(historyTransactions(testStateDir), planID, installfs.LockPath(anchor))
	return installfs.HistoryBackupPath(transaction, path)
}
