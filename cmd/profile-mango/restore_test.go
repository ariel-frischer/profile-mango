package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func TestCodexRestorePlanAndApply(t *testing.T) {
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(root, "config.toml")
	original := []byte("# retained\n[unrelated]\nvalue = 'yes'\n")
	if err := os.WriteFile(config, original, 0o640); err != nil {
		t.Fatal(err)
	}
	request := install.Request{ProfileName: "route-only", Default: true, ProfilesRoot: profiles, ResourceRoot: root, BindingsPath: bindings, Backup: true, Override: true,
		Targets: []install.TargetRequest{{Target: install.Target{Name: "codex", Version: "0.154.0"}, ConfigPath: config}}}
	installed, err := install.BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := install.ApplyPlan(installed, install.ApplyOptions{ExpectedPlanID: installed.PlanID}); err != nil {
		t.Fatalf("install: %v; plan=%+v", err, installed)
	}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	options := restoreOptions{target: "codex@0.154.0", config: config, originalPlan: installed.PlanID, jsonOutput: true}
	if err := runRestore(cmd, options); err != nil {
		t.Fatal(err)
	}
	var plan install.RestorePlan
	if err := json.Unmarshal(output.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Status != "ready" {
		t.Fatalf("plan = %+v", plan)
	}
	options.apply, options.yes, options.expectPlan = true, true, plan.PlanID
	output.Reset()
	if err := runRestore(cmd, options); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("restored = %q", got)
	}
	info, err := os.Stat(config)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, err = %v", info, err)
	}
	if err := runRestore(cmd, options); err == nil {
		t.Fatal("second restore accepted")
	}
}

func TestCodexRestoreRejectsTamperingWithoutWrites(t *testing.T) {
	tests := map[string]func(t *testing.T, config, id string){
		"edited config": func(t *testing.T, config, _ string) { writeFile(t, config, "edited") },
		"edited mode": func(t *testing.T, config, _ string) {
			if err := os.Chmod(config, 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"edited backup":   func(t *testing.T, config, id string) { writeFile(t, config+".profile-mango.bak."+id[:16], "edited") },
		"edited manifest": func(t *testing.T, config, _ string) { writeFile(t, config+".profile-mango.manifest.json", "{}") },
		"missing backup": func(t *testing.T, config, id string) {
			if err := os.Remove(config + ".profile-mango.bak." + id[:16]); err != nil {
				t.Fatal(err)
			}
		},
		"wrong journal destination": func(t *testing.T, config, id string) {
			path := config + ".profile-mango.journal." + id[:16] + ".json"
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, strings.Replace(string(data), `"path": "`+config+`"`, `"path": "`+config+`.other"`, 1))
		},
		"unexpected journal entry": func(t *testing.T, config, id string) {
			path := config + ".profile-mango.journal." + id[:16] + ".json"
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, strings.Replace(string(data), `"entries": [`, `"entries": [{"path":"/unrelated"},`, 1))
		},
		"duplicate journal key": func(t *testing.T, config, id string) {
			path := config + ".profile-mango.journal." + id[:16] + ".json"
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, strings.Replace(string(data), `"status": "committed",`, `"status": "committed", "status": "committed",`, 1))
		},
		"duplicate manifest key": func(t *testing.T, config, _ string) {
			path := config + ".profile-mango.manifest.json"
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, strings.Replace(string(data), `"owner": "profile-mango",`, `"owner": "profile-mango", "owner": "profile-mango",`, 1))
		},
		"trailing manifest JSON": func(t *testing.T, config, _ string) {
			path := config + ".profile-mango.manifest.json"
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, path, string(data)+"{}")
		},
		"symlinked backup": func(t *testing.T, config, id string) {
			path := config + ".profile-mango.bak." + id[:16]
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(config, path); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, tamper := range tests {
		t.Run(name, func(t *testing.T) {
			config, id := installedRestoreFixture(t, true)
			tamper(t, config, id)
			before, err := os.ReadFile(config)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := install.BuildRestorePlan("codex@0.154.0", config, id); err == nil {
				t.Fatal("tampered install accepted")
			}
			after, err := os.ReadFile(config)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("changed config after rejection: %v", err)
			}
		})
	}
}

