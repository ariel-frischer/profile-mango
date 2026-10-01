package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/ariel-frischer/profile-mango/pkg/render"
)

type sourceCheck struct {
	Path     string
	Snapshot installfs.Snapshot
}

type loadedInput struct {
	Profile     profilemango.ResolvedProfile
	Route       profilemango.RouteBinding
	Resources   []render.Resource
	InputSHA256 string
	Sources     []sourceCheck
	// Globals holds each target's loaded globalInstructions files.
	Globals map[string][]globalFile
	// AgentFiles holds each target's loaded agentFiles.
	AgentFiles map[string][]globalFile
	// Skills holds each skills entry's loaded folder, installed verbatim.
	Skills []skillBundle
	// RoleInstructions holds each role's loaded instructions resource.
	RoleInstructions map[string][]byte
	// Bindings resolves {{route.…}} placeholders in the resources per target.
	Bindings profilemango.Bindings
}

func BuildPlan(request Request) (Plan, error) {
	registry := request.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	targets, err := selectTargets(request, registry)
	if err != nil {
		return Plan{}, err
	}
	loaded, diagnostics, err := loadInput(request)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{APIVersion: PlanAPIVersion, Kind: PlanKind, Profile: request.ProfileName, Backup: request.Backup, Override: request.Override, Strict: request.Strict, InputSHA256: loaded.InputSHA256}
	plan.sources = loaded.Sources
	for _, targetRequest := range targets {
		targetPlan := planTarget(request, registry, targetRequest, loaded)
		attachVersionCheck(&targetPlan, request.DetectVersion)
		plan.Targets = append(plan.Targets, targetPlan)
		plan.Diagnostics = append(plan.Diagnostics, targetPlan.Diagnostics...)
	}
	plan.Diagnostics = append(plan.Diagnostics, diagnostics...)
	plan.Status = aggregateStatus(plan.Targets)
	if allSkipped(plan.Targets) {
		plan.Diagnostics.Add(profilemango.SeverityError, "install.no_agents_found", "targets", noAgentsFound, 0, 0)
	}
	plan.normalize()
	plan.PlanID, err = planID(plan)
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func selectTargets(request Request, registry *Registry) ([]TargetRequest, error) {
	if request.All && len(request.Targets) > 0 {
		return nil, fmt.Errorf("--all and explicit targets are mutually exclusive")
	}
	if request.All {
		for _, target := range registry.Targets() {
			request.Targets = append(request.Targets, TargetRequest{Target: target})
		}
	}
	if len(request.Targets) == 0 {
		return nil, fmt.Errorf("at least one exact target is required")
	}
	result := append([]TargetRequest(nil), request.Targets...)
	seen := make(map[string]struct{}, len(result))
	for index := range result {
		if result[index].Target.Name == "" || result[index].Target.Version == "" {
			return nil, fmt.Errorf("target %d must use exact target@version syntax", index+1)
		}
		key := result[index].Target.String()
		if _, found := seen[key]; found {
			return nil, fmt.Errorf("duplicate target: %s", key)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Target.String() < result[j].Target.String() })
	return result, nil
}

func loadInput(request Request) (loadedInput, profilemango.Diagnostics, error) {
	if request.ProfileName == "" || !safeName(request.ProfileName) {
		return loadedInput{}, nil, fmt.Errorf("profile name must be a simple lowercase name")
	}
	profiles := make(map[string]profilemango.PolicyProfile)
	var diagnostics profilemango.Diagnostics
	sources := make([]sourceCheck, 0, 4)
	if err := loadProfileChain(request.ProfilesRoot, request.ProfileName, profiles, &diagnostics, &sources); err != nil {
		return loadedInput{}, diagnostics, err
	}
	resolved, resolveDiagnostics := profilemango.Resolve(profiles, request.ProfileName)
	diagnostics = append(diagnostics, resolveDiagnostics...)
	if diagnostics.HasErrors() {
		return loadedInput{}, diagnostics.Sorted(), fmt.Errorf("resolve install profile: %s", diagnostics.Error())
	}
	bindingsSnapshot, err := installfs.SnapshotFile(request.BindingsPath)
	if err != nil || !bindingsSnapshot.Exists {
		if err == nil {
			err = missingBindingsError(request.BindingsPath)
		}
		return loadedInput{}, diagnostics, fmt.Errorf("read bindings: %w", err)
	}
	sources = append(sources, sourceCheck{Path: bindingsSnapshot.Path, Snapshot: bindingsSnapshot})
	bindings, bindingDiagnostics := profilemango.ParseBindings(bindingsSnapshot.Content)
	diagnostics = append(diagnostics, bindingDiagnostics...)
	route, found := bindings.Routes[resolved.RouteRef]
	if !found {
		diagnostics.Add(profilemango.SeverityError, "binding.route_missing", "spec.routeRef", "routeRef is not present in bindings", 0, 0)
	}
	resources, resourceDiagnostics, resourceSources := loadResources(request.ResourceRoot, resolved)
	diagnostics = append(diagnostics, resourceDiagnostics...)
	sources = append(sources, resourceSources...)
	globals, globalDigests, globalDiagnostics, globalSources := loadGlobalInstructions(request.ResourceRoot, resolved)
	diagnostics = append(diagnostics, globalDiagnostics...)
	sources = append(sources, globalSources...)
	roles, roleDigests, roleDiagnostics, roleSources := loadRoleInstructions(request.ResourceRoot, resolved)
	diagnostics, sources = append(diagnostics, roleDiagnostics...), append(sources, roleSources...)
	agentFiles, agentDigests, agentDiagnostics, agentSources := loadAgentFiles(request.ResourceRoot, resolved)
	diagnostics, sources = append(diagnostics, agentDiagnostics...), append(sources, agentSources...)
	skills, skillDigests, skillDiagnostics, skillSources := loadSkills(request.ResourceRoot, resolved)
	diagnostics, sources = append(diagnostics, skillDiagnostics...), append(sources, skillSources...)
	loaded := loadedInput{Resources: resources, Globals: globals, AgentFiles: agentFiles, Skills: skills, RoleInstructions: roles, Bindings: bindings}
	diagnostics = append(diagnostics, loaded.routeRefDiagnostics()...)
	if diagnostics.HasErrors() {
		return loadedInput{}, diagnostics.Sorted(), fmt.Errorf("validate install inputs: %s", diagnostics.Error())
	}
	digest := make([]profilemango.ResourceDigest, 0, len(resources))
	for _, resource := range resources {
		digest = append(digest, resource.Digest)
	}
	inputHash, err := hashValue(struct {
		Profile   profilemango.ResolvedProfile
		Route     profilemango.RouteBinding
		Resources []profilemango.ResourceDigest
		Globals   []profilemango.ResourceDigest `json:",omitempty"`
	}{Profile: resolved, Route: route, Resources: digest, Globals: append(append(append(globalDigests, roleDigests...), agentDigests...), skillDigests...)})
	if err != nil {
		return loadedInput{}, diagnostics, fmt.Errorf("hash install inputs: %w", err)
	}
	loaded.Profile, loaded.Route, loaded.InputSHA256, loaded.Sources = resolved, route, inputHash, sources
	return loaded, diagnostics.Sorted(), nil
}

// missingBindingsError names the exact fix for a missing local bindings file:
// copy the starter example scaffolded by "mango init", or run init
// again in a fresh package.
func missingBindingsError(path string) error {
	example := filepath.Join(filepath.Dir(path), "local.example.yaml")
	return fmt.Errorf("bindings file does not exist; create it with: cp %s %s, or run: mango init", example, path)
}

func loadProfileChain(root, name string, profiles map[string]profilemango.PolicyProfile, diagnostics *profilemango.Diagnostics, sources *[]sourceCheck) error {
	if _, found := profiles[name]; found {
		return nil
	}
	if !safeName(name) {
		return fmt.Errorf("profile parent name is unsafe: %s", name)
	}
	path := filepath.Join(root, name, "profile.yaml")
	snapshot, err := installfs.SnapshotFile(path)
	if err != nil {
		return fmt.Errorf("read profile %s: %w", name, err)
	}
	if !snapshot.Exists {
		return fmt.Errorf("profile %q does not exist", name)
	}
	profile, foundDiagnostics := profilemango.ParseProfileAt(snapshot.Content, name)
	for _, diagnostic := range foundDiagnostics {
		diagnostic.Path = name + "." + diagnostic.Path
		*diagnostics = append(*diagnostics, diagnostic)
	}
	profiles[name] = profile
	*sources = append(*sources, sourceCheck{Path: snapshot.Path, Snapshot: snapshot})
	if profile.Extends != "" {
		return loadProfileChain(root, profile.Extends, profiles, diagnostics, sources)
	}
	return nil
}

func loadResources(root string, profile profilemango.ResolvedProfile) ([]render.Resource, profilemango.Diagnostics, []sourceCheck) {
	type reference struct{ path, kind string }
	refs := make([]reference, 0, len(profile.Instructions))
	for _, path := range profile.Instructions {
		refs = append(refs, reference{path: path, kind: "instruction"})
	}
	seen := make(map[string]struct{}, len(refs))
	resources := make([]render.Resource, 0, len(refs))
	sources := make([]sourceCheck, 0, len(refs))
	var diagnostics profilemango.Diagnostics
	for _, reference := range refs {
		clean, err := safeResourcePath(reference.path)
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, "resource.path_escape", reference.path, err.Error(), 0, 0)
			continue
		}
		if _, found := seen[clean]; found {
			diagnostics.Add(profilemango.SeverityError, "resource.duplicate", clean, "resource path is referenced more than once", 0, 0)
			continue
		}
		seen[clean] = struct{}{}
		snapshot, err := installfs.SnapshotFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil || !snapshot.Exists {
			if err == nil {
				err = fmt.Errorf("resource does not exist")
			}
			diagnostics.Add(profilemango.SeverityError, "resource.read", clean, err.Error(), 0, 0)
			continue
		}
		resources = append(resources, render.Resource{Digest: profilemango.ResourceDigest{Path: clean, Kind: reference.kind, SHA256: snapshot.SHA256, Size: snapshot.Size}, Content: snapshot.Content})
		sources = append(sources, sourceCheck{Path: snapshot.Path, Snapshot: snapshot})
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Digest.Path < resources[j].Digest.Path })
	return resources, diagnostics.Sorted(), sources
}

