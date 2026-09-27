package install

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ownershipSkill marks a whole file of an installed skill folder.
const ownershipSkill = "skill"

// Skip reasons for skill folders a target does not install.
const (
	skillsAgentReason     = "named agent destinations do not install skill folders"
	skillsNamedOnlyReason = "skill folders apply to every profile; pass --default or run mango use"
)

// maxListedFiles caps how many unmanaged file names one plan warning lists.
const maxListedFiles = 10

// skillBundle is one loaded skill folder: every regular file beneath the folder of a
// skills entry's SKILL.md, with its installed mode.
type skillBundle struct {
	Name string // folder name the agents see
	Dir  string // package-relative folder, e.g. skills/review
	// HasName reports whether the SKILL.md frontmatter sets name.
	HasName bool
	Files   []skillBundleFile
}

// skillBundleFile is one file of a skill folder.
type skillBundleFile struct {
	Path    string // slash-separated, relative to the skill folder
	Mode    fs.FileMode
	Content []byte
}

// skillInstaller is implemented by adapters whose pinned evidence qualifies a user skill
// folder. OpenClaw implements it too, with its own gate for agent destinations.
type skillInstaller interface {
	// SkillSkipReason explains why this install writes no skill folders, or "".
	SkillSkipReason(agent AgentDestination, setsDefault bool) string
	// SkillRoot is the directory holding one folder per skill for the config at configPath.
	SkillRoot(configPath string, env PathEnv) (string, error)
}

// skillChecker lets a target reject a skill folder its agent would not load.
type skillChecker interface {
	CheckSkill(bundle skillBundle) error
}

// skillRootWarner lets a target warn when its skill root is not discovered natively.
type skillRootWarner interface {
	SkillRootWarning(configPath, root string, env PathEnv) string
}

// defaultSkillSkipReason is the gate shared by agents whose user skill folder is read by
// every profile, like roleFileSkipReason: a named agent destination, or a named profile
// not made the default, writes no skill folders.
func defaultSkillSkipReason(adapter Adapter, agent AgentDestination, setsDefault bool) string {
	if !agent.Empty() {
		return skillsAgentReason
	}
	if _, named := adapter.(namedProfileUser); named && !setsDefault {
		return skillsNamedOnlyReason
	}
	return ""
}

// configSkillRoot is the skills directory beside an agent's config file.
func configSkillRoot(configPath string) (string, error) {
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path %s: %w", configPath, err)
	}
	return filepath.Join(filepath.Dir(absolute), "skills"), nil
}

// skillSkip reports whether adapter writes no skill folders for this install, and why.
// An adapter without skill evidence skips them with no reason, like other requirements.
func skillSkip(adapter Adapter, agent AgentDestination, setsDefault bool) (bool, string) {
	installer, ok := adapter.(skillInstaller)
	if !ok {
		return true, ""
	}
	reason := installer.SkillSkipReason(agent, setsDefault)
	return reason != "", reason
}

// loadSkills reads every skills entry's folder beneath root. Each file is its own digest
// and source check, so editing, adding, or removing any bundle file changes the plan.
func loadSkills(root string, profile profilemango.ResolvedProfile) ([]skillBundle, []profilemango.ResourceDigest, profilemango.Diagnostics, []sourceCheck) {
	var bundles []skillBundle
	var digests []profilemango.ResourceDigest
	var diagnostics profilemango.Diagnostics
	var sources []sourceCheck
	for index, ref := range profile.Skills {
		bundle, bundleDigests, bundleSources, err := loadSkillBundle(root, ref)
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, "resource.skill_invalid", fmt.Sprintf("skills[%d]", index), err.Error(), 0, 0)
			continue
		}
		bundles = append(bundles, bundle)
		digests, sources = append(digests, bundleDigests...), append(sources, bundleSources...)
	}
	return bundles, digests, diagnostics.Sorted(), sources
}

