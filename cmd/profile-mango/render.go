package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/internal/staging"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/jcodefork"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
	"gitlab.com/ariel-frischer/profile-mango/pkg/render"
)

type renderOptions struct {
	profiles      string
	resourceRoot  string
	bindings      string
	target        string
	targets       []string
	targetVersion string
	out           string
	preview       bool
	jsonOutput    bool
	usesHome      bool
}

type renderAdapter struct {
	evidenceSHA256 string
	newResult      func(string, render.TargetBuild) render.Result
	render         func(render.Input) render.Result
}

func renderAdapterFor(name string) (renderAdapter, bool) {
	switch name {
	case claudecode.TargetName:
		return renderAdapter{
			evidenceSHA256: claudecode.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return claudecode.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return claudecode.Render(input)
			},
		}, true
	case codex.TargetName:
		return renderAdapter{
			evidenceSHA256: codex.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return codex.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return codex.Render(input)
			},
		}, true
	case ohmypi.TargetName:
		return renderAdapter{
			evidenceSHA256: ohmypi.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return ohmypi.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return ohmypi.Render(input)
			},
		}, true
	case openclaw.TargetName:
		return renderAdapter{
			evidenceSHA256: openclaw.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return openclaw.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return openclaw.Render(input)
			},
		}, true
	case hermes.TargetName:
		return renderAdapter{
			evidenceSHA256: hermes.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return hermes.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return hermes.Render(input)
			},
		}, true
	case opencode.TargetName:
		return renderAdapter{
			evidenceSHA256: opencode.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return opencode.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return opencode.Render(input)
			},
		}, true
	case pi.TargetName:
		return renderAdapter{
			evidenceSHA256: pi.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return pi.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return pi.Render(input)
			},
		}, true
	case jcodefork.TargetName:
		return renderAdapter{
			evidenceSHA256: jcodefork.EvidenceSHA256,
			newResult: func(profileName string, target render.TargetBuild) render.Result {
				return jcodefork.NewResult(profileName, target)
			},
			render: func(input render.Input) render.Result {
				return jcodefork.Render(input)
			},
		}, true
	default:
		return renderAdapter{}, false
	}
}

func newRenderCmd() *cobra.Command {
	var options renderOptions
	cmd := &cobra.Command{
		Use:          "render <name>",
		Aliases:      []string{"r", "preview"},
		Short:        "Preview a profile's settings for one coding agent, without touching that agent's real files",
		Long:         "Build a preview of a profile's settings for one target and exact version. It never reads or writes the target's real home directory; output only goes to the folder given by --out.",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(cmd, args[0], options)
		},
	}
	cmd.Flags().StringVar(&options.profiles, "profiles", "", "profile repository root (defaults to <home>/profiles)")
	cmd.Flags().StringVar(&options.resourceRoot, "resource-root", "", "resource package root (defaults to <home>)")
	cmd.Flags().StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	cmd.Flags().StringArrayVarP(&options.targets, "target", "t", nil, "target[@version], comma-separated or repeated; render accepts exactly one (claude-code, codex, pi, oh-my-pi, openclaw, hermes, or opencode)")
	cmd.Flags().StringVar(&options.targetVersion, "target-version", "", "exact target version")
	cmd.Flags().StringVar(&options.out, "out", "", "new directory to write preview files into")
	cmd.Flags().BoolVar(&options.preview, "preview", false, "write preview files even if the target isn't fully supported yet")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit a stable render report")
	return cmd
}

