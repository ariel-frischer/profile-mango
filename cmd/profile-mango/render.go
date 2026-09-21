package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/staging"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type renderOptions struct {
	profiles      string
	resourceRoot  string
	bindings      string
	target        string
	targetVersion string
	out           string
	preview       bool
	jsonOutput    bool
	usesHome      bool
}

func newRenderCmd() *cobra.Command {
	var options renderOptions
	cmd := &cobra.Command{
		Use:          "render <name>",
		Short:        "Render an inert, version-qualified candidate without accessing agent homes",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(cmd, args[0], options)
		},
	}
	cmd.Flags().StringVar(&options.profiles, "profiles", "", "profile repository root (defaults to <home>/profiles)")
	cmd.Flags().StringVar(&options.resourceRoot, "resource-root", "", "resource package root (defaults to <home>)")
	cmd.Flags().StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	cmd.Flags().StringVar(&options.target, "target", "", "exact target adapter name")
	cmd.Flags().StringVar(&options.targetVersion, "target-version", "", "exact target version")
	cmd.Flags().StringVar(&options.out, "out", "", "new explicit staging directory")
	cmd.Flags().BoolVar(&options.preview, "preview", false, "write inert preview artifacts despite applicability blockers")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit a stable render report")
	return cmd
}

func runRender(cmd *cobra.Command, name string, options renderOptions) error {
	resolvedOptions, err := resolveRenderInputs(options)
	if err != nil {
		return err
	}
	options = resolvedOptions
	if err := validateRenderOptions(options); err != nil {
		return err
	}
	target := codex.TargetBuild{Name: options.target, Version: options.targetVersion, EvidenceSHA256: codex.EvidenceSHA256}
	result := codex.NewResult(name, target)
	profiles, diagnostics := loadSelectedProfiles(options.profiles, name)
	bindingsData, err := os.ReadFile(options.bindings)
	if err != nil {
		if options.usesHome && os.IsNotExist(err) {
			example := filepath.Join(filepath.Dir(options.bindings), "local.example.yaml")
			return fmt.Errorf("read global bindings %s (copy %s to this path first): %w", options.bindings, example, err)
		}
		return fmt.Errorf("read bindings %s: %w", options.bindings, err)
	}
	bindings, bindingDiagnostics := profilemango.ParseBindings(bindingsData)
	diagnostics = append(diagnostics, bindingDiagnostics...)
	_, found := profiles[name]
	if !found {
		diagnostics.Add(profilemango.SeverityError, "render.profile_missing", "profile", "requested profile is not present in the selected repository", 0, 0)
	}
	var resolved profilemango.ResolvedProfile
	var route profilemango.RouteBinding
	var resources []codex.Resource
	if found {
		var resolveDiagnostics profilemango.Diagnostics
		resolved, resolveDiagnostics = profilemango.Resolve(profiles, name)
		diagnostics = append(diagnostics, resolveDiagnostics...)
		if _, exists := bindings.Routes[resolved.RouteRef]; !exists {
			diagnostics.Add(profilemango.SeverityError, "binding.route_missing", "spec.routeRef", "routeRef is not present in bindings", 0, 0)
		} else {
			route = bindings.Routes[resolved.RouteRef]
		}
		if !diagnostics.HasErrors() {
			resourceDigests, resourceDiagnostics := profilemango.DigestResources(options.resourceRoot, resolved)
			diagnostics = append(diagnostics, resourceDiagnostics...)
			if !diagnostics.HasErrors() {
				resources, diagnostics = readResourceContents(options.resourceRoot, resourceDigests, diagnostics)
			}
		}
	}
	if diagnostics.HasErrors() {
		result.Diagnostics = diagnostics.Sorted()
		return finishRender(cmd, result, options, false)
	}
	result = codex.Render(codex.Input{Profile: resolved, Route: route, Resources: resources, Target: target})
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.Diagnostics = result.Diagnostics.Sorted()
	return finishRender(cmd, result, options, true)
}

func resolveRenderInputs(options renderOptions) (renderOptions, error) {
	values := []string{options.profiles, options.resourceRoot, options.bindings}
	explicit := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			explicit++
		}
	}
	if explicit == len(values) {
		return options, nil
	}
	if explicit != 0 {
		return renderOptions{}, fmt.Errorf("set --profiles, --resource-root, and --bindings together, or omit all three to use the profile home")
	}
	home, err := selectedHome()
	if err != nil {
		return renderOptions{}, err
	}
	options.profiles = filepath.Join(home, "profiles")
	options.resourceRoot = home
	options.bindings = filepath.Join(home, "bindings", "local.yaml")
	options.usesHome = true
	return options, nil
}

