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
			err = fmt.Errorf("bindings file does not exist")
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
	profile, foundDiagnostics := profilemango.ParseProfile(snapshot.Content)
	for _, diagnostic := range foundDiagnostics {
		diagnostic.Path = name + "." + diagnostic.Path
		*diagnostics = append(*diagnostics, diagnostic)
	}
	profiles[name] = profile
	*sources = append(*sources, sourceCheck{Path: snapshot.Path, Snapshot: snapshot})
	if profile.Spec.Extends != "" {
		return loadProfileChain(root, profile.Spec.Extends, profiles, diagnostics, sources)
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
	if !metadata.Installable {
		return blockedTargetPlan(targetPlan, metadata.Reason, "install.target.blocked")
	}
	if strings.TrimSpace(targetRequest.ConfigPath) == "" {
		return blockedTargetPlan(targetPlan, "an explicit synthetic or disposable config path is required", "install.config_path_required")
	}
	config, err := installfs.SnapshotFile(targetRequest.ConfigPath)
	if err != nil {
		return blockedTargetPlan(targetPlan, fmt.Sprintf("inspect config path: %v", err), "install.config_path_unsafe")
	}
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
	patch, err := adapter.Plan(AdapterInput{Target: targetRequest.Target, ConfigPath: config.Path, ManifestPath: manifestSnapshot.Path, Profile: loaded.Profile, Route: loaded.Route, Resources: loaded.Resources, Config: snapshotFromFS(config), Manifest: snapshotFromFS(manifestSnapshot), Ownership: ownership, HasManifest: manifestSnapshot.Exists, Override: request.Override})
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
		targetPlan.Files = append(targetPlan.Files, FilePlan{Path: filepath.Base(manifestSnapshot.Path), Action: manifestAction(manifestSnapshot), BeforeSHA256: manifestSnapshot.SHA256, AfterSHA256: installfs.Hash(manifestData), Owned: manifestSnapshot.Exists})
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
		if err := installfs.ValidateContent(file.Content); err != nil {
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.patch_too_large", filepath.Base(path), err.Error(), 0, 0)
			return nil, true
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
		afterHash := installfs.Hash(file.Content)
		ownedHash, owned := ownershipHash(ownership, path)
		action, conflict := fileAction(request, patch, before, ownedHash, owned, afterHash)
		fields := fieldsForNames(file.Fields)
		targetPlan.Files = append(targetPlan.Files, FilePlan{Path: filepath.Base(path), Action: action, BeforeSHA256: before.SHA256, AfterSHA256: afterHash, Owned: owned, Fields: fields})
		targetPlan.checks = append(targetPlan.checks, installfs.Change{Path: path, Before: before, Content: append([]byte(nil), file.Content...)})
		if conflict {
			targetPlan.Diagnostics.Add(profilemango.SeverityError, "install.file_conflict", filepath.Base(path), "target file is edited or unowned; use an adapter-approved override only when the effect is understood", 0, 0)
			continue
		}
		if action != ActionNoop {
			changes = append(changes, installfs.Change{Path: path, Before: before, Content: append([]byte(nil), file.Content...)})
		}
	}
	sort.Slice(targetPlan.Files, func(i, j int) bool { return targetPlan.Files[i].Path < targetPlan.Files[j].Path })
	return changes, targetPlan.Diagnostics.HasErrors()
}

func fileAction(request Request, patch Patch, before installfs.Snapshot, ownedHash string, owned bool, afterHash string) (string, bool) {
	if before.Exists && before.SHA256 == afterHash {
		return ActionNoop, false
	}
	if !before.Exists {
		return ActionCreate, false
	}
	if owned && before.SHA256 == ownedHash {
		return ActionUpdate, false
	}
	if request.Override && patch.OverrideAllowed {
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
	manifest.PlanID = ""
	if len(changes) > 0 && snapshot.Exists {
		manifest.Generation++
	}
	for _, file := range files {
		if file.Path == filepath.Base(snapshot.Path) {
			continue
		}
		actualPath := findChangePath(changes, file.Path)
		if actualPath == "" {
			continue
		}
		manifest.Files = replaceManifestFile(manifest.Files, ManifestFile{Path: actualPath, SHA256: file.AfterSHA256, Fields: fieldNames(file.Fields)})
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encode ownership manifest: %w", err)
	}
	return &manifest, append(data, '\n'), nil
}

func findChangePath(changes []installfs.Change, base string) string {
	for _, change := range changes {
		if filepath.Base(change.Path) == base {
			return change.Path
		}
	}
	return ""
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
