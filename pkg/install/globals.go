package install

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// RequirementGlobalInstructions is the profile's globalInstructions files for a target.
const RequirementGlobalInstructions = "globalInstructions"

// Manifest ownership markers. A whole owned file (a global instruction file or a
// role subagent file) carries its kind plus the state it had before profile-mango
// first wrote it, so "mango use" can release it.
const (
	ownershipGlobalInstruction = "global-instruction"
	ownershipRoleDefinition    = "role-definition"
	priorAbsent                = "prior:absent"
	priorSHA256Prefix          = "prior-sha256:"
)

// wholeFileKinds are the ownership tags of separately owned whole files.
var wholeFileKinds = []string{ownershipGlobalInstruction, ownershipRoleDefinition}

// wholeFileKind returns the whole-file ownership tag among fields, or "".
func wholeFileKind(fields []string) string {
	for _, kind := range wholeFileKinds {
		if slices.Contains(fields, kind) {
			return kind
		}
	}
	return ""
}

// globalInstructionFiles lists, per exact target, the user-level instruction files
// the target reads from its main config directory. Each entry is backed by pinned
// evidence in docs/dev/agents/<target>.md; targets absent here skip the requirement.
var globalInstructionFiles = map[Target][]string{
	{Name: claudecode.TargetName, Version: claudecode.TargetVersion}: {"CLAUDE.md"},
	{Name: codex.TargetName, Version: codex.TargetVersion}:           {"AGENTS.md"},
	{Name: ohmypi.TargetName, Version: ohmypi.TargetVersion}:         {"AGENTS.md", "RULES.md"},
	{Name: opencode.TargetName, Version: opencode.TargetVersion}:     {"AGENTS.md"},
}

// GlobalInstructionFiles reports the qualified global instruction file names for target.
func GlobalInstructionFiles(target Target) []string {
	return append([]string(nil), globalInstructionFiles[target]...)
}

// homeInstructionOwner is the one target that installs, owns, and releases home
// instruction files. Its pinned evidence reads ~/AGENTS.md as an ancestor context file
// in every directory under the home directory (docs/dev/agents/pi.md); other targets
// read it only in some directories. A single owner keeps one manifest per file.
var homeInstructionOwner = Target{Name: pi.TargetName, Version: pi.TargetVersion}

// homeInstructionFiles lists the qualified file names under globalInstructions.home.
var homeInstructionFiles = []string{"AGENTS.md"}

// globalFile is one loaded globalInstructions entry for a target.
type globalFile struct {
	Name    string
	Content []byte
	Digest  profilemango.ResourceDigest
}

// loadGlobalInstructions reads every globalInstructions resource beneath root.
func loadGlobalInstructions(root string, profile profilemango.ResolvedProfile) (map[string][]globalFile, []profilemango.ResourceDigest, profilemango.Diagnostics, []sourceCheck) {
	result := make(map[string][]globalFile, len(profile.GlobalInstructions))
	var digests []profilemango.ResourceDigest
	var diagnostics profilemango.Diagnostics
	var sources []sourceCheck
	for _, target := range sortedNames(profile.GlobalInstructions) {
		files := profile.GlobalInstructions[target]
		for _, name := range sortedNames(files) {
			file, source, err := loadGlobalFile(root, name, files[name])
			if err != nil {
				diagnostics.Add(profilemango.SeverityError, "resource.read", "globalInstructions."+target+"."+name, err.Error(), 0, 0)
				continue
			}
			result[target] = append(result[target], file)
			digests = append(digests, file.Digest)
			sources = append(sources, source)
		}
	}
	return result, digests, diagnostics.Sorted(), sources
}