func planTarget(request Request, registry *Registry, targetRequest TargetRequest, loaded loadedInput) TargetPlan {
	adapter, found := registry.Lookup(targetRequest.Target)
	if !found {
		return blockedTarget(targetRequest, AdapterMetadata{Target: targetRequest.Target.Name, Version: targetRequest.Target.Version, Status: StatusUnavailable}, "no static adapter is registered", "install.target.unsupported")
	}
	metadata := adapter.Metadata()
	targetPlan := TargetPlan{Target: targetRequest.Target, Metadata: metadata, ConfigPath: targetRequest.ConfigPath, ManifestPath: targetRequest.ManifestPath}
	if !targetRequest.Agent.Empty() {
		agent := targetRequest.Agent
		targetPlan.Agent = &agent
	}
	if err := validateAgentDestination(targetRequest); err != nil {
		return blockedTargetPlan(targetPlan, err.Error(), "install.agent_destination_invalid")
	}
	if err := validateSkillAgent(adapter, targetRequest.SkillAgent); err != nil {
		return blockedTargetPlan(targetPlan, err.Error(), "install.agent_destination_invalid")
	}
	targetPlan.Install = installModeFor(adapter, request, targetRequest)
	loaded, err := loaded.forTarget(targetRequest.Target.Name)
	if err != nil {
		return blockedTargetPlan(targetPlan, err.Error(), render.RouteRefInvalidCode)
	}
	if !metadata.Installable {
		return blockedTargetPlan(targetPlan, metadata.Reason, "install.target.blocked")
	}
	source := ConfigSourceExplicit
	if strings.TrimSpace(targetRequest.ConfigPath) == "" {
		path, reason := resolveDefaultConfigPath(adapter, targetRequest, request.Env)
		if reason != "" {
			return blockedTargetPlan(targetPlan, reason, "install.config_path_required")
		}
		targetRequest.ConfigPath, targetPlan.ConfigPath, source = path, path, ConfigSourceDefault
	}
	targetPlan.Config = &ConfigDestination{Path: targetRequest.ConfigPath, Source: source}
	if source == ConfigSourceDefault {
		if missing, done := missingAgentFolder(request, targetPlan); done {
			return missing
		}
	}
	config, err := installfs.SnapshotFile(targetRequest.ConfigPath)
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("inspect config path: %v", err), "install.config_path_unsafe")
	}
	targetPlan.Config.Path = config.Path
	manifestPath := targetRequest.ManifestPath
	if manifestPath == "" {
		manifestPath = targetRequest.ConfigPath + ".profile-mango.manifest.json"
	}
	manifestSnapshot, err := installfs.SnapshotFile(manifestPath)
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("inspect manifest path: %v", err), "install.manifest_path_unsafe")
	}
	targetPlan.DestinationSHA256, err = hashValue([]string{config.Path, manifestSnapshot.Path})
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("hash install destination: %v", err), "install.destination_hash_failed")
	}
	ownership, err := decodeManifest(manifestSnapshot.Content, targetRequest.Target)
	if err != nil {
		return blockedTargetPlan(targetPlan, err.Error(), "install.manifest_invalid")
	}
	if manifestSnapshot.Exists && !sameAgentOwnership(ownership, targetRequest.Agent) {
		return blockedTargetPlan(targetPlan, "ownership manifest belongs to a different agent destination", "install.agent_owner_mismatch")
	}
	if targetPlan.Agent != nil && manifestSnapshot.Exists && !namedAgentFilesMatch(ownership, config.Path) {
		return blockedTargetPlan(targetPlan, "named agent ownership manifest has unrelated files", "install.agent_owner_mismatch")
	}
	if targetPlan.Agent != nil && config.Exists {
		hash, owned := ownershipHash(ownership, config.Path)
		if !owned || hash != config.SHA256 {
			return blockedTargetPlan(targetPlan, "named definition is unowned or edited; --override cannot replace it", "install.agent_file_conflict")
		}
	}
	namedFile, err := snapshotNamedFile(adapter, targetPlan.Install, config, request.Env)
	if err != nil {
		return blockedTargetPlan(targetPlan, err.Error(), "install.named_profile_path_unsafe")
	}
	useNamedProfileFile(adapter, targetPlan.Install, namedFile.Path, request.Env)
	profile, resources, route, skipped, strictReason := targetSubset(adapter, request, targetRequest, loaded)
	if strictReason != "" {
		return blockedTargetPlan(targetPlan, strictReason, "install.strict_requirement_unsupported")
	}
	targetPlan.SkippedRequirements = skipped
	input := AdapterInput{Target: targetRequest.Target, Agent: targetRequest.Agent, ConfigPath: config.Path, ManifestPath: manifestSnapshot.Path, Profile: profile, Route: route, Resources: resources, Config: snapshotFromFS(config), Manifest: snapshotFromFS(manifestSnapshot), Ownership: ownership, HasManifest: manifestSnapshot.Exists, Override: request.Override, NamedFile: snapshotFromFS(namedFile), Skills: installedSkillNames(adapter, targetRequest, request.Default, loaded.Skills), SkillAgent: targetRequest.SkillAgent}
	if targetPlan.Install != nil {
		input.Install = *targetPlan.Install
	}
	patch, err := adapter.Plan(input)
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("adapter planning failed: %v", err), "install.adapter_plan_failed")
	}
	targetPlan.Diagnostics = append(targetPlan.Diagnostics, patch.Diagnostics...)
	targetPlan.Fields = publicFields(patch.Fields)
	roles := newRoleInput(adapter, request, targetRequest, profile, loaded)
	if reason, code := addSkillPatches(adapter, request, targetRequest, loaded, ownership, config.Path, &patch, &targetPlan); reason != "" {
		return blockedTargetPlan(targetPlan, reason, code)
	}
	if reason, code := extendPatch(request, targetRequest, loaded, roles, ownership, config.Path, &patch, &targetPlan); reason != "" {
		return blockedTargetPlan(targetPlan, reason, code)
	}
	skillReleaseDirs(adapter, config.Path, request.Env, ownership, patch.Files)
	changes, blocked := planFiles(request, targetRequest, patch, ownership, config, &targetPlan)
	if blocked {
		targetPlan.Status = StatusConflict
		targetPlan.Reason = conflictReason(targetPlan.Diagnostics)
		return targetPlan
	}
	manifest, manifestData, err := nextManifest(ownership, manifestSnapshot, targetRequest, request.ProfileName, patch, targetPlan.Files, changes)
	if err != nil {
		targetPlan.Status = StatusBlocked
		targetPlan.Reason = err.Error()
		targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.manifest_build_failed", "manifest", err.Error(), 0, 0)
		return targetPlan
	}
	if manifest != nil && !bytes.Equal(manifestSnapshot.Content, manifestData) {
		changes = append(changes, installfs.Change{Path: manifestSnapshot.Path, Before: manifestSnapshot, Content: manifestData})
		targetPlan.Files = append(targetPlan.Files, FilePlan{Path: filepath.Base(manifestSnapshot.Path), Action: manifestAction(manifestSnapshot), BeforeSHA256: manifestSnapshot.SHA256, AfterSHA256: installfs.Hash(manifestData), Owned: manifestSnapshot.Exists, targetPath: manifestSnapshot.Path})
		targetPlan.checks = append(targetPlan.checks, installfs.Change{Path: manifestSnapshot.Path, Before: manifestSnapshot, Content: manifestData})
	}
	if targetPlan.Install != nil && targetPlan.Install.Mode == InstallModeNamedProfile {
		guardUnchangedConfig(&targetPlan, config)
	}
	targetPlan.changes = changes
	if len(changes) == 0 {
		targetPlan.Status = StatusNoop
	} else {
		targetPlan.Status = StatusReady
	}
	targetPlan.ManifestPath = manifestSnapshot.Path
	return targetPlan
}

