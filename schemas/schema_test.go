package schemas_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

const schemaBaseURL = "https://profilemango.dev/schemas/"

func TestSchemasAcceptValidDocuments(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		schema   string
		document string
		yaml     bool
	}{
		"profile":  {schema: "profile.schema.json", document: "../pkg/profilemango/testdata/fixtures/route-only/profile.yaml", yaml: true},
		"bindings": {schema: "bindings.schema.json", document: "../pkg/profilemango/testdata/fixtures/bindings.yaml", yaml: true},
		"plan":     {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
		"manifest": {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":1,"profile":"route-only","target":"codex","resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			schema := compileSchema(t, test.schema)
			if err := schema.Validate(loadDocument(t, test.document, test.yaml)); err != nil {
				t.Fatalf("valid document rejected: %v", err)
			}
		})
	}
}

func TestSchemasRejectInvalidDocuments(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		schema   string
		document string
	}{
		"profile missing metadata":     {schema: "profile.schema.json", document: `{"apiVersion":"profilemango.dev/v1alpha1","kind":"PolicyProfile","spec":{"routeRef":"codex-oauth"}}`},
		"bindings empty auth":          {schema: "bindings.schema.json", document: `{"routes":{"codex-oauth":{"provider":"openai","transport":"native","authentication":"","model":"gpt-5.6","effort":"high"}}}`},
		"plan malformed digest":        {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"short","size":12}]}`},
		"manifest negative generation": {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":-1,"profile":"route-only","target":"codex","resources":[]}`},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			schema := compileSchema(t, test.schema)
			if err := schema.Validate(loadDocument(t, test.document, false)); err == nil {
				t.Fatal("invalid document accepted")
			}
		})
	}
}

func compileSchema(t *testing.T, filename string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	for _, name := range []string{"profile.schema.json", "bindings.schema.json", "plan.schema.json", "manifest.schema.json"} {
		document := loadDocument(t, name, false)
		if err := compiler.AddResource(schemaBaseURL+name, document); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}
	schema, err := compiler.Compile(schemaBaseURL + filename)
	if err != nil {
		t.Fatalf("compile %s: %v", filename, err)
	}
	return schema
}

func loadDocument(t *testing.T, source string, isYAML bool) any {
	t.Helper()
	data := []byte(source)
	if isYAML || source[0] != '{' {
		var err error
		data, err = os.ReadFile(source)
		if err != nil {
			t.Fatalf("read %s: %v", source, err)
		}
	}
	var document any
	if isYAML {
		if err := yaml.Unmarshal(data, &document); err != nil {
			t.Fatalf("decode %s: %v", source, err)
		}
		return document
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode %s: %v", source, err)
	}
	return document
}
