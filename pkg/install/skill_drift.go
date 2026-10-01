package install

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// Which side of a differing skill folder was modified last.
const (
	SkillNewerProfile = "profile"
	SkillNewerGlobal  = "global"
	SkillNewerUnknown = "unknown"
)

// SkillDrift is one profile-owned skill whose folder in the global skills directory
// differs from the profile's copy, which is canonical. Files lists the differing paths
// relative to the skill folder; Newer names the side holding the newest file.
type SkillDrift struct {
	Profile  string   `json:"profile"`
	Skill    string   `json:"skill"`
	Snapshot string   `json:"snapshot"`
	Global   string   `json:"global"`
	Files    []string `json:"files"`
	Newer    string   `json:"newer"`
	Hint     string   `json:"hint"`
}

// globalSkillDrift compares the skills of every profile that is a managed agent's
// default with their folders in the global skills directory. A global file matches
// when it equals the profile's bytes or their rendering for any registered target, so
// rendered {{route.…}} placeholders are not drift. Absent global folders are not drift.
// It only reads; profile-mango never copies a global skill into the profile.
func globalSkillDrift(request StatusRequest, registry *Registry, targets []TargetStatus) []SkillDrift {
	if request.ProfilesRoot == "" {
		return nil
	}
	root := request.GlobalSkillsRoot
	if root == "" {
		if request.Env.UserHome == nil {
			return nil
		}
		home, err := request.Env.home()
		if err != nil {
			return nil
		}
		root = filepath.Join(home, ".agents", "skills")
	}
	var profiles []string
	for _, target := range targets {
		if target.State == StatusManaged && target.ownsConfig && target.Profile != "" && !slices.Contains(profiles, target.Profile) {
			profiles = append(profiles, target.Profile)
		}
	}
	sort.Strings(profiles)
	var names []string
	for _, target := range registry.Targets() {
		if !slices.Contains(names, target.Name) {
			names = append(names, target.Name)
		}
	}
	var drift []SkillDrift
	for _, profile := range profiles {
		loaded, _, err := loadInput(Request{ProfileName: profile, ProfilesRoot: request.ProfilesRoot, ResourceRoot: request.ResourceRoot, BindingsPath: request.BindingsPath})
		if err != nil {
			continue
		}
		renderings := [][]skillBundle{loaded.Skills}
		for _, name := range names {
			if rendered, err := renderSkills(loaded.Skills, loaded.Bindings, name); err == nil {
				renderings = append(renderings, rendered)
			}
		}
		for index, bundle := range loaded.Skills {
			variants := make([]skillBundle, len(renderings))
			for at, rendering := range renderings {
				variants[at] = rendering[index]
			}
			if entry, found := skillFolderDrift(profile, filepath.Join(root, bundle.Name), variants, request.Env); found {
				drift = append(drift, entry)
			}
		}
	}
	return drift
}

// skillFolderDrift compares one global skill folder with the profile's bundle variants
// (index 0 is the unrendered snapshot).
func skillFolderDrift(profile, folder string, variants []skillBundle, env PathEnv) (SkillDrift, bool) {
	bundle := variants[0]
	live, err := globalSkillFiles(folder)
	if err != nil || live == nil {
		return SkillDrift{}, false
	}
	var differing []string
	var profileNewest, globalNewest int64
	for index, file := range bundle.Files {
		profileNewest = max(profileNewest, file.ModTime)
		current, found := live[file.Path]
		if !found || !matchesVariant(current.content, variants, index) {
			differing = append(differing, file.Path)
		}
	}
	for path, file := range live {
		globalNewest = max(globalNewest, file.modTime)
		if !slices.ContainsFunc(bundle.Files, func(candidate skillBundleFile) bool { return candidate.Path == path }) {
			differing = append(differing, path)
		}
	}
	if len(differing) == 0 {
		return SkillDrift{}, false
	}
	sort.Strings(differing)
	entry := SkillDrift{Profile: profile, Skill: bundle.Name, Snapshot: bundle.Dir, Global: displayPath(folder, env), Files: differing, Newer: SkillNewerUnknown}
	switch {
	case globalNewest > profileNewest:
		entry.Newer = SkillNewerGlobal
		entry.Hint = "the global copy is newer: port its edits into " + bundle.Dir + " (keep {{route.…}} placeholders), then re-apply the profile; do not copy it over the profile, and mango install will not replace it without --override"
	case profileNewest > globalNewest:
		entry.Newer = SkillNewerProfile
		entry.Hint = "the profile copy is newer and canonical: re-apply the profile (mango use " + profile + ") to refresh the global copy; do not copy the global copy over " + bundle.Dir
	default:
		entry.Hint = "both copies have the same modification time: compare them and keep the profile copy canonical"
	}
	return entry, true
}

func matchesVariant(content []byte, variants []skillBundle, index int) bool {
	for _, variant := range variants {
		if bytes.Equal(content, variant.Files[index].Content) {
			return true
		}
	}
	return false
}

type globalSkillFile struct {
	content []byte
	modTime int64
}

// globalSkillFiles reads the files below folder by slash-separated path, following
// symlinks as agents do and skipping profile-mango backups. A missing folder returns nil.
func globalSkillFiles(folder string) (map[string]globalSkillFile, error) {
	resolved, err := filepath.EvalSymlinks(folder)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	files := map[string]globalSkillFile{}
	err = filepath.WalkDir(resolved, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || strings.Contains(entry.Name(), ".profile-mango.") {
			return err
		}
		info, err := os.Stat(current)
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(resolved, current)
		files[filepath.ToSlash(relative)] = globalSkillFile{content: content, modTime: info.ModTime().UnixNano()}
		return nil
	})
	return files, err
}

// displayPath names path as ~/<rel> under the user home, else as is.
func displayPath(path string, env PathEnv) string {
	if env.UserHome != nil {
		if home, err := env.home(); err == nil {
			if relative, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(relative, "..") {
				return "~/" + filepath.ToSlash(relative)
			}
		}
	}
	return path
}