func planFiles(request Request, target TargetRequest, patch Patch, ownership Manifest, config installfs.Snapshot, targetPlan *TargetPlan) ([]installfs.Change, bool) {
	changes := make([]installfs.Change, 0, len(patch.Files))
	seen := make(map[string]struct{}, len(patch.Files))
	planned := fieldsByPath(patch.Fields)
	for _, file := range patch.Files {
		path, err := patchPath(target.ConfigPath, file.Path)
		if err != nil {
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.patch_path_invalid", file.Path, err.Error(), 0, 0)
			return nil, true
		}
		if _, found := seen[path]; found {
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.patch_duplicate", filepath.Base(path), "adapter returned duplicate file effects", 0, 0)
			return nil, true
		}
		seen[path] = struct{}{}
		if file.Delete && len(file.Content) > 0 {
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.patch_delete_content", filepath.Base(path), "delete patches must not include replacement content", 0, 0)
			return nil, true
		}
		if !file.Delete {
			if err := installfs.ValidateContent(file.Content); err != nil {
				targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.patch_too_large", filepath.Base(path), err.Error(), 0, 0)
				return nil, true
			}
		}
		before := config
		if path != config.Path {
			var err error
			before, err = installfs.SnapshotFile(path)
			if err != nil {
				targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.file_snapshot_failed", filepath.Base(path), err.Error(), 0, 0)
				return nil, true
			}
		}
		afterHash := ""
		if !file.Delete {
			afterHash = installfs.Hash(file.Content)
		}
		ownedHash, owned := ownedBaseline(file, ownership, path, before, planned)
		action, conflict := fileAction(request, (patch.OverrideAllowed || file.Adoptable) && !file.NoOverride, before, ownedHash, owned, afterHash, file.Delete)
		if file.Release && action == ActionUpdate && !conflict {
			action = ActionRestore
		}
		if action == ActionNoop && file.Mode != 0 && before.Mode.Perm() != file.Mode {
			action = ActionUpdate
		}
		if action == ActionNoop && !file.LiveFields {
			markFieldsUnchanged(targetPlan.Fields, file.Fields)
		}
		fields := fieldsForNames(file.Fields)
		ownedTags := append(fileOwnership(file, ownership, path, config.Path, before), writtenMarkers(file, planned)...)
		targetPlan.Files = append(targetPlan.Files, FilePlan{Path: filePlanName(file, path), Action: action, BeforeSHA256: before.SHA256, AfterSHA256: afterHash, Owned: owned, Fields: fields, Delete: file.Delete, targetPath: path, ownership: ownedTags, release: file.Release})
		change := installfs.Change{Path: path, Before: before, Content: append([]byte(nil), file.Content...), Delete: file.Delete, Mode: file.Mode, RemoveEmptyDirs: file.RemoveEmptyDirs}
		targetPlan.checks = append(targetPlan.checks, change)
		addFileDiagnostic(targetPlan, filepath.Base(path), action, conflict, file.Delete)
		if conflict {
			continue
		}
		if action != ActionNoop && (!file.Delete || before.Exists) {
			changes = append(changes, change)
		}
	}
	sort.Slice(targetPlan.Files, func(i, j int) bool { return targetPlan.Files[i].Path < targetPlan.Files[j].Path })
	return changes, targetPlan.Diagnostics.HasErrors()
}

