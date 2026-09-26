package ohmypi

// profileFileDir is where install writes the Mango-owned named profile overlay,
// below the config.yml directory. Oh My Pi 18.3.2 loads it with
// `omp --config <path>` after global and project config (settings.ts
// #rebuildMerged, #mergeOwnLayers); native `--profile` relocates auth and sessions and is not used.
const profileFileDir = "profiles"

// ProfileFileName returns the named profile overlay path, relative to the
// config.yml directory. The pinned overlay loader parses any path as YAML
// (settings.ts #loadOverlayYaml); .yml matches config.yml.
func ProfileFileName(name string) string { return profileFileDir + "/" + name + ".yml" }

// ValidProfileName restricts profile names to ASCII letters, digits, '_' and '-',
// the same safe subset the other named-profile adapters use.
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