func loadGlobalFile(root, name, resource string) (globalFile, sourceCheck, error) {
	clean, err := safeResourcePath(resource)
	if err != nil {
		return globalFile{}, sourceCheck{}, err
	}
	snapshot, err := installfs.SnapshotFile(filepath.Join(root, filepath.FromSlash(clean)))
	if err != nil {
		return globalFile{}, sourceCheck{}, fmt.Errorf("read %s: %w", clean, err)
	}
	if !snapshot.Exists {
		return globalFile{}, sourceCheck{}, fmt.Errorf("resource %s does not exist", clean)
	}
	digest := profilemango.ResourceDigest{Path: clean, Kind: "global-instruction", SHA256: snapshot.SHA256, Size: snapshot.Size}
	return globalFile{Name: name, Content: snapshot.Content, Digest: digest}, sourceCheck{Path: snapshot.Path, Snapshot: snapshot}, nil
}

// globalPatches returns whole-file patches for the target's globalInstructions, or the
// requirement it skips. A non-empty reason blocks the target with code.
func globalPatches(request Request, target TargetRequest, loaded loadedInput, mode *InstallMode) ([]FilePatch, *SkippedRequirement, string, string) {
	files := loaded.Globals[target.Target.Name]
	if len(files) == 0 {
		return nil, nil, "", ""
	}
	qualified := globalInstructionFiles[target.Target]
	if reason := globalSkipReason(target, qualified, mode); reason != "" {
		skipped, blocked, code := skipOrBlock(request, len(files), "", reason)
		return nil, skipped, blocked, code
	}
	patches := make([]FilePatch, 0, len(files))
	for _, file := range files {
		if !slices.Contains(qualified, file.Name) {
			return nil, nil, fmt.Sprintf("%s does not read a global %s; qualified files: %s", target.Target.String(), file.Name, strings.Join(qualified, ", ")), "install.global_instruction_unqualified"
		}
		patches = append(patches, FilePatch{Path: file.Name, Content: file.Content, Adoptable: true, Ownership: []string{ownershipGlobalInstruction}})
	}
	return patches, nil, "", ""
}

// skipOrBlock reports count unmet global instruction files as a skipped requirement,
// or with --strict as a blocking reason and code.
func skipOrBlock(request Request, count int, value, reason string) (*SkippedRequirement, string, string) {
	if request.Strict {
		return nil, fmt.Sprintf("--strict: %d global instruction files cannot be installed: %s", count, reason), "install.strict_requirement_unsupported"
	}
	return &SkippedRequirement{Requirement: RequirementGlobalInstructions, Count: count, Value: value, Reason: reason}, "", ""
}

// homePatches returns the profile's globalInstructions.home files for the owning
// target. Another target reports them skipped only when the owner is not selected.
func homePatches(request Request, target TargetRequest, loaded loadedInput, mode *InstallMode) ([]FilePatch, *SkippedRequirement, string, string) {
	files := loaded.Globals[profilemango.HomeInstructions]
	if len(files) == 0 || (target.Target != homeInstructionOwner && homeOwnerSelected(request)) {
		return nil, nil, "", ""
	}
	home, reason := homeSkipReason(request, target, mode)
	if reason != "" {
		skipped, blocked, code := skipOrBlock(request, len(files), profilemango.HomeInstructions, reason)
		return nil, skipped, blocked, code
	}
	patches := make([]FilePatch, 0, len(files))
	for _, file := range files {
		if !slices.Contains(homeInstructionFiles, file.Name) {
			return nil, nil, fmt.Sprintf("no pinned evidence qualifies ~/%s; qualified home files: %s", file.Name, strings.Join(homeInstructionFiles, ", ")), "install.global_instruction_unqualified"
		}
		patches = append(patches, FilePatch{Path: filepath.Join(home, file.Name), Label: "~/" + file.Name, Content: file.Content, Adoptable: true, Ownership: []string{ownershipGlobalInstruction}})
	}
	return patches, nil, "", ""
}

