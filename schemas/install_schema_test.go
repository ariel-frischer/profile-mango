package schemas_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestInstallSchemasAcceptAndRejectBoundedContracts(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		schema  string
		valid   string
		invalid string
	}{
		"plan": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"blocked","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","evidenceSHA256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","installable":false,"status":"blocked","reason":"production target installation is blocked"},"status":"blocked"}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"short","profile":"route-only","status":"blocked","backup":true,"override":false,"inputSHA256":"short","targets":[]}`,
		},
		"plan delete file": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":true,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"opencode","version":"1.18.31"},"metadata":{"target":"opencode","version":"1.18.31","adapterVersion":"profilemango.dev/opencode/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","files":[{"path":"SKILL.md","action":"delete","beforeSHA256":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","owned":true,"delete":true}]}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":true,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"opencode","version":"1.18.31"},"metadata":{"target":"opencode","version":"1.18.31","adapterVersion":"profilemango.dev/opencode/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","files":[{"path":"SKILL.md","action":"delete","owned":true}]}]}`,
		},
		"plan version check": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","compatibleRange":">=0.154.0 <0.155.0","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","versionCheck":{"binary":"codex","qualified":"0.154.0","range":">=0.154.0 <0.155.0","detected":"0.155.1","status":"out-of-range"}}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","compatibleRange":">=0.154.0 <0.155.0","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","versionCheck":{"binary":"codex","qualified":"0.154.0","range":">=0.154.0 <0.155.0","detected":"codex-cli 0.155.1","status":"maybe"}}]}`,
		},
		"plan skipped target": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"hermes","version":"2026.9.1"},"config":{"source":"default"},"metadata":{"target":"hermes","version":"2026.9.1","adapterVersion":"profilemango.dev/hermes/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"skipped","reason":"config folder ~/.hermes not found"}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"route-only","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"hermes","version":"2026.9.1"},"config":{"source":"default"},"metadata":{"target":"hermes","version":"2026.9.1","adapterVersion":"profilemango.dev/hermes/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"missing"}]}`,
		},
		"plan skipped requirements": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"coding","status":"ready","backup":true,"override":false,"strict":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","skippedRequirements":[{"requirement":"permissions"},{"requirement":"instructions","count":2},{"requirement":"roles","count":2,"reason":"per-role routes are only installed for Oh My Pi modelRoles"}]}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"coding","status":"ready","backup":true,"override":false,"strict":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready","skippedRequirements":[{"requirement":"hooks"}]}]}`,
		},
		"plan named profile": {
			schema:  "install-plan.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"coding","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"install":{"mode":"named-profile","profileName":"coding","useCommand":"codex --profile coding","setsDefault":true},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready"}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-plan/v1alpha1","kind":"InstallPlan","planID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","profile":"coding","status":"ready","backup":true,"override":false,"inputSHA256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","targets":[{"target":{"name":"codex","version":"0.154.0"},"install":{"mode":"session"},"metadata":{"target":"codex","version":"0.154.0","adapterVersion":"profilemango.dev/codex/v1alpha1","installable":true,"status":"ready","reason":"synthetic"},"status":"ready"}]}`,
		},
		"manifest": {
			schema:  "install-manifest.schema.json",
			valid:   `{"apiVersion":"profilemango.dev/install-manifest/v1alpha1","kind":"InstallManifest","owner":"profile-mango","generation":1,"profile":"route-only","target":{"name":"fake","version":"1"},"planID":"","files":[{"path":"config","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fields":["config.route"]}]}`,
			invalid: `{"apiVersion":"profilemango.dev/install-manifest/v1alpha1","kind":"InstallManifest","owner":"other","generation":0,"profile":"route-only","target":{"name":"fake","version":"1"},"planID":"","files":[]}`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			schema := compileInstallSchema(t, test.schema)
			if err := schema.Validate(loadDocument(t, test.valid, false)); err != nil {
				t.Fatalf("valid install document rejected: %v", err)
			}
			if err := schema.Validate(loadDocument(t, test.invalid, false)); err == nil {
				t.Fatal("invalid install document accepted")
			}
		})
	}
}

func compileInstallSchema(t *testing.T, filename string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	path := filename
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	url := schemaBaseURL + filename
	if err := compiler.AddResource(url, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		t.Fatalf("compile %s: %v", filename, err)
	}
	return schema
}
