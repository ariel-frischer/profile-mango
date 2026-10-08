package install

import (
	"os"
	"strings"
	"testing"
)

func TestOhMyPiBindingPresetsLifecycle(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	config := request.Targets[0].ConfigPath
	original := "# keep\nmodelPresets:\n  personal:\n    modelRoles: {default: mine/model} # keep\n"
	writeInstallTestFile(t, config, original)
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings+"  alternate:\n    provider: openai\n    model: gpt-6\n    effort: low\n")
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	applySwitchTestPlan(t, request)
	data, _ := os.ReadFile(config)
	for _, want := range []string{"mango-primary:", "mango-alternate:", "default: mine/model} # keep"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s in %s", want, data)
		}
	}
	found := false
	for _, field := range plan.Targets[0].Fields {
		if field.Path == "config.modelPresets.mango-alternate" {
			found = true
		}
	}
	if !found {
		t.Fatal("presets absent from plan")
	}
	again, err := BuildPlan(request)
	if err != nil || again.Status != StatusNoop {
		t.Fatalf("reapply: %v %#v", err, again)
	}
	writeInstallTestFile(t, request.BindingsPath, ohMyPiTestBindings)
	applySwitchTestPlan(t, request)
	data, _ = os.ReadFile(config)
	if strings.Contains(string(data), "mango-alternate:") {
		t.Fatal("removed route preset survived release")
	}
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
}

func TestOhMyPiPresetNameCollision(t *testing.T) {
	request, _ := ohMyPiTestRequest(t)
	request.Override = false
	config := request.Targets[0].ConfigPath
	original := "modelPresets:\n  mango-primary:\n    modelRoles: {default: mine/model}\n"
	writeInstallTestFile(t, config, original)
	plan, err := BuildPlan(request)
	if err != nil || plan.Targets[0].Status != StatusConflict {
		t.Fatalf("collision: %v %s", err, plan.Status)
	}
	request.Override = true
	applySwitchTestPlan(t, request)
	applyUndo(t, UndoRequest{Target: request.Targets[0].Target, ConfigPath: config, Registry: request.Registry, StateDir: testStateDir})
	assertInstallTestFile(t, config, original)
}
