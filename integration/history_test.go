package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// workflowStateDir is the Mango state directory cleanInstallEnv gives the installed binary.
func workflowStateDir(root string) string {
	return filepath.Join(root, "state")
}

// workflowTransaction is the state history directory of planID, whose lock sits beside anchor.
func workflowTransaction(root, planID, anchor string) string {
	return installfs.TransactionDir(filepath.Join(workflowStateDir(root), "transactions"), planID, installfs.LockPath(anchor))
}

// workflowBackup returns the state backup of path that the committed transaction planID kept.
func workflowBackup(t *testing.T, root, planID, path string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(workflowStateDir(root), "transactions", planID[:16]+"-*"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("transactions for %s = %v, err = %v", planID[:16], matches, err)
	}
	return installfs.HistoryBackupPath(matches[0], path)
}