// markFieldsUnchanged records a byte-identical whole file's fields as already holding
// their planned values; whole-file adapters render from scratch and leave Before empty.
func markFieldsUnchanged(fields []FieldChange, names []string) {
	for index := range fields {
		if fields[index].Before == "" && slices.Contains(names, fields[index].Path) {
			fields[index].Before = fields[index].After
		}
	}
}

// filePlanName is the plan's name for a patched file: its label, else its base name.
func filePlanName(file FilePatch, path string) string {
	if file.Label != "" {
		return file.Label
	}
	return filepath.Base(path)
}

func fileAction(request Request, overrideAllowed bool, before installfs.Snapshot, ownedHash string, owned bool, afterHash string, deleteFile bool) (string, bool) {
	if deleteFile {
		if !before.Exists {
			return ActionDelete, false
		}
		if owned && ownedHash != "" && before.SHA256 == ownedHash {
			return ActionDelete, false
		}
		return ActionDelete, true
	}
	if before.Exists && before.SHA256 == afterHash {
		return ActionNoop, false
	}
	if !before.Exists {
		return ActionCreate, false
	}
	if owned && before.SHA256 == ownedHash {
		return ActionUpdate, false
	}
	if request.Override && overrideAllowed {
		return ActionOverride, false
	}
	if !owned && overrideAllowed {
		return ActionAdopt, !request.Backup
	}
	return ActionUpdate, true
}