// loadSkillBundle reads one skill folder, validates its SKILL.md, and checks its tree
// digest against pinned provenance, naming the actual digest on a mismatch.
func loadSkillBundle(root string, ref profilemango.SkillRef) (skillBundle, []profilemango.ResourceDigest, []sourceCheck, error) {
	dir, err := safeResourcePath(ref.Dir())
	if err != nil {
		return skillBundle{}, nil, nil, err
	}
	bundle := skillBundle{Name: ref.Name(), Dir: dir}
	names, err := skillFolderFiles(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		return skillBundle{}, nil, nil, fmt.Errorf("skill %s: %w", dir, err)
	}
	digests := make([]profilemango.ResourceDigest, 0, len(names))
	sources := make([]sourceCheck, 0, len(names))
	tree := make([]profilemango.SkillTreeFile, 0, len(names))
	for _, name := range names {
		snapshot, err := installfs.SnapshotFile(filepath.Join(root, filepath.FromSlash(dir), filepath.FromSlash(name)))
		if err != nil || !snapshot.Exists {
			return skillBundle{}, nil, nil, fmt.Errorf("skill %s: read %s: %w", dir, name, errOrMissing(err))
		}
		mode := profilemango.SkillMode(snapshot.Mode)
		bundle.Files = append(bundle.Files, skillBundleFile{Path: name, Mode: mode, Content: snapshot.Content})
		digests = append(digests, profilemango.ResourceDigest{Path: dir + "/" + name, Kind: ownershipSkill, SHA256: snapshot.SHA256, Size: snapshot.Size})
		sources = append(sources, sourceCheck{Path: snapshot.Path, Snapshot: snapshot})
		tree = append(tree, profilemango.SkillTreeFile{Path: name, Mode: mode, SHA256: snapshot.SHA256})
	}
	if err := checkSkillBundle(&bundle, ref, tree); err != nil {
		return skillBundle{}, nil, nil, err
	}
	return bundle, digests, sources, nil
}

// checkSkillBundle validates the bundle's SKILL.md and its pinned tree digest.
func checkSkillBundle(bundle *skillBundle, ref profilemango.SkillRef, tree []profilemango.SkillTreeFile) error {
	var document []byte
	found := false
	for _, file := range bundle.Files {
		if file.Path == profilemango.SkillFile {
			document, found = file.Content, true
		}
	}
	if !found {
		return fmt.Errorf("skill %s has no %s", bundle.Dir, profilemango.SkillFile)
	}
	name, err := profilemango.ValidateSkillDocument(bundle.Name, document)
	if err != nil {
		return fmt.Errorf("skill %s: %w", bundle.Dir, err)
	}
	bundle.HasName = name != ""
	if ref.Source == nil {
		return nil
	}
	if actual := profilemango.SkillTreeDigest(tree); actual != ref.Source.SHA256 {
		return fmt.Errorf("skill %s tree digest is %s, but source.sha256 is %s; re-vendor it from %s at %s or update source.sha256", bundle.Dir, actual, ref.Source.SHA256, ref.Source.Repo, ref.Source.Commit)
	}
	return nil
}

func errOrMissing(err error) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("file does not exist")
}

// skillFolderFiles lists the regular files beneath dir, slash-separated and sorted. It
// rejects symlinks and other non-regular entries and more than MaxSkillFiles files.
func skillFolderFiles(dir string) ([]string, error) {
	var names []string
	err := filepath.WalkDir(dir, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("list %s: %w", filepath.Base(current), err)
		}
		relative, err := filepath.Rel(dir, current)
		if err != nil {
			return fmt.Errorf("resolve %s: %w", current, err)
		}
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink; vendor the file it points to instead", filepath.ToSlash(relative))
		case entry.IsDir():
			return nil
		case !entry.Type().IsRegular():
			return fmt.Errorf("%s is not a regular file", filepath.ToSlash(relative))
		}
		if names = append(names, filepath.ToSlash(relative)); len(names) > profilemango.MaxSkillFiles {
			return fmt.Errorf("holds more than %d files", profilemango.MaxSkillFiles)
		}
		return nil
	})
	sort.Strings(names)
	return names, err
}

// addSkillPatches writes every loaded skill folder as owned whole files under the
// target's skill root. A symlink or non-directory in a destination folder is a conflict
// naming its path; unowned files already in an adopted folder are listed in a warning.
// A non-empty reason blocks the target.
func addSkillPatches(adapter Adapter, request Request, target TargetRequest, loaded loadedInput, ownership Manifest, configPath string, patch *Patch, targetPlan *TargetPlan) (string, string) {
	if len(loaded.Skills) == 0 {
		return "", ""
	}
	if skipped, _ := skillSkip(adapter, target.Agent, request.Default); skipped {
		return "", ""
	}
	root, err := adapter.(skillInstaller).SkillRoot(configPath, request.Env)
	if err != nil {
		return fmt.Sprintf("resolve skill folder: %v", err), "install.skill_root_unresolved"
	}
	if warner, ok := adapter.(skillRootWarner); ok {
		if warning := warner.SkillRootWarning(configPath, root, request.Env); warning != "" {
			targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.skill_root_undiscovered", "skills", warning, 0, 0)
		}
	}
	for _, bundle := range loaded.Skills {
		if checker, ok := adapter.(skillChecker); ok {
			if err := checker.CheckSkill(bundle); err != nil {
				return err.Error(), "install.skill_unsupported"
			}
		}
		folder := filepath.Join(root, bundle.Name)
		label := func(path string) string { return relativeOwnedPath(configPath, path, request.Env) }
		if blocked := skillDestinationConflicts(root, bundle, label, targetPlan); blocked {
			continue
		}
		warnUnmanagedSkillFiles(folder, bundle, ownership, label, targetPlan)
		for _, file := range bundle.Files {
			path := filepath.Join(folder, filepath.FromSlash(file.Path))
			patch.Files = append(patch.Files, FilePatch{Path: path, Label: label(path), Content: file.Content, Mode: file.Mode, Adoptable: true, Ownership: []string{ownershipSkill}})
		}
		targetPlan.Skills = append(targetPlan.Skills, bundle.Name)
	}
	return "", ""
}

