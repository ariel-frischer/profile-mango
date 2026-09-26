package installfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// missingParentDirs lists, deepest first, the parent directories of path that do not exist
// yet and that creating path will therefore create.
func missingParentDirs(path string) []string {
	var missing []string
	for dir := filepath.Dir(path); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			break
		}
		missing = append(missing, dir)
	}
	return missing
}

// backupPathFor keeps a deleted file's backup out of the directories its delete removes,
// next to the journal instead, so those directories can become empty.
func backupPathFor(change Change, options ApplyOptions, index int) string {
	if change.Delete && len(change.RemoveEmptyDirs) > 0 {
		return strings.TrimSuffix(options.JournalPath, ".json") + ".bak." + strconv.Itoa(index)
	}
	return BackupPath(change.Path, options.PlanID)
}

// removeCreatedFile removes a file a transaction created and then its created directories.
func removeCreatedFile(path string, dirs []string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return removeCreatedDirs(path, dirs)
}

// removeCreatedDirs removes, deepest first, the listed parent directories of path while
// they are empty. It stops at the first one that is not an empty real directory, so a
// directory that has since gained content, or been replaced, is kept.
func removeCreatedDirs(path string, dirs []string) error {
	for _, dir := range dirs {
		if !strings.HasPrefix(path, filepath.Clean(dir)+string(os.PathSeparator)) || !filepath.IsAbs(dir) {
			return fmt.Errorf("created directory %s is not a parent of %s", dir, path)
		}
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect created directory %s: %w", dir, err)
		}
		if !info.IsDir() {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
				return nil
			}
			return fmt.Errorf("remove created directory %s: %w", dir, err)
		}
	}
	return nil
}

// restoreDeleted rewrites a deleted file, recreating parent directories the delete removed.
func restoreDeleted(path string, content []byte, current Snapshot, mode fs.FileMode) error {
	if err := ensureParentDirs(path); err != nil {
		return err
	}
	return atomicReplace(path, content, current, mode)
}