// homeSkipReason returns the user home directory, or why the target writes no home
// instruction files: it is not the owner, the home is unresolvable, or a global gate.
func homeSkipReason(request Request, target TargetRequest, mode *InstallMode) (string, string) {
	if target.Target != homeInstructionOwner {
		return "", fmt.Sprintf("home instruction files install only with %s, whose pinned evidence reads ~/AGENTS.md in every directory under the home directory; add --target %s", homeInstructionOwner, homeInstructionOwner.Name)
	}
	if !request.Env.enabled() {
		return "", "the user home directory is not resolvable"
	}
	home, err := request.Env.home()
	if err != nil {
		return "", err.Error()
	}
	return home, globalSkipReason(target, homeInstructionFiles, mode)
}

// homeOwnerSelected reports whether the request plans the home instruction owner.
func homeOwnerSelected(request Request) bool {
	if request.All {
		return true
	}
	return slices.ContainsFunc(request.Targets, func(target TargetRequest) bool { return target.Target == homeInstructionOwner })
}

// globalSkipReason explains why a target receives no global instruction files: no
// pinned evidence, a named agent destination, or a named profile that is not made
// the default, since a global file would change every profile of that agent.
func globalSkipReason(target TargetRequest, qualified []string, mode *InstallMode) string {
	switch {
	case !target.Agent.Empty():
		return "named agent destinations do not install global instruction files"
	case len(qualified) == 0:
		return "no pinned evidence qualifies a global instruction file for " + target.Target.String()
	case mode != nil && mode.Mode == InstallModeNamedProfile && !mode.SetsDefault:
		return "global instruction files apply to every profile; pass --default or run mango use"
	}
	return ""
}

// extendPatch adds the target's global and home instruction files and role subagent
// files and, for mango use, releases of owned files the new plan no longer writes. A
// non-empty reason blocks the target.
func extendPatch(request Request, target TargetRequest, loaded loadedInput, roles roleInput, ownership Manifest, configPath string, patch *Patch, targetPlan *TargetPlan) (string, string) {
	for _, plan := range []func(Request, TargetRequest, loadedInput, *InstallMode) ([]FilePatch, *SkippedRequirement, string, string){globalPatches, homePatches} {
		files, skipped, reason, code := plan(request, target, loaded, targetPlan.Install)
		if reason != "" {
			return reason, code
		}
		if skipped != nil {
			targetPlan.SkippedRequirements = append(targetPlan.SkippedRequirements, *skipped)
		}
		patch.Files = append(patch.Files, files...)
	}
	addHomeWarning(patch, targetPlan)
	if reason, code := addRolePatches(request, roles, patch, targetPlan); reason != "" {
		return reason, code
	}
	if !request.Release || !target.Agent.Empty() {
		return "", ""
	}
	planned := make(map[string]struct{}, len(patch.Files))
	for _, file := range patch.Files {
		path, err := patchPath(target.ConfigPath, file.Path)
		if err != nil {
			return err.Error(), "install.patch_path_invalid"
		}
		planned[path] = struct{}{}
	}
	releases, reason, code := releasePatches(ownership, configPath, planned)
	for index := range releases {
		releases[index].Label = relativeOwnedPath(configPath, releases[index].Path, request.Env)
	}
	patch.Files = append(patch.Files, releases...)
	return reason, code
}

// addHomeWarning notes that a planned home instruction file is shared by every agent
// that reads the home directory as an ancestor of its working directory.
func addHomeWarning(patch *Patch, targetPlan *TargetPlan) {
	for _, file := range patch.Files {
		if file.Label != "" && filepath.IsAbs(file.Path) {
			targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.home_instruction_shared", "globalInstructions.home", file.Label+" is not owned by one agent: Pi reads it in every directory under the home directory, and other agents read it in some directories, such as outside a Git repository", 0, 0)
		}
	}
}

