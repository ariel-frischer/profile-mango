package install

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
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
	plan := Plan{APIVersion: PlanAPIVersion, Kind: PlanKind, Profile: request.ProfileName, Backup: request.Backup, Override: request.Override, InputSHA256: loaded.InputSHA256}
	plan.sources = loaded.Sources
	for _, targetRequest := range targets {
		targetPlan := planTarget(request, registry, targetRequest, loaded)
		plan.Targets = append(plan.Targets, targetPlan)
		plan.Diagnostics = append(plan.Diagnostics, targetPlan.Diagnostics...)
	}
	plan.Diagnostics = append(plan.Diagnostics, diagnostics...)
	plan.Status = aggregateStatus(plan.Targets)
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
	}{Profile: resolved, Route: route, Resources: digest})
	if err != nil {
		return loadedInput{}, diagnostics, fmt.Errorf("hash install inputs: %w", err)
	}
	return loadedInput{Profile: resolved, Route: route, Resources: resources, InputSHA256: inputHash, Sources: sources}, diagnostics.Sorted(), nil
}

// missingBindingsError names the exact fix for a missing local bindings file:
// copy the starter example scaffolded by "profile-mango init", or run init
// again in a fresh package.
func missingBindingsError(path string) error {
	example := filepath.Join(filepath.Dir(path), "local.example.yaml")
	return fmt.Errorf("bindings file does not exist; create it with: cp %s %s, or run: profile-mango init", example, path)
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
	refs := make([]reference, 0, len(profile.Instructions)+len(profile.Skills))
	for _, path := range profile.Instructions {
		refs = append(refs, reference{path: path, kind: "instruction"})
	}
	for _, path := range profile.Skills {
		refs = append(refs, reference{path: path, kind: "skill"})
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
	patch, err := adapter.Plan(AdapterInput{Target: targetRequest.Target, Agent: targetRequest.Agent, ConfigPath: config.Path, ManifestPath: manifestSnapshot.Path, Profile: loaded.Profile, Route: loaded.Route.For(targetRequest.Target.Name), Resources: loaded.Resources, Config: snapshotFromFS(config), Manifest: snapshotFromFS(manifestSnapshot), Ownership: ownership, HasManifest: manifestSnapshot.Exists, Override: request.Override})
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("adapter planning failed: %v", err), "install.adapter_plan_failed")
	}
	targetPlan.Diagnostics = append(targetPlan.Diagnostics, patch.Diagnostics...)
	targetPlan.Fields = publicFields(patch.Fields)
	changes, blocked := planFiles(request, targetRequest, patch, ownership, config, &targetPlan)
	if blocked {
		targetPlan.Status = StatusConflict
		targetPlan.Reason = "one or more target files conflict with unowned or edited state"
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
		ownedHash, owned := ownershipHash(ownership, path)
		action, conflict := fileAction(request, patch.OverrideAllowed && !file.NoOverride, before, ownedHash, owned, afterHash, file.Delete)
		fields := fieldsForNames(file.Fields)
		targetPlan.Files = append(targetPlan.Files, FilePlan{Path: filepath.Base(path), Action: action, BeforeSHA256: before.SHA256, AfterSHA256: afterHash, Owned: owned, Fields: fields, Delete: file.Delete, targetPath: path, ownership: append([]string(nil), file.Ownership...)})
		targetPlan.checks = append(targetPlan.checks, installfs.Change{Path: path, Before: before, Content: append([]byte(nil), file.Content...), Delete: file.Delete})
		if conflict {
			message := "target file is edited or unowned; use an adapter-approved override only when the effect is understood"
			if file.Delete {
				message = "deletion requires an unchanged profile-mango-owned file and cannot be overridden"
			}
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.file_conflict", filepath.Base(path), message, 0, 0)
			continue
		}
		if action != ActionNoop && (!file.Delete || before.Exists) {
			changes = append(changes, installfs.Change{Path: path, Before: before, Content: append([]byte(nil), file.Content...), Delete: file.Delete})
		}
	}
	sort.Slice(targetPlan.Files, func(i, j int) bool { return targetPlan.Files[i].Path < targetPlan.Files[j].Path })
	return changes, targetPlan.Diagnostics.HasErrors()
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
	return ActionUpdate, true
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
	manifest.Profile = profile
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
		if file.Delete {
			manifest.Files = removeManifestFile(manifest.Files, actualPath)
			continue
		}
		if file.Action == ActionNoop {
			priorHash, owned := ownershipHash(ownership, actualPath)
			if !owned || priorHash != file.BeforeSHA256 {
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
	if manifest.Target.Name != "" && manifest.Target != target {
		return Manifest{}, fmt.Errorf("ownership manifest target does not match %s", target.String())
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
	if len(targets) == 0 {
		return StatusBlocked
	}
	allNoop := true
	for _, target := range targets {
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
		Targets     []TargetPlan
	}
	identityPlan := identity{Profile: plan.Profile, InputSHA256: plan.InputSHA256, Backup: plan.Backup, Override: plan.Override, Targets: append([]TargetPlan(nil), plan.Targets...)}
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