func validateRenderOptions(options renderOptions) error {
	values := []struct {
		flag  string
		value string
	}{
		{flag: "--target", value: options.target},
		{flag: "--target-version", value: options.targetVersion},
		{flag: "--out", value: options.out},
	}
	for _, item := range values {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("%s is required", item.flag)
		}
	}
	return nil
}

func loadSelectedProfiles(root, name string) (map[string]profilemango.PolicyProfile, profilemango.Diagnostics) {
	profiles := make(map[string]profilemango.PolicyProfile)
	var diagnostics profilemango.Diagnostics
	if !safeProfileName(name) {
		diagnostics.Add(profilemango.SeverityError, "repository.profile_name_invalid", "profile", "profile name must be a simple name", 0, 0)
		return profiles, diagnostics
	}
	loadProfileChain(root, name, profiles, &diagnostics)
	return profiles, diagnostics.Sorted()
}

func loadProfileChain(root, name string, profiles map[string]profilemango.PolicyProfile, diagnostics *profilemango.Diagnostics) {
	if _, found := profiles[name]; found || !safeProfileName(name) {
		return
	}
	data, err := os.ReadFile(filepath.Join(root, name, "profile.yaml"))
	if err != nil {
		if os.IsNotExist(err) && len(profiles) > 0 {
			return
		}
		diagnostics.Add(profilemango.SeverityError, "repository.read", name, err.Error(), 0, 0)
		return
	}
	profile, found := profilemango.ParseProfile(data)
	for _, diagnostic := range found {
		diagnostic.Path = name + "." + diagnostic.Path
		*diagnostics = append(*diagnostics, diagnostic)
	}
	if profile.Metadata.Name != "" && profile.Metadata.Name != name {
		diagnostics.Add(profilemango.SeverityError, "repository.name_mismatch", name, "folder name must match metadata.name", 0, 0)
	}
	profiles[name] = profile
	if profile.Spec.Extends != "" {
		loadProfileChain(root, profile.Spec.Extends, profiles, diagnostics)
	}
}

func safeProfileName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\")
}

func readResourceContents(root string, digests []profilemango.ResourceDigest, diagnostics profilemango.Diagnostics) ([]codex.Resource, profilemango.Diagnostics) {
	resources := make([]codex.Resource, 0, len(digests))
	for _, digest := range digests {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(digest.Path)))
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, "resource.read", digest.Path, err.Error(), 0, 0)
			continue
		}
		resources = append(resources, codex.Resource{Digest: digest, Content: data})
	}
	return resources, diagnostics
}

func finishRender(cmd *cobra.Command, result codex.Result, options renderOptions, canStage bool) error {
	result = result.Report(options.preview)
	if canStage && options.preview {
		if err := stageRender(options.out, result); err != nil {
			result.Diagnostics.Add(profilemango.SeverityError, "staging.write", "out", "unable to atomically create the explicit staging directory", 0, 0)
			result.Diagnostics = result.Diagnostics.Sorted()
			if err := writeRenderOutput(cmd, result, options.jsonOutput); err != nil {
				return err
			}
			return fmt.Errorf("render staging failed")
		}
	}
	if err := writeRenderOutput(cmd, result, options.jsonOutput); err != nil {
		return err
	}
	if result.Diagnostics.HasErrors() {
		return fmt.Errorf("render blocked")
	}
	if !options.preview {
		return fmt.Errorf("render requires --preview for inert output")
	}
	return nil
}

func stageRender(destination string, result codex.Result) error {
	report, err := result.ReportJSON(true)
	if err != nil {
		return err
	}
	files := []staging.File{{Path: "render.json", Content: report}}
	for _, artifact := range result.Artifacts {
		files = append(files, staging.File{Path: artifact.Path, Content: artifact.Content})
	}
	return staging.Write(destination, files)
}

func writeRenderOutput(cmd *cobra.Command, result codex.Result, jsonOutput bool) error {
	resultJSON, err := result.ReportJSON(result.Preview)
	if err != nil {
		return err
	}
	if jsonOutput {
		if _, err := cmd.OutOrStdout().Write(resultJSON); err != nil {
			return fmt.Errorf("write render report: %w", err)
		}
	}
	for _, diagnostic := range result.Diagnostics.Sorted() {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), diagnosticLine(diagnostic)); err != nil {
			return fmt.Errorf("write diagnostic: %w", err)
		}
	}
	return nil
}
