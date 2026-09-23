package profilemango

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DigestResources validates and hashes instruction and skill resources beneath root.
func DigestResources(root string, profile ResolvedProfile) ([]ResourceDigest, Diagnostics) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		var diagnostics Diagnostics
		diagnostics.Add(SeverityError, "resource.root", "resources", err.Error(), 0, 0)
		return nil, diagnostics
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		var diagnostics Diagnostics
		diagnostics.Add(SeverityError, "resource.root", "resources", err.Error(), 0, 0)
		return nil, diagnostics
	}

	type resourceRef struct{ path, kind string }
	refs := make([]resourceRef, 0, len(profile.Instructions)+len(profile.Skills))
	for _, path := range profile.Instructions {
		refs = append(refs, resourceRef{path, "instruction"})
	}
	for _, path := range profile.Skills {
		refs = append(refs, resourceRef{path, "skill"})
	}

	seen := map[string]struct{}{}
	var diagnostics Diagnostics
	resources := make([]ResourceDigest, 0, len(refs))
	for _, ref := range refs {
		clean := filepath.Clean(ref.path)
		if filepath.IsAbs(ref.path) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			diagnostics.Add(SeverityError, "resource.path_escape", ref.path, "resource path must be relative and remain beneath the package root", 0, 0)
			continue
		}
		portable := filepath.ToSlash(clean)
		if _, duplicate := seen[portable]; duplicate {
			diagnostics.Add(SeverityError, "resource.duplicate", portable, "resource path is referenced more than once", 0, 0)
			continue
		}
		seen[portable] = struct{}{}

		candidate := filepath.Join(rootReal, clean)
		real, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			diagnostics.Add(SeverityError, "resource.read", portable, err.Error(), 0, 0)
			continue
		}
		rel, err := filepath.Rel(rootReal, real)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			diagnostics.Add(SeverityError, "resource.symlink_escape", portable, "resource resolves outside the package root", 0, 0)
			continue
		}
		info, err := os.Stat(real)
		if err != nil || !info.Mode().IsRegular() {
			diagnostics.Add(SeverityError, "resource.not_file", portable, "resource must resolve to a regular file", 0, 0)
			continue
		}
		data, err := os.ReadFile(real)
		if err != nil {
			diagnostics.Add(SeverityError, "resource.read", portable, err.Error(), 0, 0)
			continue
		}
		sum := sha256.Sum256(data)
		resources = append(resources, ResourceDigest{Path: portable, Kind: ref.kind, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))})
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	if diagnostics.HasErrors() {
		return nil, diagnostics.Sorted()
	}
	return resources, diagnostics.Sorted()
}

// BuildPlan binds an already resolved profile to its effective route for target.
func BuildPlan(profile ResolvedProfile, target string, bindings Bindings, resources []ResourceDigest) (Plan, error) {
	route, found := bindings.RouteFor(profile.RouteRef, target)
	if !found {
		return Plan{}, fmt.Errorf("route %q is not present in bindings", profile.RouteRef)
	}
	return Plan{APIVersion: PlanVersion, Kind: "Plan", Profile: profile.Metadata.Name, Target: target, Route: route, Resources: append([]ResourceDigest(nil), resources...)}, nil
}