func conflictReason(diagnostics profilemango.Diagnostics) string {
	for _, diagnostic := range diagnostics {
		switch diagnostic.Code {
		case "install.adopt_requires_backup":
			return "adopting an existing file that profile-mango does not own requires a backup; remove --no-backup"
		case "install.skill_destination_newer":
			return "an unmanaged skill file is newer than the profile's copy; port its edits into the profile, or pass --override to replace it"
		}
	}
	return "one or more target files conflict with unowned or edited state"
}

// addFileDiagnostic explains an adoption backup or a file conflict for one planned file.
func addFileDiagnostic(targetPlan *TargetPlan, name, action string, conflict, deleteFile bool) {
	switch {
	case action == ActionAdopt && conflict:
		targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.adopt_requires_backup", name, "adopting an existing file that profile-mango does not own requires a backup; remove --no-backup", 0, 0)
	case action == ActionAdopt:
		targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.adopt_backup", name, "existing "+name+" is not managed by profile-mango yet; it will be backed up before the first managed change, and unrelated settings are kept", 0, 0)
	case conflict && deleteFile:
		targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.file_conflict", name, "deletion requires an unchanged profile-mango-owned file and cannot be overridden", 0, 0)
	case conflict:
		targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.file_conflict", name, "target file is edited or unowned; use an adapter-approved override only when the effect is understood", 0, 0)
	}
}

