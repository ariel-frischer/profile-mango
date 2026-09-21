package installfs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	MaxFileBytes   = int64(8 << 20)
	JournalVersion = "profilemango.dev/install-journal/v1alpha1"
)

var (
	ErrStale            = errors.New("file changed after planning")
	ErrRecoveryRequired = errors.New("transaction recovery is required")
	ErrLockHeld         = errors.New("installer lock is already held")
)

// Identity distinguishes an unchanged file from an ABA replacement.
type Identity struct {
	Device  uint64 `json:"device"`
	Inode   uint64 `json:"inode"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	Mode    uint32 `json:"mode"`
}

// Snapshot is a bounded, content-addressed observation of one regular file.
type Snapshot struct {
	Path     string      `json:"path"`
	Exists   bool        `json:"exists"`
	SHA256   string      `json:"sha256,omitempty"`
	Size     int64       `json:"size,omitempty"`
	Mode     fs.FileMode `json:"mode,omitempty"`
	Identity Identity    `json:"identity,omitempty"`
	Content  []byte      `json:"-"`
}

// Change is one independently replaceable file effect.
type Change struct {
	Path    string
	Before  Snapshot
	Content []byte
	Delete  bool
}

type ApplyOptions struct {
	PlanID       string
	Backup       bool
	JournalPath  string
	LockPath     string
	FaultAfter   int
	LeaveJournal bool
}

type ApplyResult struct {
	Status      string
	JournalPath string
	Backups     []string
	Changed     []string
}

type Journal struct {
	APIVersion string         `json:"apiVersion"`
	PlanID     string         `json:"planID"`
	Status     string         `json:"status"`
	LockPath   string         `json:"lockPath"`
	Entries    []JournalEntry `json:"entries"`
}

type JournalEntry struct {
	Path         string `json:"path"`
	BeforeExists bool   `json:"beforeExists"`
	BeforeSHA256 string `json:"beforeSHA256,omitempty"`
	BeforeMode   uint32 `json:"beforeMode,omitempty"`
	AfterSHA256  string `json:"afterSHA256"`
	BackupPath   string `json:"backupPath,omitempty"`
	Delete       bool   `json:"delete,omitempty"`
	Applied      bool   `json:"applied"`
}

func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func ValidateContent(data []byte) error {
	if int64(len(data)) > MaxFileBytes {
		return fmt.Errorf("file exceeds %d-byte limit", MaxFileBytes)
	}
	return nil
}

func SnapshotFile(path string) (Snapshot, error) {
	abs, info, err := inspectPath(path, true)
	if err != nil {
		return Snapshot{}, err
	}
	if info == nil {
		return Snapshot{Path: abs}, nil
	}
	if err := validateRegular(abs, info); err != nil {
		return Snapshot{}, err
	}
	data, openedInfo, err := readRegular(abs, info)
	if err != nil {
		return Snapshot{}, err
	}
	if !sameIdentity(info, openedInfo) {
		return Snapshot{}, fmt.Errorf("file changed while reading %s: %w", abs, ErrStale)
	}
	return Snapshot{
		Path: abs, Exists: true, SHA256: Hash(data), Size: int64(len(data)),
		Mode: info.Mode().Perm(), Identity: fileIdentity(info), Content: data,
	}, nil
}

func (snapshot Snapshot) Equal(other Snapshot) bool {
	if snapshot.Exists != other.Exists {
		return false
	}
	if !snapshot.Exists {
		return true
	}
	return snapshot.SHA256 == other.SHA256 && snapshot.Identity == other.Identity
}

func BackupPath(path, planID string) string {
	return filepath.Clean(path) + ".profile-mango.bak." + shortID(planID)
}

func JournalPath(path, planID string) string {
	return filepath.Clean(path) + ".profile-mango.journal." + shortID(planID) + ".json"
}

func Apply(changes []Change, options ApplyOptions) (ApplyResult, error) {
	changes, err := normalizeChanges(changes)
	if err != nil {
		return ApplyResult{}, err
	}
	if len(changes) == 0 {
		return ApplyResult{Status: "noop"}, nil
	}
	if options.PlanID == "" {
		options.PlanID = Hash(changeIdentity(changes))
	}
	if options.JournalPath == "" {
		options.JournalPath = JournalPath(changes[0].Path, options.PlanID)
	}
	if options.LockPath == "" {
		options.LockPath = changes[0].Path + ".profile-mango.lock"
	}
	release, err := acquireLock(options.LockPath)
	if err != nil {
		return ApplyResult{}, err
	}
	defer release()
	if err := preflight(changes); err != nil {
		return ApplyResult{}, err
	}
	entries, backups, err := prepareBackups(changes, options)
	if err != nil {
		return ApplyResult{}, err
	}
	journal := Journal{APIVersion: JournalVersion, PlanID: options.PlanID, Status: "prepared", LockPath: options.LockPath, Entries: entries}
	if err := writeJournal(options.JournalPath, journal); err != nil {
		return ApplyResult{}, err
	}
	for index := range changes {
		if options.FaultAfter > 0 && index == options.FaultAfter {
			return failTransaction(changes[:index], &journal, options, options.LeaveJournal)
		}
		if err := applyChange(changes[index]); err != nil {
			return failTransaction(changes[:index], &journal, options, options.LeaveJournal, err)
		}
		journal.Entries[index].Applied = true
		journal.Status = "applying"
		if err := writeJournal(options.JournalPath, journal); err != nil {
			return failTransaction(changes[:index+1], &journal, options, true, err)
		}
	}
	journal.Status = "committed"
	if err := writeJournal(options.JournalPath, journal); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Status: "committed", JournalPath: options.JournalPath, Backups: backups, Changed: changePaths(changes)}, nil
}

func changePaths(changes []Change) []string {
	paths := make([]string, len(changes))
	for index, change := range changes {
		paths[index] = change.Path
	}
	return paths
}

func Recover(journalPath string) error {
	journal, err := readJournal(journalPath)
	if err != nil {
		return err
	}
	if journal.Status == "committed" || journal.Status == "rolled-back" {
		return nil
	}
	release, err := acquireLock(journal.LockPath)
	if err != nil {
		return err
	}
	defer release()
	if err := validateRecovery(journal); err != nil {
		return err
	}
	for index := len(journal.Entries) - 1; index >= 0; index-- {
		entry := journal.Entries[index]
		if !entry.Applied {
			continue
		}
		if err := restoreEntry(entry); err != nil {
			return fmt.Errorf("restore %s: %w", entry.Path, err)
		}
		journal.Entries[index].Applied = false
	}
	journal.Status = "rolled-back"
	return writeJournal(journalPath, journal)
}

func Preflight(changes []Change) error {
	changes, err := normalizeChanges(changes)
	if err != nil {
		return err
	}
	return preflight(changes)
}

func normalizeChanges(changes []Change) ([]Change, error) {
	result := append([]Change(nil), changes...)
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	seen := make(map[string]struct{}, len(result))
	for index := range result {
		abs, _, err := inspectPath(result[index].Path, true)
		if err != nil {
			return nil, err
		}
		if _, found := seen[abs]; found {
			return nil, fmt.Errorf("duplicate change path: %s", abs)
		}
		seen[abs] = struct{}{}
		result[index].Path = abs
		if err := ValidateContent(result[index].Content); err != nil && !result[index].Delete {
			return nil, fmt.Errorf("validate %s: %w", abs, err)
		}
	}
	return result, nil
}

func preflight(changes []Change) error {
	for _, change := range changes {
		current, err := SnapshotFile(change.Path)
		if err != nil {
			return err
		}
		if !current.Equal(change.Before) {
			return fmt.Errorf("%s: %w", change.Path, ErrStale)
		}
	}
	return nil
}

func prepareBackups(changes []Change, options ApplyOptions) ([]JournalEntry, []string, error) {
	entries := make([]JournalEntry, len(changes))
	backups := make([]string, 0, len(changes))
	for index, change := range changes {
		entry := JournalEntry{Path: change.Path, BeforeExists: change.Before.Exists, BeforeSHA256: change.Before.SHA256, BeforeMode: uint32(change.Before.Mode), AfterSHA256: Hash(change.Content), Delete: change.Delete}
		if options.Backup && change.Before.Exists {
			backup := BackupPath(change.Path, options.PlanID)
			if _, err := os.Lstat(backup); err == nil {
				return nil, nil, fmt.Errorf("backup already exists: %s", backup)
			} else if !os.IsNotExist(err) {
				return nil, nil, fmt.Errorf("inspect backup %s: %w", backup, err)
			}
			if err := writeNewFile(backup, change.Before.Content, change.Before.Mode); err != nil {
				return nil, nil, fmt.Errorf("create backup %s: %w", backup, err)
			}
			entry.BackupPath = backup
			backups = append(backups, backup)
		}
		entries[index] = entry
	}
	return entries, backups, nil
}

func applyChange(change Change) error {
	current, err := SnapshotFile(change.Path)
	if err != nil {
		return err
	}
	if !current.Equal(change.Before) {
		return fmt.Errorf("%s: %w", change.Path, ErrStale)
	}
	if change.Delete {
		if !current.Exists {
			return nil
		}
		return os.Remove(change.Path)
	}
	return atomicReplace(change.Path, change.Content, current, change.Before.Mode)
}

func failTransaction(changes []Change, journal *Journal, options ApplyOptions, leave bool, cause ...error) (ApplyResult, error) {
	if leave {
		journal.Status = "recovery-required"
		if err := writeJournal(options.JournalPath, *journal); err != nil {
			return ApplyResult{}, fmt.Errorf("journal failure after transaction error: %w", err)
		}
		return ApplyResult{Status: "recovery-required", JournalPath: options.JournalPath}, transactionError(cause...)
	}
	if err := rollback(changes, journal); err != nil {
		journal.Status = "recovery-required"
		_ = writeJournal(options.JournalPath, *journal)
		return ApplyResult{Status: "recovery-required", JournalPath: options.JournalPath}, fmt.Errorf("%w: %v", ErrRecoveryRequired, err)
	}
	journal.Status = "rolled-back"
	if err := writeJournal(options.JournalPath, *journal); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{Status: "rolled-back", JournalPath: options.JournalPath}, transactionError(cause...)
}

func rollback(changes []Change, journal *Journal) error {
	for index := len(changes) - 1; index >= 0; index-- {
		if !journal.Entries[index].Applied {
			continue
		}
		if err := restoreChange(changes[index], journal.Entries[index]); err != nil {
			return err
		}
		journal.Entries[index].Applied = false
	}
	return nil
}

func restoreChange(change Change, entry JournalEntry) error {
	current, err := SnapshotFile(change.Path)
	if err != nil {
		return err
	}
	if !current.Exists || current.SHA256 != entry.AfterSHA256 {
		return fmt.Errorf("%s is not transaction-owned", change.Path)
	}
	if !entry.BeforeExists {
		return os.Remove(change.Path)
	}
	content := change.Before.Content
	mode := change.Before.Mode
	if entry.BackupPath != "" {
		backup, err := SnapshotFile(entry.BackupPath)
		if err != nil {
			return err
		}
		content, mode = backup.Content, backup.Mode
	}
	return atomicReplace(change.Path, content, current, mode)
}

func validateRecovery(journal Journal) error {
	for _, entry := range journal.Entries {
		if !entry.Applied {
			continue
		}
		current, err := SnapshotFile(entry.Path)
		if err != nil {
			return err
		}
		if current.Exists && current.SHA256 == entry.BeforeSHA256 {
			continue
		}
		if !current.Exists || current.SHA256 != entry.AfterSHA256 {
			return fmt.Errorf("%s: %w", entry.Path, ErrRecoveryRequired)
		}
		if entry.BeforeExists && entry.BackupPath == "" {
			return fmt.Errorf("%s has no recovery backup: %w", entry.Path, ErrRecoveryRequired)
		}
	}
	return nil
}

func restoreEntry(entry JournalEntry) error {
	current, err := SnapshotFile(entry.Path)
	if err != nil {
		return err
	}
	if current.Exists && current.SHA256 == entry.BeforeSHA256 {
		return nil
	}
	if !current.Exists || current.SHA256 != entry.AfterSHA256 {
		return fmt.Errorf("%s: %w", entry.Path, ErrRecoveryRequired)
	}
	if !entry.BeforeExists {
		return os.Remove(entry.Path)
	}
	backup, err := SnapshotFile(entry.BackupPath)
	if err != nil {
		return err
	}
	return atomicReplace(entry.Path, backup.Content, current, backup.Mode)
}

func transactionError(cause ...error) error {
	if len(cause) == 0 {
		return errors.New("transaction fault injected")
	}
	return cause[0]
}

func readJournal(path string) (Journal, error) {
	snapshot, err := SnapshotFile(path)
	if err != nil {
		return Journal{}, err
	}
	if !snapshot.Exists {
		return Journal{}, fmt.Errorf("journal does not exist: %s", path)
	}
	var journal Journal
	decoder := json.NewDecoder(strings.NewReader(string(snapshot.Content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return Journal{}, fmt.Errorf("decode journal: %w", err)
	}
	if journal.APIVersion != JournalVersion {
		return Journal{}, fmt.Errorf("unsupported journal version %q", journal.APIVersion)
	}
	return journal, nil
}

func writeJournal(path string, journal Journal) error {
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal: %w", err)
	}
	return writeAtomicUnconditional(path, append(data, '\n'), 0o600)
}

func writeNewFile(path string, data []byte, mode fs.FileMode) error {
	abs, _, err := inspectPath(path, true)
	if err != nil {
		return err
	}
	if err := ValidateContent(data); err != nil {
		return err
	}
	file, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, chooseMode(mode))
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(abs)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(abs)
		return err
	}
	return file.Close()
}

func writeAtomicUnconditional(path string, data []byte, mode fs.FileMode) error {
	abs, _, err := inspectPath(path, true)
	if err != nil {
		return err
	}
	if err := ValidateContent(data); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(abs), ".profile-mango-install-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(chooseMode(mode)); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, abs); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(abs))
}

func atomicReplace(path string, data []byte, expected Snapshot, mode fs.FileMode) error {
	current, err := SnapshotFile(path)
	if err != nil {
		return err
	}
	if !current.Equal(expected) {
		return fmt.Errorf("%s: %w", path, ErrStale)
	}
	return writeAtomicUnconditional(path, data, chooseMode(mode))
}

func acquireLock(path string) (func(), error) {
	abs, _, err := inspectPath(path, true)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("%s: %w", abs, ErrLockHeld)
		}
		return nil, err
	}
	if _, err := io.WriteString(file, "profile-mango installer lock\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(abs)
		return nil, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(abs)
		return nil, err
	}
	return func() { _ = os.Remove(abs) }, nil
}

func inspectPath(path string, allowMissing bool) (string, os.FileInfo, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return "", nil, fmt.Errorf("path is required and must not contain NUL")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("resolve path: %w", err)
	}
	abs = filepath.Clean(abs)
	parts := strings.Split(strings.TrimPrefix(abs, string(os.PathSeparator)), string(os.PathSeparator))
	current := string(os.PathSeparator)
	for index, part := range parts {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			if os.IsNotExist(statErr) && allowMissing && index == len(parts)-1 {
				return abs, nil, nil
			}
			return "", nil, fmt.Errorf("inspect path %s: %w", current, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fmt.Errorf("path component is a symlink: %s", current)
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", nil, fmt.Errorf("path parent is not a directory: %s", current)
		}
		if index == len(parts)-1 {
			return abs, info, nil
		}
	}
	return abs, nil, fmt.Errorf("invalid path: %s", path)
}

func validateRegular(path string, info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file: %s", path)
	}
	if links := linkCount(info); links > 1 {
		return fmt.Errorf("hard-linked files are not supported: %s", path)
	}
	if info.Size() > MaxFileBytes {
		return fmt.Errorf("file exceeds %d-byte limit: %s", MaxFileBytes, path)
	}
	return nil
}

func readRegular(path string, expected os.FileInfo) ([]byte, os.FileInfo, error) {
	file, err := openReadNoFollow(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s without following links: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if err := validateRegular(path, info); err != nil {
		return nil, nil, err
	}
	if !sameIdentity(expected, info) {
		return nil, nil, fmt.Errorf("file changed while opening %s: %w", path, ErrStale)
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	if err := ValidateContent(data); err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, info, nil
}

func chooseMode(mode fs.FileMode) fs.FileMode {
	if mode.Perm() == 0 {
		return 0o600
	}
	return mode.Perm()
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func shortID(value string) string {
	if value == "" {
		return "unbound"
	}
	value = strings.Trim(value, "-_")
	if len(value) > 16 {
		return value[:16]
	}
	return value
}

func changeIdentity(changes []Change) []byte {
	var builder strings.Builder
	for _, change := range changes {
		builder.WriteString(change.Path)
		builder.WriteByte('\x00')
		builder.WriteString(Hash(change.Content))
		builder.WriteByte('\x00')
	}
	return []byte(builder.String())
}
