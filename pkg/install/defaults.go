package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigSourceDefault  = "default"
	ConfigSourceExplicit = "explicit"
)

// ConfigDestination reports the resolved main config path and whether it came from --config or the target default.
// Path stays out of plan JSON, which carries no raw config paths (only a named profile's
// use command may name its file); human plans display it for consent.
type ConfigDestination struct {
	Path   string `json:"-"`
	Source string `json:"source"`
}

// PathEnv supplies the environment used to resolve documented default config paths.
// A zero PathEnv resolves no defaults, so callers must opt in explicitly.
type PathEnv struct {
	Getenv   func(string) string
	UserHome func() (string, error)
	Exists   func(string) bool
}

// OSPathEnv resolves defaults from the process environment and user home.
func OSPathEnv() PathEnv {
	return PathEnv{
		Getenv:   os.Getenv,
		UserHome: os.UserHomeDir,
		Exists: func(path string) bool {
			_, err := os.Lstat(path)
			return err == nil
		},
	}
}

func (env PathEnv) enabled() bool { return env.Getenv != nil && env.UserHome != nil }

func (env PathEnv) home() (string, error) {
	home, err := env.UserHome()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if strings.TrimSpace(home) == "" || !filepath.IsAbs(home) {
		return "", fmt.Errorf("resolve user home: home %q is not an absolute path", home)
	}
	return home, nil
}

// dirOrHome returns an absolute relocation variable value, or home joined with fallback.
func (env PathEnv) dirOrHome(variable string, fallback ...string) (string, error) {
	if value := strings.TrimSpace(env.Getenv(variable)); value != "" {
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("%s must be an absolute path", variable)
		}
		return filepath.Clean(value), nil
	}
	home, err := env.home()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{home}, fallback...)...), nil
}

func (env PathEnv) exists(path string) bool { return env.Exists != nil && env.Exists(path) }

// defaultConfigLocator is implemented by adapters whose docs/dev/agents page documents a user-level config location.
type defaultConfigLocator interface {
	DefaultConfigPath(PathEnv) (string, error)
}

// resolveDefaultConfigPath returns the documented default path, or a blocked reason when none applies.
func resolveDefaultConfigPath(adapter Adapter, target TargetRequest, env PathEnv) (string, string) {
	explicitOnly := fmt.Sprintf("--config %s=<path> is required", target.Target.Name)
	if !target.Agent.Empty() {
		return "", "named agent definitions require an explicit " + explicitOnly
	}
	locator, found := adapter.(defaultConfigLocator)
	if !found {
		return "", "no documented default config path for this target; " + explicitOnly
	}
	if !env.enabled() {
		return "", "default config path resolution is unavailable; " + explicitOnly
	}
	path, err := locator.DefaultConfigPath(env)
	if err != nil {
		return "", fmt.Sprintf("resolve default config path: %v; %s", err, explicitOnly)
	}
	return path, ""
}

// DefaultConfigPath resolves the documented default config path for a registered target.
func (registry *Registry) DefaultConfigPath(target Target, env PathEnv) (string, error) {
	adapter, found := registry.Lookup(target)
	if !found {
		return "", fmt.Errorf("no static adapter is registered for %s", target.String())
	}
	path, reason := resolveDefaultConfigPath(adapter, TargetRequest{Target: target}, env)
	if reason != "" {
		return "", fmt.Errorf("%s", reason)
	}
	return path, nil
}

// Documented user-level locations; see docs/dev/agents/<target>.md.

func (claudeCodeAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	home, err := env.home()
	return filepath.Join(home, ".claude", "settings.json"), err
}

func (codexAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	dir, err := env.dirOrHome("CODEX_HOME", ".codex")
	return filepath.Join(dir, "config.toml"), err
}

func (hermesAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	dir, err := env.dirOrHome("HERMES_HOME", ".hermes")
	return filepath.Join(dir, "config.yaml"), err
}

func (ohMyPiAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	home, err := env.home()
	return filepath.Join(home, ".omp", "agent", "config.yml"), err
}

func (openClawAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	if value := strings.TrimSpace(env.Getenv("OPENCLAW_CONFIG_PATH")); value != "" {
		if !filepath.IsAbs(value) {
			return "", fmt.Errorf("OPENCLAW_CONFIG_PATH must be an absolute path")
		}
		return filepath.Clean(value), nil
	}
	home, err := env.home()
	return filepath.Join(home, ".openclaw", "openclaw.json"), err
}

func (piAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	dir, err := env.dirOrHome("PI_CODING_AGENT_DIR", ".pi", "agent")
	return filepath.Join(dir, "settings.json"), err
}

// DefaultConfigPath prefers opencode.json, uses opencode.jsonc when only it exists, and refuses to guess when both exist.
func (openCodeAdapter) DefaultConfigPath(env PathEnv) (string, error) {
	dir, err := env.dirOrHome("XDG_CONFIG_HOME", ".config")
	if err != nil {
		return "", err
	}
	jsonPath := filepath.Join(dir, "opencode", "opencode.json")
	jsoncPath := filepath.Join(dir, "opencode", "opencode.jsonc")
	switch hasJSON, hasJSONC := env.exists(jsonPath), env.exists(jsoncPath); {
	case hasJSON && hasJSONC:
		return "", fmt.Errorf("both opencode.json and opencode.jsonc exist in %s", filepath.Dir(jsonPath))
	case hasJSONC:
		return jsoncPath, nil
	default:
		return jsonPath, nil
	}
}

const noAgentsFound = "no supported agents found; run mango doctor"

// missingAgentFolder handles a default config path whose folder does not exist. With
// SkipNotInstalled (install --all) the target is skipped whether or not its command is
// on PATH, so the rest of the plan stays ready; an explicit target is blocked with the
// fix. It reports false when the folder exists or cannot be classified, leaving other
// failures to the normal path inspection.
func missingAgentFolder(request Request, targetPlan TargetPlan) (TargetPlan, bool) {
	dir := filepath.Dir(targetPlan.ConfigPath)
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		return targetPlan, false
	}
	if request.SkipNotInstalled {
		targetPlan.Status = StatusSkipped
		targetPlan.Reason = "config folder " + tildePath(request.Env, dir) + " not found"
		return targetPlan, true
	}
	name := targetPlan.Target.Name
	reason := fmt.Sprintf("%s config folder %s not found — is %s installed? Pass --config %s=<path> to choose a file.", name, dir, name, name)
	return blockedTargetPlan(targetPlan, reason, "install.config_folder_missing"), true
}

// tildePath shows a path beneath the user home as ~/relative so plan output and JSON
// stay free of absolute home paths; other paths are returned unchanged.
func tildePath(env PathEnv, path string) string {
	if !env.enabled() {
		return path
	}
	home, err := env.home()
	if err != nil {
		return path
	}
	relative, err := filepath.Rel(home, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, "../") {
		return path
	}
	return "~/" + filepath.ToSlash(relative)
}

func allSkipped(targets []TargetPlan) bool {
	for _, target := range targets {
		if target.Status != StatusSkipped {
			return false
		}
	}
	return len(targets) > 0
}