func ownershipHash(manifest Manifest, path string) (string, bool) {
	for _, file := range manifest.Files {
		if file.Path == path {
			return file.SHA256, true
		}
	}
	return "", false
}

func nextManifest(ownership Manifest, snapshot installfs.Snapshot, target TargetRequest, profile string, patch Patch, files []FilePlan, changes []installfs.Change) (*Manifest, []byte, error) {
	if snapshot.Exists && ownership.Owner != "profile-mango" {
		return nil, nil, fmt.Errorf("manifest owner %q is not profile-mango", ownership.Owner)
	}
	manifest := ownership
	if !snapshot.Exists {
		manifest = Manifest{APIVersion: ManifestAPIVersion, Kind: ManifestKind, Owner: "profile-mango", Generation: 1}
	}
	manifest.APIVersion = ManifestAPIVersion
	manifest.Kind = ManifestKind
	manifest.Owner = "profile-mango"
	// A reinstall that changes no file keeps the recorded profile, so it stays a no-op.
	if len(changes) > 0 || !snapshot.Exists {
		manifest.Profile = profile
	}
	manifest.Target = target.Target
	manifest.Fields = nil
	if !target.Agent.Empty() {
		manifest.Fields = []string{"agent:" + target.Agent.Mode + ":" + target.Agent.Name}
	}
	manifest.PlanID = ""
	if len(changes) > 0 && snapshot.Exists {
		manifest.Generation++
	}
	originalFiles := append([]ManifestFile(nil), manifest.Files...)
	for _, file := range files {
		actualPath := file.targetPath
		if actualPath == "" || actualPath == snapshot.Path {
			continue
		}
		if file.Delete || file.release {
			manifest.Files = removeManifestFile(manifest.Files, actualPath)
			continue
		}
		if file.Action == ActionNoop {
			priorHash, owned := ownershipHash(ownership, actualPath)
			// A whole owned file already holding the profile's bytes is claimed as is.
			if (!owned || priorHash != file.BeforeSHA256) && wholeFileKind(file.ownership) == "" {
				continue
			}
		}
		manifest.Files = replaceManifestFile(manifest.Files, ManifestFile{Path: actualPath, SHA256: file.AfterSHA256, Fields: manifestFields(file)})
	}
	if snapshot.Exists && len(changes) == 0 && !manifestFilesEqual(originalFiles, manifest.Files) {
		manifest.Generation++
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encode ownership manifest: %w", err)
	}
	return &manifest, append(data, '\n'), nil
}

