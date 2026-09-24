package hermes

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Pinned Hermes Agent v0.21.3 (source release v2026.9.14, commit 345cd2b0) profile
// layout. hermes_cli/profiles.py:132-135 _get_profiles_root and :232-244 get_profile_dir
// resolve a named profile to <hermes-home>/profiles/<name>/config.yaml, where
// <hermes-home> is the directory holding the main config.yaml (default ~/.hermes;
// hermes_cli/profiles.py:138-142 _get_default_hermes_home). hermes_cli/main.py:508-556
// _apply_profile_override pre-parses `-p`/`--profile <name>` and points HERMES_HOME at
// that directory via hermes_cli/profiles.py:1789-1813 resolve_profile_env, which only
// requires the directory to exist (profile_dir.is_dir()) and not be tombstoned; no
// separate marker file is required, so writing config.yaml there (which creates the
// parent directory) is sufficient to make the profile selectable.
const (
	profilesDirName       = "profiles"
	profileConfigFileName = "config.yaml"
)

// profileNamePattern mirrors hermes_cli/profiles.py:24 _PROFILE_ID_RE.
var profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// reservedProfileNames mirrors hermes_cli/profiles.py:120 _RESERVED_NAMES, minus
// "default": hermes_cli/profiles.py:189-206 validate_profile_name special-cases
// "default" to pass instead of rejecting it, and IsDefaultProfile below handles it.
var reservedProfileNames = map[string]bool{"hermes": true, "test": true, "tmp": true, "root": true, "sudo": true}

// ValidProfileName reports whether `hermes -p <name>` accepts name as a non-default
// profile id: hermes_cli/profiles.py:189-206 validate_profile_name.
func ValidProfileName(name string) bool {
	return profileNamePattern.MatchString(name) && !reservedProfileNames[name]
}

// IsDefaultProfile reports whether name selects Hermes' base HERMES_HOME instead of a
// profiles/<name> directory. hermes_cli/profiles.py:172-186 normalize_profile_name
// case-folds "default" before dispatch, and :232-236 get_profile_dir / :1808-1809
// resolve_profile_env return the root unchanged for it.
func IsDefaultProfile(name string) bool { return strings.EqualFold(name, "default") }

// ProfileConfigPath returns the config.yaml file `hermes -p <name>` reads: the resolved
// main config itself for "default" (hermes_cli/profiles.py:1808-1809), or
// <config-dir>/profiles/<name>/config.yaml for any other valid id, derived from
// whatever directory holds the resolved main config.yaml. Only that directory needs to
// exist for `-p <name>` to resolve (hermes_cli/profiles.py:1810-1812), so an install that
// writes config.yaml there is enough; no `hermes profile create` bootstrap is required.
func ProfileConfigPath(configPath, name string) (string, error) {
	clean := filepath.Clean(configPath)
	if IsDefaultProfile(name) {
		return clean, nil
	}
	if !ValidProfileName(name) {
		return "", fmt.Errorf("hermes profile names must match [a-z0-9][a-z0-9_-]{0,63} and not be a reserved name (hermes, test, tmp, root, sudo)")
	}
	return filepath.Join(filepath.Dir(clean), profilesDirName, name, profileConfigFileName), nil
}
