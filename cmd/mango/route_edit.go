package main

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/install"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

type routeShowJSON struct {
	Name      string                     `json:"name"`
	Bindings  string                     `json:"bindings"`
	Route     profilemango.RouteBinding  `json:"route"`
	Target    string                     `json:"target,omitempty"`
	Effective *profilemango.RouteBinding `json:"effective,omitempty"`
	Profiles  []string                   `json:"profiles"`
}

func runRouteShow(output io.Writer, options *routeOptions, name string) error {
	if options.target != "" && !slices.Contains(profilemango.RouteTargets, options.target) {
		return fmt.Errorf("unknown target %q; expected one of %s", options.target, strings.Join(profilemango.RouteTargets, ", "))
	}
	bindings, data, path, users, err := options.routeInputs()
	if err != nil {
		return err
	}
	route, found := bindings.Routes[name]
	if !found {
		return fmt.Errorf("route %q is not in %s; see mango route list", name, path)
	}
	show := routeShowJSON{Name: name, Bindings: path, Route: route, Target: options.target, Profiles: nonNil(users[name])}
	if options.target != "" {
		effective := route.For(options.target)
		show.Effective = &effective
	}
	if options.jsonOutput {
		return writeRouteJSON(output, show)
	}
	source, err := profilemango.RouteSource(data, name)
	if err != nil {
		return err
	}
	return writeRouteShow(output, show, source)
}

func writeRouteShow(output io.Writer, show routeShowJSON, source string) error {
	var text strings.Builder
	fmt.Fprintf(&text, "routes.%s in %s:\n%s", show.Name, show.Bindings, source)
	if show.Effective != nil {
		writeEffectiveRoute(&text, show.Target, *show.Effective)
	}
	if len(show.Profiles) == 0 {
		text.WriteString("\nNo profile uses this route.\n")
	} else {
		fmt.Fprintf(&text, "\nUsed by profiles: %s\n", strings.Join(show.Profiles, ", "))
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return fmt.Errorf("writing route: %w", err)
	}
	return nil
}

func writeEffectiveRoute(text *strings.Builder, target string, route profilemango.RouteBinding) {
	fmt.Fprintf(text, "\nWhat %s gets:\n", target)
	fmt.Fprintf(text, "  provider: %s\n  transport: %s\n  authentication: %s\n  model: %s\n  effort: %s\n",
		route.Provider, route.Transport, route.Authentication, route.Model, route.Effort)
	if route.SubagentMaxEffort != "" {
		fmt.Fprintf(text, "  subagentMaxEffort: %s\n", route.SubagentMaxEffort)
	}
	if len(route.Roles) == 0 {
		return
	}
	text.WriteString("  roles:\n")
	for _, name := range route.SortedRoleNames() {
		role := route.Roles[name]
		fmt.Fprintf(text, "    %s: %s %s", name, role.Provider, role.Model)
		if role.Effort != "" {
			fmt.Fprintf(text, ", effort %s", role.Effort)
		}
		text.WriteString("\n")
	}
}

// runRouteEdit applies one route set or unset to the bindings file: it prints
// the diff, then writes atomically unless this is a dry run or nothing changed.
func runRouteEdit(output io.Writer, options *routeOptions, edit profilemango.RouteEdit, verb string) error {
	bindingsPath, profilesRoot, err := options.paths()
	if err != nil {
		return err
	}
	snapshot, err := readBindingsFile(bindingsPath)
	if err != nil {
		return err
	}
	edited, err := profilemango.EditBindings(snapshot.Content, edit)
	if err != nil {
		return fmt.Errorf("%s: %w", bindingsPath, err)
	}
	if bytes.Equal(edited, snapshot.Content) {
		_, err = fmt.Fprintf(output, "No change: %s already says that.\n", bindingsPath)
		return err
	}
	if _, err := io.WriteString(output, install.UnifiedDiff(bindingsPath, "after route "+verb, snapshot.Content, edited)); err != nil {
		return fmt.Errorf("writing diff: %w", err)
	}
	if err := writeEffortWarning(output, edit); err != nil {
		return err
	}
	if options.dryRun {
		_, err = fmt.Fprintln(output, "Dry run: nothing was written.")
		return err
	}
	if err := installfs.ReplaceFile(bindingsPath, edited, snapshot); err != nil {
		return fmt.Errorf("write bindings %s: %w", bindingsPath, err)
	}
	return writeRouteApplyHint(output, bindingsPath, edit, profilesByRoute(profilesRoot)[edit.Route])
}