func replaceManifestFile(files []ManifestFile, value ManifestFile) []ManifestFile {
	result := make([]ManifestFile, 0, len(files)+1)
	found := false
	for _, file := range files {
		if file.Path == value.Path {
			result = append(result, value)
			found = true
			continue
		}
		result = append(result, file)
	}
	if !found {
		result = append(result, value)
	}
	return result
}

func removeManifestFile(files []ManifestFile, path string) []ManifestFile {
	result := make([]ManifestFile, 0, len(files))
	for _, file := range files {
		if file.Path != path {
			result = append(result, file)
		}
	}
	return result
}

func manifestFields(file FilePlan) []string {
	names := fieldNames(file.Fields)
	seen := make(map[string]struct{}, len(names)+len(file.ownership))
	for _, name := range names {
		seen[name] = struct{}{}
	}
	for _, name := range file.ownership {
		if _, found := seen[name]; found {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func manifestFilesEqual(left, right []ManifestFile) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Path != right[index].Path || left[index].SHA256 != right[index].SHA256 || strings.Join(left[index].Fields, "\x00") != strings.Join(right[index].Fields, "\x00") {
			return false
		}
	}
	return true
}

func decodeManifest(data []byte, target Target) (Manifest, error) {
	if len(data) == 0 {
		return Manifest{}, nil
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode ownership manifest: %w", err)
	}
	if manifest.APIVersion != ManifestAPIVersion || manifest.Kind != ManifestKind {
		return Manifest{}, fmt.Errorf("ownership manifest version or kind is unsupported")
	}
	// A manifest recorded by an earlier qualified version of the same target stays
	// valid; the next apply rewrites it at the current version.
	if manifest.Target.Name != "" && manifest.Target.Name != target.Name {
		return Manifest{}, fmt.Errorf("ownership manifest target %s does not match %s", manifest.Target.String(), target.String())
	}
	return manifest, nil
}

func patchPath(configPath, patchPath string) (string, error) {
	if patchPath == "" {
		abs, err := filepath.Abs(configPath)
		return filepath.Clean(abs), err
	}
	if filepath.IsAbs(patchPath) {
		return filepath.Clean(patchPath), nil
	}
	clean := filepath.Clean(patchPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("relative patch path escapes the config directory")
	}
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(configPath), clean))
	return filepath.Clean(abs), err
}

func blockedTarget(request TargetRequest, metadata AdapterMetadata, reason, code string) TargetPlan {
	targetPlan := TargetPlan{Target: request.Target, Metadata: metadata, Status: StatusBlocked, Reason: reason, ConfigPath: request.ConfigPath, ManifestPath: request.ManifestPath}
	targetPlan.Diagnostics.Add(profilemango.SeverityError, code, "target", reason, 0, 0)
	return targetPlan
}

func blockedTargetPlan(targetPlan TargetPlan, reason, code string) TargetPlan {
	targetPlan.Status = StatusBlocked
	targetPlan.Reason = reason
	targetPlan.Diagnostics.Add(profilemango.SeverityError, code, "target", reason, 0, 0)
	return targetPlan
}

