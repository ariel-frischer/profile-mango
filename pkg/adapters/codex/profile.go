package codex

import (
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
)

// profileFileSuffix names the Codex 0.157.1 profile layer that `codex --profile <name>`
// reads from $CODEX_HOME/<name>.config.toml on top of config.toml.
const profileFileSuffix = ".config.toml"

var errLegacyProfileSelector = errors.New("codex config contains a legacy profile = setting, which Codex 0.157.1 refuses to start with; remove it before installing")

// ProfileFileName returns the profile file name, relative to the Codex home.
func ProfileFileName(name string) string { return name + profileFileSuffix }

// ValidProfileName mirrors the Codex 0.157.1 --profile name rule: ASCII letters, digits, '_' and '-'.
func ValidProfileName(name string) bool {
	if name == "" {
		return false
	}
	for _, character := range []byte(name) {
		letter := (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z')
		digit := character >= '0' && character <= '9'
		if !letter && !digit && character != '_' && character != '-' {
			return false
		}
	}
	return true
}

// CheckBaseConfig rejects config.toml state that stops `codex --profile <name>` from
// starting or shadows the built-in openai provider the profile selects. It never changes source.
func CheckBaseConfig(source []byte, name string) error {
	if !ValidProfileName(name) {
		return fmt.Errorf("codex profile name must use only letters, digits, '_' or '-'")
	}
	if len(source) > maxConfigBytes {
		return fmt.Errorf("codex config exceeds %d-byte limit", maxConfigBytes)
	}
	if err := validateConfigBytes(source); err != nil {
		return err
	}
	if err := validateTOML(source); err != nil {
		return err
	}
	var decoded map[string]any
	if err := toml.Unmarshal(source, &decoded); err != nil {
		return fmt.Errorf("invalid codex TOML; correct syntax before installing")
	}
	return checkBaseValues(decoded, name)
}

func checkBaseValues(config map[string]any, name string) error {
	if _, found := config["profile"]; found {
		return errLegacyProfileSelector
	}
	if profiles, ok := config["profiles"].(map[string]any); ok {
		if _, found := profiles[name]; found {
			return fmt.Errorf("codex config contains a legacy [profiles.%s] table, which Codex 0.157.1 refuses to combine with --profile %s; move or remove it before installing", name, name)
		}
	}
	providers, _ := config["model_providers"].(map[string]any)
	if _, found := providers["openai"]; found {
		return fmt.Errorf("codex config contains provider override state that may shadow the built-in openai provider")
	}
	if _, found := config["openai_base_url"]; found {
		return fmt.Errorf("codex config contains provider override state that may shadow the built-in openai provider")
	}
	return nil
}
