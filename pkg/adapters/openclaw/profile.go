package openclaw

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Pinned OpenClaw 2026.9.5 (ec9c1a13) profile layout: src/cli/profile-utils.ts
// resolves `--profile <name>` to the state directory <home>/.openclaw-<name>, and
// src/cli/profile.ts points OPENCLAW_CONFIG_PATH at openclaw.json inside it.
const (
	defaultStateDirName = ".openclaw"
	configFileName      = "openclaw.json"
)

// profileNamePattern mirrors PROFILE_NAME_RE in src/cli/profile-utils.ts.
var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidProfileName reports whether `openclaw --profile <name>` accepts name.
func ValidProfileName(name string) bool { return profileNamePattern.MatchString(name) }

// IsDefaultProfile reports whether name selects OpenClaw's default state directory:
// resolveProfileStateDir maps "default" (any case) to <home>/.openclaw.
func IsDefaultProfile(name string) bool { return strings.EqualFold(name, "default") }

// ProfileConfigPath returns the config file `openclaw --profile <name>` reads. The
// "default" profile is the default config itself, including a relocated one, because
// applyCliProfileEnv keeps an existing OPENCLAW_CONFIG_PATH when the profile does not
// change. Other names derive <home>/.openclaw-<name>/openclaw.json from a config at
// <home>/.openclaw/openclaw.json; other config paths are refused because the profile
// location depends only on OpenClaw's home, not on a relocated config.
func ProfileConfigPath(configPath, name string) (string, error) {
	if !ValidProfileName(name) {
		return "", fmt.Errorf("openclaw profile names must start with a letter or digit, use only letters, digits, '_' or '-', and be at most 64 characters")
	}
	clean := filepath.Clean(configPath)
	if IsDefaultProfile(name) {
		return clean, nil
	}
	stateDir := filepath.Dir(clean)
	if !filepath.IsAbs(clean) || filepath.Base(clean) != configFileName || filepath.Base(stateDir) != defaultStateDirName {
		return "", fmt.Errorf("openclaw --profile reads <home>/.openclaw-<name>/openclaw.json, which can only be derived from a config at <home>/.openclaw/openclaw.json, not %s", configPath)
	}
	return filepath.Join(filepath.Dir(stateDir), defaultStateDirName+"-"+name, configFileName), nil
}