// skillDestinationConflicts reports every symlink or non-directory on the way to a
// bundle file below root, and a symlinked file, as a conflict naming its path.
func skillDestinationConflicts(root string, bundle skillBundle, label func(string) string, targetPlan *TargetPlan) bool {
	seen := map[string]struct{}{}
	conflict := false
	for _, file := range bundle.Files {
		parts := strings.Split(path.Join(bundle.Name, file.Path), "/")
		current := root
		for index := -1; index < len(parts); index++ {
			if index >= 0 {
				current = filepath.Join(current, parts[index])
			}
			if _, done := seen[current]; done {
				continue
			}
			seen[current] = struct{}{}
			if reason := destinationProblem(current, index == len(parts)-1); reason != "" {
				targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.skill_destination_conflict", label(current), label(current)+" "+reason+"; profile-mango does not write through it, so move it aside to install skill "+bundle.Name, 0, 0)
				conflict = true
				break
			}
		}
	}
	return conflict
}

// destinationProblem says why an existing path cannot hold a skill folder or file.
func destinationProblem(path string, isFile bool) string {
	info, err := os.Lstat(path)
	switch {
	case err != nil:
		return ""
	case info.Mode()&fs.ModeSymlink != 0:
		return "is a symlink"
	case !isFile && !info.IsDir():
		return "is not a directory"
	}
	return ""
}

// warnUnmanagedSkillFiles lists files in an existing skill folder that neither the
// bundle nor the ownership manifest accounts for; they stay, and the agent may read them.
// Only that folder is read.
func warnUnmanagedSkillFiles(folder string, bundle skillBundle, ownership Manifest, label func(string) string, targetPlan *TargetPlan) {
	planned := make(map[string]struct{}, len(bundle.Files))
	for _, file := range bundle.Files {
		planned[filepath.Join(folder, filepath.FromSlash(file.Path))] = struct{}{}
	}
	var extra []string
	_ = filepath.WalkDir(folder, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		_, bundled := planned[current]
		if _, owned := manifestEntry(ownership, current); bundled || owned || strings.Contains(entry.Name(), ".profile-mango.") {
			return nil
		}
		relative, _ := filepath.Rel(folder, current)
		extra = append(extra, filepath.ToSlash(relative))
		return nil
	})
	if len(extra) == 0 {
		return
	}
	sort.Strings(extra)
	listed := extra
	if len(listed) > maxListedFiles {
		listed = append(append([]string(nil), extra[:maxListedFiles]...), fmt.Sprintf("and %d more", len(extra)-maxListedFiles))
	}
	message := fmt.Sprintf("%s has files profile-mango does not manage: %s; they are kept, and the agent may still read them", label(folder), strings.Join(listed, ", "))
	targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.skill_unmanaged_files", label(folder), message, 0, 0)
}

// skillReleaseDirs sets, for each released skill file profile-mango created, the folders
// its delete removes afterwards while they are empty: from its directory up to its skill
// folder directly below the target's skill root.
func skillReleaseDirs(adapter Adapter, configPath string, env PathEnv, files []FilePatch) {
	installer, ok := adapter.(skillInstaller)
	if !ok || !slices.ContainsFunc(files, func(file FilePatch) bool { return file.Release && file.Delete }) {
		return
	}
	root, err := installer.SkillRoot(configPath, env)
	if err != nil {
		return
	}
	for index := range files {
		file := &files[index]
		if !file.Release || !file.Delete {
			continue
		}
		relative, err := filepath.Rel(root, filepath.Dir(file.Path))
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			continue
		}
		for dir := filepath.Dir(file.Path); dir != root; dir = filepath.Dir(dir) {
			file.RemoveEmptyDirs = append(file.RemoveEmptyDirs, dir)
		}
	}
}
