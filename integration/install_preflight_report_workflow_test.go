package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
)

func TestInstalledBinaryPreflightReportPreservesNoop(t *testing.T) {
	w := mixedPreflightWorkflow(t)
	plan := w.plan(t)
	if plan.Status != install.StatusReady || len(plan.Targets) != 2 {
		t.Fatalf("expected mixed ready/noop plan: %#v", plan)
	}
	before := preflightWorkflowFiles(t, w)
	result := w.run(t, "--apply", "--yes", "--expect-plan", "not-"+plan.PlanID)
	if result.err == nil || !strings.Contains(result.stderr, "does not match actual plan") {
		t.Fatalf("expected consent mismatch exit: %v\n%s", result.err, result.stderr)
	}
	var report install.ApplyReport
	if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
		t.Fatalf("expected exactly one report: %v\n%s", err, result.stdout)
	}
	if report.Status != "not-attempted" || len(report.Targets) != 2 {
		t.Fatalf("wrong preflight report: %#v", report)
	}
	if report.Targets[0].Target != "opencode@1.18.31" || report.Targets[0].Status != install.StatusNoop || report.Targets[0].Error != "" {
		t.Fatalf("unchanged target mislabeled: %#v", report.Targets[0])
	}
	if report.Targets[1].Target != "pi@0.86.1" || report.Targets[1].Status != "not-attempted" || report.Targets[1].Error == "" {
		t.Fatalf("pending target mislabeled: %#v", report.Targets[1])
	}
	if after := preflightWorkflowFiles(t, w); !reflect.DeepEqual(before, after) {
		t.Fatal("preflight rejection changed synthetic files or created transaction artifacts")
	}
}

func mixedPreflightWorkflow(t *testing.T) installWorkflow {
	t.Helper()
	w := newInstallWorkflow(t, "opencode@1.18.31", "openai", "gpt-5.6", "{\"model\":\"openai/old\"}\n", "")
	plan := w.plan(t)
	if result := w.run(t, "--apply", "--yes", "--expect-plan", plan.PlanID); result.err != nil {
		t.Fatalf("seed no-op target: %v\n%s", result.err, result.stderr)
	}
	pi := filepath.Join(w.root, "pi.json")
	if err := os.WriteFile(pi, []byte("{\"defaultModel\":\"old\",\"keep\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.args = append(w.args, "--target", "pi@0.86.1", "--config-path", "pi="+pi)
	return w
}

func preflightWorkflowFiles(t *testing.T, w installWorkflow) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(w.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && path != w.binary {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			files[path] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