func TestCodexRestoreRejectsWrongConsentAndNoBackup(t *testing.T) {
	config, id := installedRestoreFixture(t, true)
	plan, err := install.BuildRestorePlan("codex@0.154.0", config, id)
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]string{"omitted": "", "wrong": strings.Repeat("f", 64)} {
		t.Run(name, func(t *testing.T) {
			if err := install.ApplyRestorePlan(plan, expected); err == nil {
				t.Fatal("missing or wrong consent accepted")
			}
		})
	}
	if _, err := install.BuildRestorePlan("codex@0.154.0", config, strings.Repeat("0", 64)); err == nil {
		t.Fatal("wrong original ID accepted")
	}
	if _, err := install.BuildRestorePlan("codex@0.153.0", config, id); err == nil {
		t.Fatal("wrong target accepted")
	}
	if err := os.WriteFile(config, []byte("changed after preview"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := install.ApplyRestorePlan(plan, plan.PlanID); err == nil {
		t.Fatal("stale preview accepted")
	}
	withoutBackup, otherID := installedRestoreFixture(t, false)
	if _, err := install.BuildRestorePlan("codex@0.154.0", withoutBackup, otherID); err == nil {
		t.Fatal("no-backup install accepted")
	}
}

func TestCodexRestoreDeletesOriginallyAbsentConfigAndManifest(t *testing.T) {
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(root, "config.toml")
	plan, err := install.BuildPlan(install.Request{ProfileName: "route-only", Default: true, ProfilesRoot: profiles, ResourceRoot: root, BindingsPath: bindings, Backup: true,
		Targets: []install.TargetRequest{{Target: install.Target{Name: "codex", Version: "0.154.0"}, ConfigPath: config}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := install.ApplyPlan(plan, install.ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	restore, err := install.BuildRestorePlan("codex@0.154.0", config, plan.PlanID)
	if err != nil {
		t.Fatal(err)
	}
	if err := install.ApplyRestorePlan(restore, restore.PlanID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{config, config + ".profile-mango.manifest.json"} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("originally absent %s still exists: %v", path, err)
		}
	}
}

func TestRestoreCLIHelpAndConsent(t *testing.T) {
	var output bytes.Buffer
	command := newRestoreCmd()
	command.SetOut(&output)
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"--original-plan", "--expect-plan", "--config", "--json"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("help lacks %s", text)
		}
	}
	config, id := installedRestoreFixture(t, true)
	for name, options := range map[string]restoreOptions{
		"omitted consent":      {target: "codex@0.154.0", config: config, originalPlan: id, apply: true},
		"bare target":          {target: "codex", config: config, originalPlan: id, apply: true},
		"deprecated alias":     {target: "codex", legacyConfig: config, originalPlan: id, apply: true},
		"json without consent": {target: "codex@0.154.0", config: config, originalPlan: id, apply: true, jsonOutput: true},
	} {
		t.Run(name, func(t *testing.T) {
			if err := runRestore(command, options); err == nil || !strings.Contains(err.Error(), "--yes --expect-plan") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRestoreInteractiveConsentAndHumanPath(t *testing.T) {
	for name, test := range map[string]struct {
		input           string
		terminal, allow bool
	}{
		"terminal yes":   {input: "y\n", terminal: true, allow: true},
		"terminal empty": {input: "\n", terminal: true},
		"terminal no":    {input: "n\n", terminal: true},
		"redirected yes": {input: "y\n"},
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(test.input))
			cmd.SetOut(&output)
			err := authorizeRestore(cmd, restoreOptions{apply: true}, strings.Repeat("a", 64), test.terminal)
			if (err == nil) != test.allow {
				t.Fatalf("authorization = %v", err)
			}
		})
	}
	var output bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&output)
	path := "/synthetic/unsafe\n\x1b[31m.toml"
	plan := install.RestorePlan{PlanID: strings.Repeat("a", 64), Status: "ready", Files: []install.RestoreFile{{Path: path, Action: "update"}}}
	if err := writeRestorePlan(cmd, plan, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), path) || !strings.Contains(output.String(), `\n\x1b`) {
		t.Fatalf("unsanitized human restore path: %q", output.String())
	}
}

func TestCodexRestoreRejectsSameContentReplacementAfterPreview(t *testing.T) {
	for name, pathFor := range map[string]func(string, string) string{
		"config":   func(config, _ string) string { return config },
		"manifest": func(config, _ string) string { return config + ".profile-mango.manifest.json" },
		"journal":  func(config, id string) string { return config + ".profile-mango.journal." + id[:16] + ".json" },
		"backup":   func(config, id string) string { return config + ".profile-mango.bak." + id[:16] },
	} {
		t.Run(name, func(t *testing.T) {
			config, id := installedRestoreFixture(t, true)
			plan, err := install.BuildRestorePlan("codex@0.154.0", config, id)
			if err != nil {
				t.Fatal(err)
			}
			path := pathFor(config, id)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			replacement := path + ".replacement"
			if err := os.WriteFile(replacement, data, info.Mode().Perm()); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(replacement, path); err != nil {
				t.Fatal(err)
			}
			if err := install.ApplyRestorePlan(plan, plan.PlanID); err == nil {
				t.Fatal("same-content replacement accepted")
			}
			if _, err := os.Stat(config); err != nil {
				t.Fatalf("restore wrote on stale source: %v", err)
			}
		})
	}
}

func TestCodexRestoreOriginalManifestCanonicalBoundary(t *testing.T) {
	for name, canonical := range map[string]bool{"generated": true, "reformatted": false} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			profiles, bindings := writeCodexInstallInputs(t, root)
			config := filepath.Join(root, "config.toml")
			original := []byte("# original\n")
			if err := os.WriteFile(config, original, 0o640); err != nil {
				t.Fatal(err)
			}
			manifestPath := config + ".profile-mango.manifest.json"
			owner := install.Manifest{APIVersion: install.ManifestAPIVersion, Kind: install.ManifestKind, Owner: "profile-mango", Generation: 1,
				Profile: "route-only", Target: install.Target{Name: "codex", Version: "0.154.0"}, Files: []install.ManifestFile{{Path: config, SHA256: installfs.Hash(original)}}}
			prior, err := json.MarshalIndent(owner, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			prior = append(prior, '\n')
			if !canonical {
				prior = append(prior, '\n')
			}
			if err := os.WriteFile(manifestPath, prior, 0o640); err != nil {
				t.Fatal(err)
			}
			plan, err := install.BuildPlan(install.Request{ProfileName: "route-only", Default: true, ProfilesRoot: profiles, ResourceRoot: root, BindingsPath: bindings, Backup: true,
				Targets: []install.TargetRequest{{Target: owner.Target, ConfigPath: config}}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := install.ApplyPlan(plan, install.ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
				t.Fatal(err)
			}
			restore, err := install.BuildRestorePlan("codex@0.154.0", config, plan.PlanID)
			if !canonical {
				if err == nil {
					t.Fatal("reformatted original ownership manifest accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := install.ApplyRestorePlan(restore, restore.PlanID); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(manifestPath)
			if err != nil || !bytes.Equal(got, prior) {
				t.Fatalf("manifest restoration: %v, bytes=%q", err, got)
			}
		})
	}
}

func installedRestoreFixture(t *testing.T, backup bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	profiles, bindings := writeCodexInstallInputs(t, root)
	config := filepath.Join(root, "config.toml")
	if err := os.WriteFile(config, []byte("# original\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	plan, err := install.BuildPlan(install.Request{ProfileName: "route-only", Default: true, ProfilesRoot: profiles, ResourceRoot: root, BindingsPath: bindings, Backup: backup, Override: true,
		Targets: []install.TargetRequest{{Target: install.Target{Name: "codex", Version: "0.154.0"}, ConfigPath: config}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := install.ApplyPlan(plan, install.ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	return config, plan.PlanID
}
