package main

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
	"gitlab.com/ariel-frischer/profile-mango/pkg/install"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
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
		profile, target := "<profile>", "<target>"
		if len(profiles) == 1 {
			profile = profiles[0]
		}
		if edit.Target != "" {
			target = edit.Target
		}
		fmt.Fprintf(&text, "Profiles using route %s: %s\n", edit.Route, strings.Join(profiles, ", "))
		fmt.Fprintf(&text, "apply with: mango use %s\n        or: mango install %s --target %s\n", profile, profile, target)
	}
	if _, err := io.WriteString(output, text.String()); err != nil {
		return fmt.Errorf("writing hint: %w", err)
	}
	return nil
}
