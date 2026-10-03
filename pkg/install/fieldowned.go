package install

import (
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
)

// writtenSHA256Prefix marks, on a live-patched file's manifest entry, the SHA-256 of
// the value profile-mango wrote to one owned field: written-sha256:<field>=<hex>.
// Hashes keep sensitive values out of the manifest.
const writtenSHA256Prefix = "written-sha256:"

// fieldsByPath indexes planned field changes by path, keeping the first of duplicates.
func fieldsByPath(fields []FieldChange) map[string]FieldChange {
	result := make(map[string]FieldChange, len(fields))
	for _, field := range fields {
		if _, found := result[field.Path]; !found {
			result[field.Path] = field
		}
	}
	return result
}

// writtenMarkers records the value this patch writes to each owned field of a
// live-patched file. A field without a planned value leaves the file unrecorded, so
// later plans fall back to the whole-file hash.
func writtenMarkers(file FilePatch, fields map[string]FieldChange) []string {
	if !file.LiveFields || file.Delete || file.Release || len(file.Fields) == 0 {
		return nil
	}
	markers := make([]string, 0, len(file.Fields))
	for _, name := range file.Fields {
		field, found := fields[name]
		if !found {
			return nil
		}
		markers = append(markers, writtenSHA256Prefix+name+"="+installfs.Hash([]byte(field.After)))
	}
	return markers
}

// ownedFieldsIntact reports whether every owned field of a live-patched file still
// holds the value profile-mango recorded writing, so other bytes (a re-serialization
// or unrelated keys) changed but no owned value did. Entries without written markers,
// such as manifests from earlier versions, never qualify.
func ownedFieldsIntact(file FilePatch, entry ManifestFile, fields map[string]FieldChange) bool {
	if !file.LiveFields {
		return false
	}
	recorded := 0
	for _, field := range entry.Fields {
		marker, isMarker := strings.CutPrefix(field, writtenSHA256Prefix)
		name, hash, found := strings.Cut(marker, "=")
		if !isMarker || !found {
			continue
		}
		live, read := fields[name]
		if !read || installfs.Hash([]byte(live.Before)) != hash {
			return false
		}
		recorded++
	}
	return recorded > 0
}

// ownedBaseline is the hash the live file must have to count as unedited: the
// recorded hash, or the live hash itself when only unowned bytes of a live-patched
// file changed.
func ownedBaseline(file FilePatch, ownership Manifest, path string, before installfs.Snapshot, fields map[string]FieldChange) (string, bool) {
	entry, owned := manifestEntry(ownership, path)
	if !owned || !before.Exists || before.SHA256 == entry.SHA256 {
		return entry.SHA256, owned
	}
	if ownedFieldsIntact(file, entry, fields) || ohMyPiPresetFieldsIntact(file, entry, fields) {
		return before.SHA256, true
	}
	return entry.SHA256, true
}

// fieldPriors reads the recorded pre-install value of every owned key whose
// manifest marker starts with prefix.
func fieldPriors(ownership Manifest, configPath, prefix string) map[string]string {
	priors := map[string]string{}
	for _, file := range ownership.Files {
		if file.Path != configPath {
			continue
		}
		for _, field := range file.Fields {
			marker, found := strings.CutPrefix(field, prefix)
			if key, prior, ok := strings.Cut(marker, "="); found && ok {
				priors[key] = prior
			}
		}
	}
	return priors
}

// fieldPriorMarker records a key's pre-install value, keeping the first recorded
// value while the key stays owned.
func fieldPriorMarker(prefix string, priors map[string]string, key, before string) string {
	prior, recorded := priors[key]
	if !recorded {
		prior = before
	}
	return prefix + key + "=" + prior
}
