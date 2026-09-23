package schemas_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		"profile":          {schema: "profile.schema.json", document: "../pkg/profilemango/testdata/fixtures/route-only/profile.yaml", yaml: true},
		"bindings":         {schema: "bindings.schema.json", document: "../pkg/profilemango/testdata/fixtures/bindings.yaml", yaml: true},
		"profile legacy":   {schema: "profile.schema.json", document: `{"apiVersion":"profilemango.dev/v1alpha1","kind":"PolicyProfile","metadata":{"name":"route-only"},"spec":{"routeRef":"codex-oauth"}}`},
		"profile nameless": {schema: "profile.schema.json", document: `{"extends":"base","permissions":{"mode":"read-only"}}`},
		"bindings minimal": {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-6-sol","effort":"high"}}}`},
		"bindings targets": {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"claude-code":{"provider":"anthropic","model":"claude-sonnet-5"}}}}}`},
		"plan":             {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
		"manifest":         {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":1,"profile":"route-only","target":"codex","resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
		"render":           {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"codex","targetVersion":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","applicable":false,"preview":true,"evidence":{"target":"codex","version":"0.154.0","sha256":"3188814c35471432d4123203e0eb38e5bddc60226e3d7ddf0e59e649ea140022","source":"docs/dev/target-evidence.md","level":"native-config-parsing"},"capabilities":[{"field":"route.authentication","status":"blocking","reason":"authentication identity is not verified"}],"diagnostics":[{"severity":"error","code":"codex.route.authentication_unverified","path":"route.authentication","message":"exact Codex authentication route is unverified"}],"artifacts":[{"path":"preview/route-only.config.toml.preview","kind":"candidate-config","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":10}]}`},
		"render-oh-my-pi":  {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"oh-my-pi","targetVersion":"18.2.6","adapterVersion":"profilemango.dev/oh-my-pi/v1alpha1","applicable":false,"preview":true,"evidence":{"target":"oh-my-pi","version":"18.2.6","sha256":"4d9558530fdd8c76798181545d7cde8b558731596515b2f300e61c8d403bcb6e","source":"docs/dev/target-evidence.md","level":"source-entrypoint-version"},"capabilities":[{"field":"target.artifact","status":"blocking","reason":"compiled Oh My Pi artifact is unavailable"}],"diagnostics":[{"severity":"error","code":"ohmypi.config.inspector_unsafe","path":"target.config-inspection","message":"config inspection is blocked"}],"artifacts":[{"path":"preview/route-only.config.yml.preview","kind":"candidate-config","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":10}]}`},
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
		"profile mixed wrapper":        {schema: "profile.schema.json", document: `{"kind":"PolicyProfile","route":"codex-oauth"}`},
		"profile old route key":        {schema: "profile.schema.json", document: `{"routeRef":"codex-oauth"}`},
		"profile invalid name":         {schema: "profile.schema.json", document: `{"name":"Default","route":"codex-oauth"}`},
		"bindings missing model":       {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","effort":"high"}}}`},
		"bindings empty auth":          {schema: "bindings.schema.json", document: `{"routes":{"codex-oauth":{"provider":"openai","transport":"native","authentication":"","model":"gpt-5.6","effort":"high"}}}`},
		"bindings unknown target":      {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"claude":{"provider":"anthropic"}}}}}`},
		"bindings empty override":      {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{}}}}}`},
		"bindings override field":      {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{"token":"x"}}}}}`},
		"plan route with targets":      {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{"model":"x"}}},"resources":[]}`},
		"plan malformed digest":        {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"short","size":12}]}`},
		"manifest negative generation": {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":-1,"profile":"route-only","target":"codex","resources":[]}`},
		"render missing evidence":      {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"codex","targetVersion":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","applicable":false,"preview":true,"evidence":{},"capabilities":[],"diagnostics":[],"artifacts":[]}`},
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
	for _, name := range []string{"profile.schema.json", "bindings.schema.json", "plan.schema.json", "manifest.schema.json", "render.schema.json"} {
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

func TestBindingsSchemaTargetsMatchDomain(t *testing.T) {
	t.Parallel()
	var document struct {
		Properties struct {
			Routes struct {
				AdditionalProperties struct {
					Properties struct {
						Targets struct {
							PropertyNames struct {
								Enum []string `json:"enum"`
							} `json:"propertyNames"`
						} `json:"targets"`
					} `json:"properties"`
				} `json:"additionalProperties"`
			} `json:"routes"`
		} `json:"properties"`
	}
	data, err := os.ReadFile("bindings.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	enum := document.Properties.Routes.AdditionalProperties.Properties.Targets.PropertyNames.Enum
	if !reflect.DeepEqual(enum, profilemango.RouteTargets) {
		t.Fatalf("schema targets %v, domain targets %v", enum, profilemango.RouteTargets)
	}
}
