package schemas_test

import (
	"encoding/json"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The doctor report schema has no $ref into the other contracts, so it is
// compiled with its own standalone compiler instead of sharing
// compileSchema's cross-referenced registration list.
func compileDoctorSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	document := loadDocument(t, "doctor.schema.json", false)
	const id = schemaBaseURL + "doctor.schema.json"
	if err := compiler.AddResource(id, document); err != nil {
		t.Fatalf("register doctor.schema.json: %v", err)
	}
	schema, err := compiler.Compile(id)
	if err != nil {
		t.Fatalf("compile doctor.schema.json: %v", err)
	}
	return schema
}

func TestDoctorSchemaAcceptsValidReport(t *testing.T) {
	t.Parallel()
	schema := compileDoctorSchema(t)
	document := `{"apiVersion":"profilemango.dev/doctor/v1alpha1","kind":"DoctorReport","profile":"default","targets":[{"target":"codex@0.154.0","binary":"codex","binaryFound":true,"binaryPath":"/usr/local/bin/codex","detectedVersion":"codex-cli 0.154.0","qualifiedVersion":"0.154.0","versionMatch":true,"configPath":"/home/user/.codex/config.toml","configExists":false,"planStatus":"ready"}]}`
	var parsed any
	if err := json.Unmarshal([]byte(document), &parsed); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(parsed); err != nil {
		t.Fatalf("valid doctor report rejected: %v", err)
	}
}

func TestDoctorSchemaRejectsInvalidReport(t *testing.T) {
	t.Parallel()
	schema := compileDoctorSchema(t)
	tests := map[string]string{
		"missing kind":        `{"apiVersion":"profilemango.dev/doctor/v1alpha1","profile":"default","targets":[]}`,
		"unknown plan status": `{"apiVersion":"profilemango.dev/doctor/v1alpha1","kind":"DoctorReport","profile":"default","targets":[{"target":"codex@0.154.0","binary":"codex","binaryFound":false,"qualifiedVersion":"0.154.0","versionMatch":false,"configExists":false,"planStatus":"maybe"}]}`,
		"unknown field":       `{"apiVersion":"profilemango.dev/doctor/v1alpha1","kind":"DoctorReport","profile":"default","targets":[],"extra":true}`,
	}
	for name, document := range tests {
		t.Run(name, func(t *testing.T) {
			var parsed any
			if err := json.Unmarshal([]byte(document), &parsed); err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(parsed); err == nil {
				t.Fatal("invalid doctor report accepted")
			}
		})
	}
}