func runRender(cmd *cobra.Command, name string, options renderOptions) error {
	values := options.targets
	if options.target != "" {
		values = append([]string{options.target}, values...)
	}
	selected, err := targetList(values)
	if err != nil {
		return err
	}
	selected, err = resolvedRenderTargets(selected, options.targetVersion)
	if err != nil {
		return err
	}
	if len(selected) > 1 {
		return fmt.Errorf("render accepts exactly one --target; run one preview per target with a separate --out")
	}
	if len(selected) == 1 {
		selector, err := install.ParseTargetSelector(selected[0])
		if err != nil {
			return err
		}
		options.target = selector.Name
		if selector.Version != "" {
			if options.targetVersion != "" && options.targetVersion != selector.Version {
				return fmt.Errorf("--target version conflicts with --target-version")
			}
			options.targetVersion = selector.Version
		}
	}
	resolvedOptions, err := resolveRenderInputs(options)
	if err != nil {
		return err
	}
	options = resolvedOptions
	if err := validateRenderOptions(options); err != nil {
		return err
	}
	adapter, found := renderAdapterFor(options.target)
	target := render.TargetBuild{Name: options.target, Version: options.targetVersion}
	if !found {
		return finishRender(cmd, render.UnknownTarget(name, target), options, false)
	}
	target.EvidenceSHA256 = adapter.evidenceSHA256
	result := adapter.newResult(name, target)
	profiles, diagnostics := loadSelectedProfiles(options.profiles, name)
	bindingsData, err := os.ReadFile(options.bindings)
	if err != nil {
		message := fmt.Sprintf("read bindings %s: %v", options.bindings, err)
		if options.usesHome && os.IsNotExist(err) {
			example := filepath.Join(filepath.Dir(options.bindings), "local.example.yaml")
			message = fmt.Sprintf("read global bindings %s (copy %s to this path first, or run: mango init): %v", options.bindings, example, err)
		}
		result.Diagnostics.Add(profilemango.SeverityError, "render.bindings.read", "bindings", message, 0, 0)
		result = result.Report(options.preview)
		if writeErr := writeRenderOutput(cmd, result, options.jsonOutput); writeErr != nil {
			return writeErr
		}
		return fmt.Errorf("%s", message)
	}
	bindings, bindingDiagnostics := profilemango.ParseBindings(bindingsData)
	diagnostics = append(diagnostics, bindingDiagnostics...)
	_, profileFound := profiles[name]
	if !profileFound {
		diagnostics.Add(profilemango.SeverityError, "render.profile_missing", "profile", "requested profile is not present in the selected repository", 0, 0)
	}
	var resolved profilemango.ResolvedProfile
	var route profilemango.RouteBinding
	var resources []render.Resource
	if profileFound {
		var resolveDiagnostics profilemango.Diagnostics
		resolved, resolveDiagnostics = profilemango.Resolve(profiles, name)
		diagnostics = append(diagnostics, resolveDiagnostics...)
		effective, exists := bindings.RouteFor(resolved.RouteRef, options.target)
		if !exists {
			diagnostics.Add(profilemango.SeverityError, "binding.route_missing", "route", "route is not present in bindings", 0, 0)
		}
		route = effective
		if !diagnostics.HasErrors() {
			resourceDigests, resourceDiagnostics := profilemango.DigestResources(options.resourceRoot, resolved)
			diagnostics = append(diagnostics, resourceDiagnostics...)
			if !diagnostics.HasErrors() {
				resources, diagnostics = readResourceContents(options.resourceRoot, resourceDigests, diagnostics)
			}
		}
	}
	if diagnostics.HasErrors() {
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		result.Diagnostics = result.Diagnostics.Sorted()
		return finishRender(cmd, result, options, false)
	}
	result = adapter.render(render.Input{Profile: resolved, Route: route, Resources: resources, Target: target})
	addUnrenderedRolesDiagnostic(&result, options.target, resolved, route)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.Diagnostics = result.Diagnostics.Sorted()
	return finishRender(cmd, result, options, true)
}

// addUnrenderedRolesDiagnostic reports role requirements a target preview omits:
// only Oh My Pi renders route roles (as modelRoles slots) and subagentMaxEffort
// (as task.maxEffort), and no target renders profile role definitions.
func addUnrenderedRolesDiagnostic(result *render.Result, target string, profile profilemango.ResolvedProfile, route profilemango.RouteBinding) {
	if count := len(profile.Roles); count > 0 {
		result.Diagnostics.Add(profilemango.SeverityWarning, "render.profile.role_definitions_unsupported", "roles", fmt.Sprintf("%d role definitions are not rendered for %s; no qualified target installs role descriptions or instructions", count, target), 0, 0)
	}
	if target == ohmypi.TargetName {
		return
	}
	if len(route.Roles) > 0 {
		result.Diagnostics.Add(profilemango.SeverityWarning, "render.route.roles_unsupported", "route.roles", fmt.Sprintf("%d route roles are not rendered for %s; only Oh My Pi renders per-role routes", len(route.Roles), target), 0, 0)
	}
	if route.SubagentMaxEffort != "" {
		result.Diagnostics.Add(profilemango.SeverityWarning, "render.route.subagent_max_effort_unsupported", "route.subagentMaxEffort", fmt.Sprintf("subagentMaxEffort is not rendered for %s; only Oh My Pi renders it (task.maxEffort)", target), 0, 0)
	}
}

