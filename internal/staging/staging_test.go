package staging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCreatesOnlyExplicitNewDirectory(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "bundle")
	files := []File{
		{Path: "render.json", Content: []byte(`{"applicable":false}`)},
		{Path: "preview/demo.config.toml.preview", Content: []byte("# inert\n")},
		{Path: "resources/instructions/system.md", Content: []byte("system\n")},
	}
	if err := Write(destination, files); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(file.Path))); err != nil {
			t.Fatalf("missing %s: %v", file.Path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("active config exists, err=%v", err)
	}
}

func TestWriteRejectsExistingDirectoryWithoutMutation(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "marker")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(destination, []File{{Path: "render.json", Content: []byte("new")}}); err == nil {
		t.Fatal("existing destination was overwritten")
	}
	content, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep" {
		t.Fatalf("existing destination changed: %q", content)
	}
	if _, err := os.Stat(filepath.Join(destination, "render.json")); !os.IsNotExist(err) {
		t.Fatalf("unexpected output in existing destination: %v", err)
	}
}

func TestWriteDoesNotReplaceDestinationCreatedBeforeCommit(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "bundle")
	originalCommit := commitDirectory
	commitDirectory = func(source, target string) error {
		if err := os.Mkdir(target, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(target, "marker"), []byte("keep"), 0o644); err != nil {
			return err
		}
		return originalCommit(source, target)
	}
	defer func() { commitDirectory = originalCommit }()

	err := Write(destination, []File{{Path: "render.json", Content: []byte("new")}})
	if err == nil {
		t.Fatal("staging succeeded after destination was created before commit")
	}
	marker, readErr := os.ReadFile(filepath.Join(destination, "marker"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(marker) != "keep" {
		t.Fatalf("concurrent destination was changed: %q", marker)
	}
	if _, statErr := os.Stat(filepath.Join(destination, "render.json")); !os.IsNotExist(statErr) {
		t.Fatalf("staging wrote into concurrent destination: %v", statErr)
	}
}

func TestWriteRejectsInvalidInputWithoutCreatingDestination(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := map[string][]File{
		"escape":   {{Path: "../escape", Content: []byte("bad")}},
		"absolute": {{Path: "/escape", Content: []byte("bad")}},
		"duplicate": {
			{Path: "render.json", Content: []byte("one")},
			{Path: "render.json", Content: []byte("two")},
		},
	}
	for name, files := range tests {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(destination, name)
			if err := Write(out, files); err == nil {
				t.Fatal("invalid files were accepted")
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("invalid input created output: %v", err)
			}
		})
	}
}

func TestWriteRejectsOversizedOutput(t *testing.T) {
	t.Parallel()
	destination := filepath.Join(t.TempDir(), "bundle")
	if err := Write(destination, []File{{Path: "render.json", Content: []byte(strings.Repeat("x", int(MaxFileBytes)+1))}}); err == nil {
		t.Fatal("oversized output was accepted")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("oversized input created output: %v", err)
	}
}
