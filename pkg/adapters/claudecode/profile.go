package claudecode

// profileFileDir is where install writes the Mango-owned named profile file,
// sibling to settings.json. Claude Code has no native named profiles; this is
// a profile-mango convention consumed with `claude --settings <path>`.
const profileFileDir = "profiles"

// ProfileFileName returns the named profile file path, relative to the
// settings.json directory.
func ProfileFileName(name string) string { return profileFileDir + "/" + name + ".json" }

// ValidProfileName restricts profile names to ASCII letters, digits, '_' and '-',
// the same safe subset Codex profile names use.
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