func writeRouteApplyHint(output io.Writer, bindingsPath string, edit profilemango.RouteEdit, profiles []string) error {
	var text strings.Builder
	fmt.Fprintf(&text, "Updated %s. Agent config is unchanged until you apply a profile.\n", bindingsPath)
	if len(profiles) == 0 {
		fmt.Fprintf(&text, "No profile uses route %s yet; set route: %s in a profile to use it.\n", edit.Route, edit.Route)
	} else {
		fmt.Fprintf(&text, "Profiles using route %s: %s\n", edit.Route, strings.Join(profiles, ", "))
		writeRouteReapply(&text, edit, profiles)
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return fmt.Errorf("writing hint: %w", err)
	}
	return nil
}

// writeRouteReapply names, per profile using the route, the install that re-applies it
// to exactly the agents whose ownership manifest records that profile, so no agent on
// another profile is switched. Manifests are read read-only at their default paths.
func writeRouteReapply(text *strings.Builder, edit profilemango.RouteEdit, profiles []string) {
	report, err := install.InspectStatus(install.StatusRequest{Registry: install.DefaultRegistry(), Env: install.OSPathEnv()})
	if err != nil {
		fmt.Fprintf(text, "Could not read ownership manifests (%v); see mango status.\n", err)
	}
	recorded := recordedTargets(report, edit.Target)
	for _, profile := range profiles {
		groups := recorded[profile]
		if len(groups[false]) == 0 && len(groups[true]) == 0 {
			target := cmp.Or(edit.Target, "<target>")
			fmt.Fprintf(text, "No agent records profile %s; to adopt one: %s\n", profile, install.ReapplyCommand(profile, []string{target}, false))
			continue
		}
		for _, makeDefault := range []bool{false, true} {
			if targets := groups[makeDefault]; len(targets) > 0 {
				fmt.Fprintf(text, "apply with: %s\n", install.ReapplyCommand(profile, targets, makeDefault))
			}
		}
	}
}

// recordedTargets groups managed agent names by recorded profile, then by whether the
// profile is the agent's default. A non-empty only keeps that agent.
func recordedTargets(report install.StatusReport, only string) map[string]map[bool][]string {
	result := map[string]map[bool][]string{}
	for _, status := range report.Targets {
		if status.State != install.StatusManaged || (only != "" && status.Target.Name != only) {
			continue
		}
		if result[status.Profile] == nil {
			result[status.Profile] = map[bool][]string{}
		}
		groups := result[status.Profile]
		if !slices.Contains(groups[status.OwnsConfig()], status.Target.Name) {
			groups[status.OwnsConfig()] = append(groups[status.OwnsConfig()], status.Target.Name)
		}
	}
	return result
}

// knownEfforts are the effort values at least one supported agent installs. Effort
// stays free-form (each agent checks its own set at install), so others only warn.
var knownEfforts = []string{"none", "off", "minimal", "low", "medium", "high", "xhigh", "max", "ultra", "auto"}

func writeEffortWarning(output io.Writer, edit profilemango.RouteEdit) error {
	effort, ok := edit.Set["effort"]
	if !ok || slices.Contains(knownEfforts, effort) {
		return nil
	}
	_, err := fmt.Fprintf(output, "warning: no supported agent uses effort %q (known: %s); install will skip or block it\n", effort, strings.Join(knownEfforts, ", "))
	return err
}
