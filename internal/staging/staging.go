package staging

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	MaxFileBytes  int64 = 8 << 20
	MaxTotalBytes int64 = 32 << 20
)

// File is one relative regular file to place in a new staging directory.
type File struct {
	Path    string
	Content []byte
}

// Stage atomically creates destination and writes only the supplied regular files.
func Stage(destination string, files []File) error {
	parent, err := validateDestination(destination)
	if err != nil {
		return err
	}
	if err := validateFiles(files); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".profile-mango-stage-*")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(temporary)
		}
	}()
	if err := writeFiles(temporary, files); err != nil {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("commit staging directory: %w", err)
	}
	committed = true
	return nil
}

// Write is the descriptive alias for Stage used by callers that write a bundle.
func Write(destination string, files []File) error {
	return Stage(destination, files)
}

func validateDestination(destination string) (string, error) {
	if strings.TrimSpace(destination) == "" {
		return "", fmt.Errorf("staging destination is required")
	}
	info, err := os.Lstat(destination)
	if err == nil {
		if info.IsDir() {
			return "", fmt.Errorf("staging destination already exists: %s", destination)
		}
		return "", fmt.Errorf("staging destination is not a directory: %s", destination)
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("inspect staging destination: %w", err)
	}
	parent := filepath.Dir(destination)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return "", fmt.Errorf("inspect staging parent: %w", err)
	}
	if !parentInfo.IsDir() {
		return "", fmt.Errorf("staging parent is not a directory: %s", parent)
	}
	return parent, nil
}

func validateFiles(files []File) error {
	seen := make(map[string]struct{}, len(files))
	var total int64
	for _, file := range files {
		if !validRelativePath(file.Path) {
			return fmt.Errorf("invalid staged file path: %q", file.Path)
		}
		if _, found := seen[file.Path]; found {
			return fmt.Errorf("duplicate staged file path: %q", file.Path)
		}
		seen[file.Path] = struct{}{}
		if int64(len(file.Content)) > MaxFileBytes {
			return fmt.Errorf("staged file exceeds size limit: %q", file.Path)
		}
		total += int64(len(file.Content))
		if total > MaxTotalBytes {
			return fmt.Errorf("staged files exceed total size limit")
		}
	}
	return nil
}

func writeFiles(root string, files []File) error {
	for _, file := range files {
		filePath := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return fmt.Errorf("create directory for %q: %w", file.Path, err)
		}
		if err := os.WriteFile(filePath, file.Content, 0o644); err != nil {
			return fmt.Errorf("write staged file %q: %w", file.Path, err)
		}
	}
	return nil
}

func validRelativePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	clean := path.Clean(value)
	return clean == value && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
