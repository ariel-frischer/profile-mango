package schemas_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		"profile":                  {schema: "profile.schema.json", document: "../pkg/profilemango/testdata/fixtures/route-only/profile.yaml", yaml: true},
		"bindings":                 {schema: "bindings.schema.json", document: "../pkg/profilemango/testdata/fixtures/bindings.yaml", yaml: true},
		"profile legacy":           {schema: "profile.schema.json", document: `{"apiVersion":"profilemango.dev/v1alpha1","kind":"PolicyProfile","metadata":{"name":"route-only"},"spec":{"routeRef":"codex-oauth"}}`},
		"profile nameless":         {schema: "profile.schema.json", document: `{"extends":"base","permissions":{"mode":"read-only"}}`},
		"bindings minimal":         {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-6-sol","effort":"high"}}}`},
		"bindings targets":         {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"claude-code":{"provider":"anthropic","model":"claude-sonnet-5"}}}}}`},
		"bindings roles":           {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"anthropic","model":"claude-opus-5-5","effort":"medium","subagentMaxEffort":"high","roles":{"research":{"provider":"opencode-go","model":"gpt-6-luna","effort":"high"},"tiny":{"provider":"opencode-go","model":"glm-5.3-flash"}}}}}`},
		"bindings target roles":    {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai-codex","model":"gpt-6-sol","effort":"high","roles":{"research":{"provider":"openai-codex","model":"gpt-6-luna"}},"targets":{"codex":{"roles":{"research":{"provider":"openai"}}}}}}}`},
		"profile roles":            {schema: "profile.schema.json", document: `{"route":"main","roles":{"worker":{"description":"Implements"},"research":{"description":"Scouts","instructions":"roles/research.md"}}}`},
		"profile custom roles":     {schema: "profile.schema.json", document: `{"route":"main","roles":{"coder":{"description":"Implements"},"reviewer":{"description":"Reviews"}}}`},
		"bindings custom roles":    {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"m","effort":"high","roles":{"code":{"provider":"openai","model":"m"},"review":{"provider":"openai","model":"r"}},"targets":{"codex":{"roles":{"code":{"model":"other"}}}}}}}`},
		"profile agent files":      {schema: "profile.schema.json", document: `{"route":"main","agentFiles":{"oh-my-pi":{"scout.md":"agents/omp/scout.md"},"codex":{"reviewer.toml":"agents/codex/reviewer.toml"}}}`},
		"profile global fragments": {schema: "profile.schema.json", document: `{"route":"main","globalInstructions":{"oh-my-pi":{"AGENTS.md":["shared/core.md","omp/agents.md"],"RULES.md":"omp/rules.md"}}}`},
		"plan":                     {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
		"manifest":                 {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":1,"profile":"route-only","target":"codex","resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":12}]}`},
		"render":                   {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"codex","targetVersion":"0.157.1","adapterVersion":"profilemango.dev/codex/v1alpha1","applicable":false,"preview":true,"evidence":{"target":"codex","version":"0.157.1","sha256":"3e2584f3f3829a43a0495011a1cecb2facbe64a2403e2b682351fd9c2983f970","source":"docs/dev/target-evidence.md","level":"native-config-parsing"},"capabilities":[{"field":"route.authentication","status":"blocking","reason":"authentication identity is not verified"}],"diagnostics":[{"severity":"error","code":"codex.route.authentication_unverified","path":"route.authentication","message":"exact Codex authentication route is unverified"}],"artifacts":[{"path":"preview/route-only.config.toml.preview","kind":"candidate-config","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":10}]}`},
		"render-oh-my-pi":          {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"oh-my-pi","targetVersion":"18.4.6","adapterVersion":"profilemango.dev/oh-my-pi/v1alpha1","applicable":false,"preview":true,"evidence":{"target":"oh-my-pi","version":"18.4.6","sha256":"bca974df221fecc1e706c642e1329a761849e46f76554d3df587f3629c8864f4","source":"docs/dev/target-evidence.md","level":"source-entrypoint-version"},"capabilities":[{"field":"target.artifact","status":"blocking","reason":"compiled Oh My Pi artifact is unavailable"}],"diagnostics":[{"severity":"error","code":"ohmypi.config.inspector_unsafe","path":"target.config-inspection","message":"config inspection is blocked"}],"artifacts":[{"path":"preview/route-only.config.yml.preview","kind":"candidate-config","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":10}]}`},
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
		"profile missing metadata":          {schema: "profile.schema.json", document: `{"apiVersion":"profilemango.dev/v1alpha1","kind":"PolicyProfile","spec":{"routeRef":"codex-oauth"}}`},
		"profile mixed wrapper":             {schema: "profile.schema.json", document: `{"kind":"PolicyProfile","route":"codex-oauth"}`},
		"profile old route key":             {schema: "profile.schema.json", document: `{"routeRef":"codex-oauth"}`},
		"profile invalid name":              {schema: "profile.schema.json", document: `{"name":"Default","route":"codex-oauth"}`},
		"bindings missing model":            {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","effort":"high"}}}`},
		"bindings empty auth":               {schema: "bindings.schema.json", document: `{"routes":{"codex-oauth":{"provider":"openai","transport":"native","authentication":"","model":"gpt-5.6","effort":"high"}}}`},
		"bindings unknown target":           {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"claude":{"provider":"anthropic"}}}}}`},
		"bindings empty override":           {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{}}}}}`},
		"bindings override field":           {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{"token":"x"}}}}}`},
		"bindings empty roles":              {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{}}}}`},
		"bindings default role":             {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"default":{"provider":"openai","model":"m"}}}}}`},
		"bindings role missing model":       {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"research":{"provider":"openai"}}}}}`},
		"bindings role unknown field":       {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"research":{"provider":"openai","model":"m","transport":"native"}}}}}`},
		"bindings role invalid name":        {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"Smol":{"provider":"openai","model":"m"}}}}}`},
		"bindings role dotted name":         {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"bad.name":{"provider":"openai","model":"m"}}}}}`},
		"bindings unknown subagent cap":     {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","subagentMaxEffort":"ultra"}}}`},
		"bindings empty target role":        {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"research":{"provider":"openai","model":"m"}},"targets":{"codex":{"roles":{"research":{}}}}}}}`},
		"bindings target role field":        {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"research":{"provider":"openai","model":"m"}},"targets":{"codex":{"roles":{"research":{"transport":"native"}}}}}}}`},
		"bindings target role default":      {schema: "bindings.schema.json", document: `{"routes":{"main":{"provider":"openai","model":"gpt-5.6","effort":"high","roles":{"research":{"provider":"openai","model":"m"}},"targets":{"codex":{"roles":{"default":{"model":"m"}}}}}}}`},
		"profile role no description":       {schema: "profile.schema.json", document: `{"roles":{"worker":{"instructions":"roles/worker.md"}}}`},
		"profile invalid role":              {schema: "profile.schema.json", document: `{"roles":{"Bad_Name":{"description":"x"}}}`},
		"profile role unknown field":        {schema: "profile.schema.json", document: `{"roles":{"worker":{"description":"x","model":"gpt"}}}`},
		"profile agent file unknown target": {schema: "profile.schema.json", document: `{"agentFiles":{"claude":{"scout.md":"a.md"}}}`},
		"profile agent file nested name":    {schema: "profile.schema.json", document: `{"agentFiles":{"oh-my-pi":{"sub/scout.md":"a.md"}}}`},
		"profile agent file extension":      {schema: "profile.schema.json", document: `{"agentFiles":{"oh-my-pi":{"scout.txt":"a.md"}}}`},
		"profile global empty list":         {schema: "profile.schema.json", document: `{"globalInstructions":{"codex":{"AGENTS.md":[]}}}`},
		"profile global empty fragment":     {schema: "profile.schema.json", document: `{"globalInstructions":{"codex":{"AGENTS.md":["a.md",""]}}}`},
		"plan route with targets":           {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high","targets":{"codex":{"model":"x"}}},"resources":[]}`},
		"plan malformed digest":             {schema: "plan.schema.json", document: `{"apiVersion":"profilemango.dev/plan/v1alpha1","kind":"Plan","profile":"route-only","target":"codex","route":{"provider":"openai","transport":"native","authentication":"oauth","model":"gpt-5.6","effort":"high"},"resources":[{"path":"instructions/system.md","kind":"instruction","sha256":"short","size":12}]}`},
		"manifest negative generation":      {schema: "manifest.schema.json", document: `{"apiVersion":"profilemango.dev/manifest/v1alpha1","kind":"Manifest","owner":"profile-mango","generation":-1,"profile":"route-only","target":"codex","resources":[]}`},
		"render missing evidence":           {schema: "render.schema.json", document: `{"apiVersion":"profilemango.dev/render/v1alpha1","kind":"RenderReport","profile":"route-only","target":"codex","targetVersion":"0.157.1","adapterVersion":"profilemango.dev/codex/v1alpha1","applicable":false,"preview":true,"evidence":{},"capabilities":[],"diagnostics":[],"artifacts":[]}`},
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

// TestSchemaRoleVocabularyMatchesDomain covers every input and output role contract.
func TestSchemaRoleVocabularyMatchesDomain(t *testing.T) {
	t.Parallel()
	contracts := map[string]struct {
		file string
		path []string
	}{
		"profile":  {"profile.schema.json", []string{"$defs", "roles", "propertyNames"}},
		"bindings": {"bindings.schema.json", []string{"$defs", "roles", "propertyNames"}},
		"target":   {"bindings.schema.json", []string{"$defs", "targetOverride", "properties", "roles", "propertyNames"}},
		"skipped":  {"install-plan.schema.json", []string{"$defs", "skippedRequirement", "properties", "role"}},
	}
	for name, contract := range contracts {
		t.Run(name, func(t *testing.T) {
			checkRoleSchemaContract(t, schemaNode(t, contract.file, contract.path...))
		})
	}
	efforts := schemaNode(t, "bindings.schema.json", "$defs", "subagentMaxEffort", "enum").([]any)
	want := make([]any, len(profilemango.SubagentEfforts))
	for index, effort := range profilemango.SubagentEfforts {
		want[index] = effort
	}
	if !reflect.DeepEqual(efforts, want) {
		t.Fatalf("schema subagentMaxEffort %v, domain %v", efforts, want)
	}
}

func schemaNode(t *testing.T, file string, path ...string) any {
	t.Helper()
	node := loadDocument(t, file, false)
	for _, key := range path {
		node = node.(map[string]any)[key]
	}
	return node
}

func checkRoleSchemaContract(t *testing.T, contract any) {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaBaseURL+"role", contract); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaBaseURL + "role")
	if err != nil {
		t.Fatal(err)
	}
	for name := range map[string]struct{}{
		"worker": {}, "planner": {}, "research": {}, "tiny": {},
		"coder": {}, "reviewer": {}, "code": {}, "review": {}, "smol": {},
		"a": {}, "a-1": {}, "default": {}, "": {}, "A": {}, "1role": {},
		"bad_name": {}, "bad.name": {}, "../escape": {}, "role\n": {},
		strings.Repeat("a", 63): {}, strings.Repeat("a", 64): {},
	} {
		if accepted := schema.Validate(name) == nil; accepted != profilemango.ValidRoleName(name) {
			t.Errorf("schema accepts %q = %t, domain = %t", name, accepted, profilemango.ValidRoleName(name))
		}
	}
}