func resolvedRenderTargets(values []string, version string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		selector, err := install.ParseTargetSelector(value)
		if err != nil {
			return nil, err
		}
		if selector.Version == "" {
			selector.Version = version
			if selector.Version == "" && selector.Name == jcodefork.TargetName {
				selector.Version = jcodefork.TargetVersion
			}
			if selector.Version == "" {
				if qualified, err := install.DefaultRegistry().ResolveTarget(selector.Name); err == nil {
					selector.Version = qualified.Version
				}
			}
		}
		if _, exists := seen[selector.String()]; !exists {
			seen[selector.String()] = struct{}{}
			result = append(result, selector.String())
		}
	}
	return result, nil
}

// bindingsCreateHint names the exact fix for a missing local bindings file:
// copy the starter example scaffolded by "mango init", or run init
// again in a fresh package.
func bindingsCreateHint(path string) string {
	example := filepath.Join(filepath.Dir(path), "local.example.yaml")
	return fmt.Sprintf("create it with: cp %s %s, or run: mango init", example, path)
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
	profile, found := profilemango.ParseProfileAt(data, name)
	for _, diagnostic := range found {
		diagnostic.Path = name + "." + diagnostic.Path
		*diagnostics = append(*diagnostics, diagnostic)
	}
	profiles[name] = profile
	if profile.Extends != "" {
		loadProfileChain(root, profile.Extends, profiles, diagnostics)
	}
}

func safeProfileName(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\")
}

func readResourceContents(root string, digests []profilemango.ResourceDigest, diagnostics profilemango.Diagnostics) ([]render.Resource, profilemango.Diagnostics) {
	resources := make([]render.Resource, 0, len(digests))
	for _, digest := range digests {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(digest.Path)))
		if err != nil {
			diagnostics.Add(profilemango.SeverityError, "resource.read", digest.Path, err.Error(), 0, 0)
			continue
		}
		resources = append(resources, render.Resource{Digest: digest, Content: data})
	}
	return resources, diagnostics
}

// finishRender writes the report and picks the exit status. A staged
// --preview succeeds even when the output is not applicable to the target:
// the preview files are inert, so each blocking finding is shown as a warning
// and the report keeps applicable:false with its original diagnostics.
func finishRender(cmd *cobra.Command, result render.Result, options renderOptions, canStage bool) error {
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
		return writeStagedPreview(cmd, result, options)
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

// writeStagedPreview reports a successfully staged preview: blocking
// findings print as warnings, and human output ends with where the preview
// went and whether it is applicable.
func writeStagedPreview(cmd *cobra.Command, result render.Result, options renderOptions) error {
	if options.jsonOutput {
		data, err := result.ReportJSON(result.Preview)
		if err != nil {
			return err
		}
		if _, err := cmd.OutOrStdout().Write(data); err != nil {
			return fmt.Errorf("write render report: %w", err)
		}
	}
	styles := stylesFor(cmd.ErrOrStderr(), !options.jsonOutput)
	for _, diagnostic := range result.Diagnostics {
		diagnostic.Severity = profilemango.SeverityWarning
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), styledDiagnosticLine(diagnostic, styles)); err != nil {
			return fmt.Errorf("write diagnostic: %w", err)
		}
	}
	if options.jsonOutput {
		return nil
	}
	summary := fmt.Sprintf("preview staged in %s; applicable to %s@%s: yes\n", options.out, result.Target, result.TargetVersion)
	if !result.Applicable {
		summary = fmt.Sprintf("preview staged in %s; applicable to %s@%s: no (%d blocking finding(s) above; details in render.json)\n", options.out, result.Target, result.TargetVersion, len(result.Diagnostics.Errors()))
	}
	if _, err := fmt.Fprint(cmd.OutOrStdout(), summary); err != nil {
		return fmt.Errorf("write render summary: %w", err)
	}
	return nil
}

func stageRender(destination string, result render.Result) error {
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

func writeRenderOutput(cmd *cobra.Command, result render.Result, jsonOutput bool) error {
	resultJSON, err := result.ReportJSON(result.Preview)
	if err != nil {
		return err
	}
	if jsonOutput {
		if _, err := cmd.OutOrStdout().Write(resultJSON); err != nil {
			return fmt.Errorf("write render report: %w", err)
		}
	}
	styles := stylesFor(cmd.ErrOrStderr(), !jsonOutput)
	for _, diagnostic := range result.Diagnostics.Sorted() {
		if _, err := fmt.Fprintln(cmd.ErrOrStderr(), styledDiagnosticLine(diagnostic, styles)); err != nil {
			return fmt.Errorf("write diagnostic: %w", err)
		}
	}
	return nil
}
