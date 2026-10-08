package installfs

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	// DefaultJournalName is the journal file inside a transaction history directory.
	DefaultJournalName = "journal.json"
	historyBackupDir   = "backups"
	lockSuffix         = ".profile-mango.lock"
)

// TransactionDir is the private directory under historyDir that holds one transaction's
// journal and backups. The lock path, which sits beside the transaction's anchor file,
// keeps equal plans applied to different config homes apart.
func TransactionDir(historyDir, planID, lockPath string) string {
	return filepath.Join(filepath.Clean(historyDir), shortID(planID)+"-"+Hash([]byte(filepath.Clean(lockPath)))[:16])
}

// HistoryBackupPath is where the transaction kept in transactionDir backs up path.
func HistoryBackupPath(transactionDir, path string) string {
	clean := filepath.Clean(path)
	return filepath.Join(transactionDir, historyBackupDir, Hash([]byte(clean))[:16]+"-"+filepath.Base(clean))
}

// LockPath is the transient installer lock beside a transaction's anchor file.
func LockPath(anchor string) string {
	return filepath.Clean(anchor) + lockSuffix
}

// LockAnchor returns the anchor file a lock path was derived from.
func LockAnchor(lockPath string) (string, bool) {
	return strings.CutSuffix(lockPath, lockSuffix)
}

// WriteHistoryFile atomically writes a small installer-owned 0600 file outside a
// transaction, creating its missing private parent directories without following symlinks.
func WriteHistoryFile(path string, data []byte) error {
	if err := ensureParentDirs(path); err != nil {
		return err
	}
	return writeAtomicUnconditional(path, data, 0o600)
}

// resolveTransactionPaths fills in the lock and journal paths, and, for a history
// transaction, the private directory that holds its journal and backups.
func resolveTransactionPaths(changes []Change, options *ApplyOptions) error {
	anchor, err := journalAnchor(changes, options.Anchors)
	if err != nil {
		return err
	}
	if options.LockPath == "" {
		options.LockPath = LockPath(anchor)
	}
	if options.HistoryDir == "" {
		if options.JournalPath == "" {
			options.JournalPath = JournalPath(anchor, options.PlanID)
		}
		return nil
	}
	if !filepath.IsAbs(options.HistoryDir) {
		return fmt.Errorf("install history directory must be absolute: %s", options.HistoryDir)
	}
	options.transactionDir = TransactionDir(options.HistoryDir, options.PlanID, options.LockPath)
	if options.JournalPath == "" {
		name := options.JournalName
		if name == "" {
			name = DefaultJournalName
		}
		options.JournalPath = filepath.Join(options.transactionDir, name)
	}
	return nil
}

// prepareTransactionDir creates a history transaction's private backup directory, and so
// its journal directory, once the lock is held and the transaction has been preflighted.
func prepareTransactionDir(options ApplyOptions) error {
	if options.transactionDir == "" {
		return nil
	}
	if err := ensureDirs(filepath.Join(options.transactionDir, historyBackupDir)); err != nil {
		return fmt.Errorf("prepare install history: %w", err)
	}
	return nil
}
