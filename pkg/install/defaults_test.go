package install

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
)

// syntheticPathEnv never consults the process environment or the real user home.
func syntheticPathEnv(t *testing.T, home string, vars map[string]string) PathEnv {
	t.Helper()
	return PathEnv{
		Getenv:   func(name string) string { return vars[name] },
		UserHome: func() (string, error) { return home, nil },
		Exists: func(path string) bool {
			if !strings.HasPrefix(path, home) && !strings.HasPrefix(path, filepath.Dir(home)) {
				t.Fatalf("default path probe escaped synthetic home: %s", path)
			}
			_, err := os.Lstat(path)
			return err == nil
		},
	}
}

func TestDefaultConfigPathPerTarget(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	tests := map[string]struct {
		adapter defaultConfigLocator
		vars    map[string]string
		want    string
	}{
		"claude-code":         {adapter: claudeCodeAdapter{}, want: filepath.Join(home, ".claude", "settings.json")},
		"codex":               {adapter: codexAdapter{}, want: filepath.Join(home, ".codex", "config.toml")},
		"codex CODEX_HOME":    {adapter: codexAdapter{}, vars: map[string]string{"CODEX_HOME": filepath.Join(root, "cx")}, want: filepath.Join(root, "cx", "config.toml")},
		"hermes":              {adapter: hermesAdapter{}, want: filepath.Join(home, ".hermes", "config.yaml")},
		"hermes HERMES_HOME":  {adapter: hermesAdapter{}, vars: map[string]string{"HERMES_HOME": filepath.Join(root, "hx")}, want: filepath.Join(root, "hx", "config.yaml")},
		"oh-my-pi":            {adapter: ohMyPiAdapter{}, want: filepath.Join(home, ".omp", "agent", "config.yml")},
		"openclaw":            {adapter: openClawAdapter{}, want: filepath.Join(home, ".openclaw", "openclaw.json")},
		"openclaw relocated":  {adapter: openClawAdapter{}, vars: map[string]string{"OPENCLAW_CONFIG_PATH": filepath.Join(root, "oc.json")}, want: filepath.Join(root, "oc.json")},
		"pi":                  {adapter: piAdapter{}, want: filepath.Join(home, ".pi", "agent", "settings.json")},
		"pi relocated":        {adapter: piAdapter{}, vars: map[string]string{"PI_CODING_AGENT_DIR": filepath.Join(root, "pi")}, want: filepath.Join(root, "pi", "settings.json")},
		"opencode":            {adapter: openCodeAdapter{}, want: filepath.Join(home, ".config", "opencode", "opencode.json")},
		"opencode XDG":        {adapter: openCodeAdapter{}, vars: map[string]string{"XDG_CONFIG_HOME": filepath.Join(root, "xdg")}, want: filepath.Join(root, "xdg", "opencode", "opencode.json")},
		"opencode jsonc only": {adapter: openCodeAdapter{}, vars: map[string]string{"XDG_CONFIG_HOME": filepath.Join(root, "jsonc")}, want: filepath.Join(root, "jsonc", "opencode", "opencode.jsonc")},
	}
	if err := os.MkdirAll(filepath.Join(root, "jsonc", "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(root, "jsonc", "opencode", "opencode.jsonc"), "{}\n")
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := test.adapter.DefaultConfigPath(syntheticPathEnv(t, home, test.vars))
			if err != nil || got != test.want {
				t.Fatalf("DefaultConfigPath = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestDefaultConfigPathBlockedReasons(t *testing.T) {
	root := t.TempDir()
	both := filepath.Join(root, "both", "opencode")
	if err := os.MkdirAll(both, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, filepath.Join(both, "opencode.json"), "{}\n")
	writeInstallTestFile(t, filepath.Join(both, "opencode.jsonc"), "{}\n")
	codex := Target{Name: "codex", Version: "0.157.1"}
	tests := map[string]struct {
		adapter Adapter
		target  Target
		env     PathEnv
		want    string
	}{
		"no documented default":  {adapter: testAdapterFor("fake", "1", "x", false), target: Target{Name: "fake", Version: "1"}, env: syntheticPathEnv(t, root, nil), want: "no documented default config path for this target; --config fake=<path> is required"},
		"resolution disabled":    {adapter: codexAdapter{}, target: codex, env: PathEnv{}, want: "default config path resolution is unavailable; --config codex=<path> is required"},
		"relative relocation":    {adapter: codexAdapter{}, target: codex, env: syntheticPathEnv(t, root, map[string]string{"CODEX_HOME": "rel"}), want: "CODEX_HOME must be an absolute path"},
		"relative home":          {adapter: codexAdapter{}, target: codex, env: PathEnv{Getenv: func(string) string { return "" }, UserHome: func() (string, error) { return "rel", nil }}, want: "not an absolute path"},
		"opencode json and json": {adapter: openCodeAdapter{}, target: Target{Name: "opencode", Version: "1.18.31"}, env: syntheticPathEnv(t, root, map[string]string{"XDG_CONFIG_HOME": filepath.Join(root, "both")}), want: "both opencode.json and opencode.jsonc exist"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path, reason := resolveDefaultConfigPath(test.adapter, TargetRequest{Target: test.target}, test.env)
			if path != "" || !strings.Contains(reason, test.want) {
				t.Fatalf("path=%q reason=%q, want %q", path, reason, test.want)
			}
		})
	}
}

func TestOSPathEnvUsesProcessEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	got, err := (codexAdapter{}).DefaultConfigPath(OSPathEnv())
	if err != nil || got != filepath.Join(home, ".codex", "config.toml") {
		t.Fatalf("OS default = %q, %v", got, err)
	}
}

func TestDefaultPathPlanApplyPreservesCredentialsAndDetectsDrift(t *testing.T) {
	request, root := claudeCodeTestRequest(t)
	home := filepath.Join(root, "home")
	config := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	before := "{\n  \"model\": \"old/model\",\n  \"apiKeyHelper\": \"/synthetic/key-helper\",\n  \"env\": {\"ANTHROPIC_API_KEY\": \"SYNTHETIC-CREDENTIAL\"},\n  \"permissions\": {\"allow\": [\"Bash(ls)\"]}\n}\n"
	writeInstallTestFile(t, config, before)
	request.Override = true
	request.Env = syntheticPathEnv(t, home, nil)
	request.Targets = []TargetRequest{{Target: Target{Name: claudecode.TargetName, Version: claudecode.TargetVersion}}}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	target := plan.Targets[0]
	if target.Config == nil || target.Config.Path != config || target.Config.Source != ConfigSourceDefault || target.ConfigPath != config {
		t.Fatalf("config destination = %#v / %q", target.Config, target.ConfigPath)
	}
	data, err := plan.JSON()
	if err != nil || strings.Contains(string(data), root) || !strings.Contains(string(data), `"source": "default"`) {
		t.Fatalf("plan JSON = %s, err=%v", data, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, config, strings.Replace(before, `"model": "old/model"`, "\"effortLevel\": \"high\",\n  \"model\": \"claude-sonnet-4-5\"", 1))
	assertInstallTestFile(t, installedBackup(t, plan.PlanID, config), before)
	if _, err := os.Stat(config + ".profile-mango.manifest.json"); err != nil {
		t.Fatalf("ownership manifest missing: %v", err)
	}
	request.Env = syntheticPathEnv(t, home, nil)
	writeInstallTestFile(t, config, before)
	drift, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	writeInstallTestFile(t, config, before+" ")
	if _, err := ApplyPlan(drift, ApplyOptions{ExpectedPlanID: drift.PlanID}); !errors.Is(err, installfs.ErrStale) {
		t.Fatalf("drifted apply error = %v", err)
	}
	assertInstallTestFile(t, config, before+" ")
}

func TestExplicitConfigOverridesDefault(t *testing.T) {
	request, root := claudeCodeTestRequest(t)
	explicit := request.Targets[0].ConfigPath
	writeInstallTestFile(t, explicit, "{\"model\": \"claude-sonnet-4-5\"}\n")
	home := filepath.Join(root, "home")
	request.Env = syntheticPathEnv(t, home, nil)
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	target := plan.Targets[0]
	if target.Config == nil || target.Config.Path != explicit || target.Config.Source != ConfigSourceExplicit {
		t.Fatalf("config destination = %#v", target.Config)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("explicit plan touched default home: %v", err)
	}
}

func TestDefaultPathCodexPreservesAuthSiblings(t *testing.T) {
	request, root := codexTestRequest(t)
	request.Registry = DefaultRegistry()
	request.Override, request.Default = true, false
	codexHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(codexHome, "config.toml")
	auth := filepath.Join(codexHome, "auth.json")
	before := "# keep\nunknown = true\n[features]\napps = false\n"
	writeInstallTestFile(t, config, before)
	writeInstallTestFile(t, auth, "synthetic credential sentinel")
	request.Env = syntheticPathEnv(t, filepath.Join(root, "home"), map[string]string{"CODEX_HOME": codexHome})
	request.Targets = []TargetRequest{{Target: Target{Name: "codex", Version: "0.157.1"}}}
	plan, err := BuildPlan(request)
	if err != nil || plan.Status != StatusReady || plan.Targets[0].Config.Path != config {
		t.Fatalf("plan = %#v, err=%v", plan, err)
	}
	if _, err := ApplyPlan(plan, ApplyOptions{ExpectedPlanID: plan.PlanID}); err != nil {
		t.Fatal(err)
	}
	assertInstallTestFile(t, config, before)
	assertInstallTestFile(t, filepath.Join(codexHome, "route-only.config.toml"), "model_provider = \"openai\"\nmodel = \"gpt-5.6\"\nmodel_reasoning_effort = \"high\"\n")
	assertInstallTestFile(t, auth, "synthetic credential sentinel")
}