func manifestAction(snapshot installfs.Snapshot) string {
	if snapshot.Exists {
		return ActionUpdate
	}
	return ActionCreate
}

func publicFields(fields []FieldChange) []FieldChange {
	result := append([]FieldChange(nil), fields...)
	for index := range result {
		result[index] = result[index].public()
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func fieldsForNames(names []string) []FieldChange {
	result := make([]FieldChange, 0, len(names))
	for _, name := range names {
		result = append(result, FieldChange{Path: name})
	}
	return result
}

func fieldNames(fields []FieldChange) []string {
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		result = append(result, field.Path)
	}
	sort.Strings(result)
	return result
}

func aggregateStatus(targets []TargetPlan) string {
	if len(targets) == 0 || allSkipped(targets) {
		return StatusBlocked
	}
	allNoop := true
	for _, target := range targets {
		if target.Status == StatusSkipped {
			continue
		}
		if target.Status != StatusReady && target.Status != StatusNoop {
			return StatusBlocked
		}
		if target.Status != StatusNoop {
			allNoop = false
		}
	}
	if allNoop {
		return StatusNoop
	}
	return StatusReady
}

func planID(plan Plan) (string, error) {
	type identity struct {
		Profile     string
		InputSHA256 string
		Backup      bool
		Override    bool
		Strict      bool
		Targets     []TargetPlan
	}
	identityPlan := identity{Profile: plan.Profile, InputSHA256: plan.InputSHA256, Backup: plan.Backup, Override: plan.Override, Strict: plan.Strict, Targets: append([]TargetPlan(nil), plan.Targets...)}
	for index := range identityPlan.Targets {
		identityPlan.Targets[index].ConfigPath = ""
		identityPlan.Targets[index].Config = nil
		identityPlan.Targets[index].ManifestPath = ""
		identityPlan.Targets[index].changes = nil
		identityPlan.Targets[index].checks = nil
	}
	return hashValue(identityPlan)
}

func validateSources(plan Plan) error {
	for _, source := range plan.sources {
		current, err := installfs.SnapshotFile(source.Path)
		if err != nil {
			return fmt.Errorf("revalidate %s: %w", filepath.Base(source.Path), err)
		}
		if !source.Snapshot.Equal(current) {
			return fmt.Errorf("source %s is stale: %w", filepath.Base(source.Path), installfs.ErrStale)
		}
	}
	return nil
}

func safeResourcePath(value string) (string, error) {
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("resource path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsRune(clean, '\x00') {
		return "", fmt.Errorf("resource path must remain beneath the package root")
	}
	return clean, nil
}

func safeName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\\x00")
}

func validateAgentDestination(target TargetRequest) error {
	if target.Agent.Empty() {
		return nil
	}
	if target.Target != (Target{Name: "opencode", Version: "1.18.31"}) {
		return fmt.Errorf("named agents require exact opencode@1.18.31")
	}
	if target.Agent.Mode != "primary" && target.Agent.Mode != "subagent" {
		return fmt.Errorf("agent mode must be primary or subagent")
	}
	name := target.Agent.Name
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		return fmt.Errorf("agent name must be simple lowercase letters, digits, or hyphens")
	}
	for _, char := range name {
		letter := char >= 'a' && char <= 'z'
		digit := char >= '0' && char <= '9'
		if !letter && !digit && char != '-' {
			return fmt.Errorf("agent name must be simple lowercase letters, digits, or hyphens")
		}
	}
	for _, reserved := range []string{"build", "plan", "general", "explore", "compaction", "title", "summary"} {
		if name == reserved {
			return fmt.Errorf("agent name %q is a reserved OpenCode built-in", name)
		}
	}
	if filepath.Base(target.ConfigPath) != name+".md" || filepath.Base(filepath.Dir(target.ConfigPath)) != "agents" {
		return fmt.Errorf("named agent --config path must end in agents/%s.md", name)
	}
	return nil
}

func sameAgentOwnership(manifest Manifest, agent AgentDestination) bool {
	if agent.Empty() {
		return len(manifest.Fields) == 0
	}
	return len(manifest.Fields) == 1 && manifest.Fields[0] == "agent:"+agent.Mode+":"+agent.Name
}

func namedAgentFilesMatch(manifest Manifest, path string) bool {
	return len(manifest.Files) == 1 && manifest.Files[0].Path == path
}
