package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const starterProfileYAML = `# The profile name defaults to its folder name (default).
description: Default profile scaffolded by mango init
route: local
`

const starterBindingsYAML = `# Route identity only. Credentials remain target-owned and are never stored here.
# transport defaults to native and authentication to oauth when omitted.
routes:
  local:
    provider: openai
    model: gpt-6-sol
    effort: high
    # Each listed agent gets the route above with these fields replaced.
    targets:
      claude-code:
        provider: anthropic
        model: claude-sonnet-5
`

const starterBindingsGitignore = `# Keep machine-local bindings out of version control.
local.yaml
`

type starterFile struct {
	path    string
	content []byte
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "init [directory]",
		Aliases: []string{"new"},
		Short:   "Create a starter profile you can edit",
		Args:    cobra.MaximumNArgs(1),
		RunE:    runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	destination, err := initDestination(args)
	if err != nil {
		return err
	}
	files := starterFiles()
	if err := validateStarterFiles(files); err != nil {
		return err
	}
	if err := preflightInit(destination, files); err != nil {
		return err
	}
	if err := writeInitFiles(destination, files); err != nil {
		return err
	}
	if err := writeInitLogo(cmd.OutOrStdout()); err != nil {
		return err
	}
	styles := stylesFor(cmd.OutOrStdout(), true)
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s profile scaffold in %s\n", styles.success("Created"), styles.path(filepath.Clean(destination))); err != nil {
		return fmt.Errorf("reporting initialized directory: %w", err)
	}
	return nil
}

func initDestination(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return selectedHome()
}

func starterFiles() []starterFile {
	return []starterFile{
		{path: "profiles/default/profile.yaml", content: []byte(starterProfileYAML)},
		{path: "bindings/local.example.yaml", content: []byte(starterBindingsYAML)},
		{path: "bindings/local.yaml", content: []byte(starterBindingsYAML)},
		{path: "bindings/.gitignore", content: []byte(starterBindingsGitignore)},
	}
}

func validateStarterFiles(files []starterFile) error {
	profile, profileDiagnostics := profilemango.ParseProfileAt(files[0].content, "default")
	bindings, bindingDiagnostics := profilemango.ParseBindings(files[1].content)
	diagnostics := append(profileDiagnostics, bindingDiagnostics...)
	if diagnostics.HasErrors() {
		return fmt.Errorf("validating starter files: %w", diagnostics.Err())
	}
	if _, found := bindings.Routes[profile.Route]; !found {
		return fmt.Errorf("validating starter files: route %q is absent from bindings", profile.Route)
	}
	return nil
}

func preflightInit(destination string, files []starterFile) error {
	for _, file := range files {
		path := filepath.Join(destination, file.path)
		if err := checkInitParents(path); err != nil {
			return fmt.Errorf("checking parent for %s: %w", path, err)
		}
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("refusing to overwrite %s (%s already exists)", path, file.path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("checking %s: %w", path, err)
		}
	}
	return nil
}

func checkInitParents(path string) error {
	current := filepath.Clean(filepath.Dir(path))
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is not a directory", current)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspecting %s: %w", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func writeInitFiles(destination string, files []starterFile) error {
	createdDirs := make([]string, 0)
	createdFiles := make([]string, 0, len(files))
	for _, file := range files {
		path := filepath.Join(destination, file.path)
		if err := ensureInitDirectory(filepath.Dir(path), &createdDirs); err != nil {
			return initWriteFailure(err, createdFiles, createdDirs)
		}
		if err := writeInitFile(path, file.content, &createdFiles); err != nil {
			return initWriteFailure(err, createdFiles, createdDirs)
		}
	}
	return nil
}

func ensureInitDirectory(path string, created *[]string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is not a directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", path, err)
	}
	parent := filepath.Dir(path)
	if parent == path {
		return fmt.Errorf("cannot create directory %s", path)
	}
	if err := ensureInitDirectory(parent, created); err != nil {
		return err
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	*created = append(*created, path)
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("setting mode on %s: %w", path, err)
	}
	return nil
}

func writeInitFile(path string, content []byte, created *[]string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	*created = append(*created, path)
	if written, writeErr := file.Write(content); writeErr != nil {
		_ = file.Close()
		return fmt.Errorf("writing %s: %w", path, writeErr)
	} else if written != len(content) {
		_ = file.Close()
		return fmt.Errorf("writing %s: %w", path, io.ErrShortWrite)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		return fmt.Errorf("setting mode on %s: %w", path, err)
	}
	return nil
}

func initWriteFailure(cause error, files, dirs []string) error {
	if err := rollbackInit(files, dirs); err != nil {
		return fmt.Errorf("%w; rolling back scaffold: %v", cause, err)
	}
	return cause
}

func rollbackInit(files, dirs []string) error {
	var firstErr error
	for index := len(files) - 1; index >= 0; index-- {
		if err := os.Remove(files[index]); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("removing %s: %w", files[index], err)
		}
	}
	for index := len(dirs) - 1; index >= 0; index-- {
		if err := os.Remove(dirs[index]); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("removing %s: %w", dirs[index], err)
		}
	}
	return firstErr
}