// homeLabel names an owned file outside the config directory by its path under the home
// directory, e.g. ~/AGENTS.md, so plans and status do not show a bare base name.
func homeLabel(env PathEnv, configPath, path string) string {
	if relative, err := filepath.Rel(filepath.Dir(configPath), path); err == nil && !strings.HasPrefix(relative, "..") {
		return ""
	}
	if !env.enabled() {
		return ""
	}
	home, err := env.home()
	if err != nil {
		return ""
	}
	relative, err := filepath.Rel(home, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return ""
	}
	return "~/" + filepath.ToSlash(relative)
}

// fileOwnership is the manifest provenance for one planned file: the adapter's tags plus,
// for a whole owned file, the state it had before profile-mango first wrote it.
func fileOwnership(file FilePatch, ownership Manifest, path, configPath string, before installfs.Snapshot) []string {
	tags := append([]string(nil), file.Ownership...)
	if path == configPath || file.Delete || file.Release || wholeFileKind(tags) == "" {
		return tags
	}
	if marker := priorMarker(ownership, path, before); marker != "" {
		tags = append(tags, marker)
	}
	return tags
}

// priorMarker is the pre-install state recorded for a separately owned file: the
// marker already in the manifest while it stays owned, otherwise its current state.
func priorMarker(ownership Manifest, path string, before installfs.Snapshot) string {
	if entry, owned := manifestEntry(ownership, path); owned {
		return entryPrior(entry)
	}
	if before.Exists {
		return priorSHA256Prefix + before.SHA256
	}
	return priorAbsent
}

func entryPrior(entry ManifestFile) string {
	for _, field := range entry.Fields {
		if field == priorAbsent || strings.HasPrefix(field, priorSHA256Prefix) {
			return field
		}
	}
	return ""
}

func manifestEntry(manifest Manifest, path string) (ManifestFile, bool) {
	for _, file := range manifest.Files {
		if file.Path == path {
			return file, true
		}
	}
	return ManifestFile{}, false
}

// releasePatches returns patches that give back every whole owned file the new plan no
// longer writes: deleting it when profile-mango created it, or restoring its pre-install
// bytes from a create-only backup. Other owned files, such as a named profile, are kept.
func releasePatches(ownership Manifest, configPath string, planned map[string]struct{}) ([]FilePatch, string, string) {
	var patches []FilePatch
	for _, entry := range ownership.Files {
		if _, kept := planned[entry.Path]; kept || entry.Path == configPath || wholeFileKind(entry.Fields) == "" {
			continue
		}
		prior := entryPrior(entry)
		switch {
		case prior == priorAbsent:
			patches = append(patches, FilePatch{Path: entry.Path, Delete: true, Release: true})
		case prior != "":
			backup, err := findBackup(entry.Path, strings.TrimPrefix(prior, priorSHA256Prefix))
			if err != nil {
				return nil, err.Error(), "install.release_backup_missing"
			}
			patches = append(patches, FilePatch{Path: entry.Path, Content: backup.Content, NoOverride: true, Release: true})
		default:
			return nil, fmt.Sprintf("owned file %s has no recorded pre-install state; run mango undo for its install instead", filepath.Base(entry.Path)), "install.release_state_unknown"
		}
	}
	return patches, "", ""
}

// findBackup locates a create-only backup of path whose bytes hash to sha256.
func findBackup(path, sha256 string) (installfs.Snapshot, error) {
	matches, err := filepath.Glob(filepath.Clean(path) + ".profile-mango.bak.*")
	if err != nil {
		return installfs.Snapshot{}, fmt.Errorf("list backups of %s: %w", filepath.Base(path), err)
	}
	sort.Strings(matches)
	for _, match := range matches {
		snapshot, err := installfs.SnapshotFile(match)
		if err == nil && snapshot.Exists && snapshot.SHA256 == sha256 {
			return snapshot, nil
		}
	}
	return installfs.Snapshot{}, fmt.Errorf("no backup of %s holds its pre-install bytes; restore it manually or run mango undo", filepath.Base(path))
}

func sortedNames[V any](values map[string]V) []string {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
