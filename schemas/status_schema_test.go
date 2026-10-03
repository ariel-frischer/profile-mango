package schemas_test

import (
	"encoding/json"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestStatusSchemaPresetSwitch(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	for _, name := range []string{"install-plan.schema.json", "status.schema.json"} {
		if err := compiler.AddResource(schemaBaseURL+name, loadDocument(t, name, false)); err != nil {
			t.Fatal(err)
		}
	}
	schema, err := compiler.Compile(schemaBaseURL + "status.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	report := install.StatusReport{APIVersion: "profilemango.dev/status/v1alpha1", Kind: "Status", Targets: []install.TargetStatus{{Target: install.Target{Name: "oh-my-pi", Version: "18.6.0"}, State: install.StatusManaged, Source: install.SourceCurrent, Files: []install.FileStatus{{Path: "config.yml", Kind: "config", State: install.FilePresetSwitched}}, ModelPreset: &install.ModelPresetStatus{Name: "mango-sol", Route: "sol", TaskMaxEffort: "high", RouteSubagentMaxEffort: "medium", MaxEffortDiffers: true}}}}
	bytes, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed any
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(parsed); err != nil {
		t.Fatalf("valid status rejected: %v", err)
	}
	preset := parsed.(map[string]any)["targets"].([]any)[0].(map[string]any)["modelPreset"].(map[string]any)
	preset["maxEffortDiffers"] = "yes"
	if err := schema.Validate(parsed); err == nil {
		t.Fatal("invalid preset cap status accepted")
	}
}
