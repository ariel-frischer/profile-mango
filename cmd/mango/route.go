package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/ariel-frischer/profile-mango/internal/installfs"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/spf13/cobra"
)

type routeOptions struct {
	bindings, profiles string
	resourceRoot       string
	target, role       string
	dryRun, jsonOutput bool
}

// routeFieldFlags maps each route set flag and route unset field name to its YAML field.
var routeFieldFlags = []struct{ flag, field, usage string }{
	{"provider", "provider", "provider name, e.g. openai or anthropic"},
	{"model", "model", "model name"},
	{"effort", "effort", "reasoning effort, e.g. low, medium, or high"},
	{"subagent-max-effort", "subagentMaxEffort", "highest effort one subagent may ask for (whole route only)"},
}

func newRouteCmd() *cobra.Command {
	options := &routeOptions{}
	cmd := &cobra.Command{
		Use:   "route",
		Short: "List, show, and change the routes in your bindings file",
		Long: "A route is the provider, model, and effort a profile runs on. Routes live in <home>/bindings/local.yaml " +
			"and a profile picks one with route:. route set and route unset change that file in place, keep its comments " +
			"and layout, and check it before writing. They never touch agent config: run the mango install command they print after.",
		Args: cobra.NoArgs,
	}
	flags := cmd.PersistentFlags()
	flags.StringVar(&options.bindings, "bindings", "", "local route bindings file (defaults to <home>/bindings/local.yaml)")
	flags.StringVar(&options.profiles, "profiles", "", "profile repository root, to find profiles that use a route (defaults to <home>/profiles)")
	flags.StringVar(&options.resourceRoot, "resource-root", "", "resource package root, searched for the old value route set replaces (defaults to <home>)")
	cmd.AddCommand(newRouteListCmd(options), newRouteShowCmd(options), newRouteSetCmd(options), newRouteUnsetCmd(options))
	return cmd
}

func newRouteListCmd(options *routeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List each route with its provider, model, effort, and the profiles that use it",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRouteList(cmd.OutOrStdout(), options)
		},
	}
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the routes as JSON")
	return cmd
}

func newRouteShowCmd(options *routeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "show <route>",
		Short:        "Show a route as written; with --target, also what that agent gets",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRouteShow(cmd.OutOrStdout(), options, args[0])
		},
	}
	cmd.Flags().StringVarP(&options.target, "target", "t", "", "agent to resolve the route for, e.g. oh-my-pi")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "emit the route as JSON")
	return cmd
}

func newRouteSetCmd(options *routeOptions) *cobra.Command {
	values := make(map[string]*string, len(routeFieldFlags))
	cmd := &cobra.Command{
		Use:   "set <route>",
		Short: "Change fields of an existing route, for the whole route, one agent (--target), one role (--role), or one role for one agent (both)",
		Long: "Changes only the lines it needs, adding the targets.<agent>, roles.<role>, or targets.<agent>.roles.<role> entry when missing, " +
			"then checks the whole file the same way install does and prints the change. Nothing is written if the " +
			"result is invalid or unchanged. To add a new route, edit the file by hand.",
		Example:      "  mango route set sol --effort medium\n  mango route set sol --target oh-my-pi --model gpt-6-sol\n  mango route set opus55 --role research --effort high --dry-run\n  mango route set luna --target codex --role research --provider openai",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			edit := profilemango.RouteEdit{Route: args[0], Target: options.target, Role: options.role, Set: map[string]string{}}
			for _, item := range routeFieldFlags {
				if cmd.Flags().Changed(item.flag) {
					edit.Set[item.field] = *values[item.flag]
				}
			}
			if len(edit.Set) == 0 {
				return fmt.Errorf("pass at least one of --provider, --model, --effort, or --subagent-max-effort")
			}
			return runRouteEdit(cmd.OutOrStdout(), options, edit, "set")
		},
	}
	for _, item := range routeFieldFlags {
		values[item.flag] = cmd.Flags().String(item.flag, "", item.usage)
	}
	addRouteScopeFlags(cmd, options)
	return cmd
}

func newRouteUnsetCmd(options *routeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <route> <field>...",
		Short: "Remove fields from a route, one agent (--target), one role (--role), or one role for one agent (both)",
		Long: "Fields are provider, model, effort, and subagent-max-effort. A targets.<agent>, roles.<role>, or " +
			"targets.<agent>.roles.<role> entry left empty is removed, and so is each enclosing map left empty. The file is checked before writing, so fields " +
			"a route needs, such as its base model, cannot be removed.",
		Example:      "  mango route unset sol --target oh-my-pi effort\n  mango route unset opus55 subagent-max-effort",
		Args:         cobra.MinimumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			edit := profilemango.RouteEdit{Route: args[0], Target: options.target, Role: options.role}
			for _, name := range args[1:] {
				field, err := routeField(name)
				if err != nil {
					return err
				}
				edit.Unset = append(edit.Unset, field)
			}
			return runRouteEdit(cmd.OutOrStdout(), options, edit, "unset")
		},
	}
	addRouteScopeFlags(cmd, options)
	return cmd
}

func addRouteScopeFlags(cmd *cobra.Command, options *routeOptions) {
	flags := cmd.Flags()
	flags.StringVarP(&options.target, "target", "t", "", "change only this agent's override, e.g. oh-my-pi")
	flags.StringVar(&options.role, "role", "", "change only this subagent role: worker, planner, research, or tiny; with --target, only that agent's role")
	flags.BoolVar(&options.dryRun, "dry-run", false, "print the change without writing it")
}

func routeField(name string) (string, error) {
	names := make([]string, 0, len(routeFieldFlags))
	for _, item := range routeFieldFlags {
		if item.flag == name {
			return item.field, nil
		}
		names = append(names, item.flag)
	}
	return "", fmt.Errorf("unknown route field %q; expected one of %s", name, strings.Join(names, ", "))
}

// paths resolves the bindings file and profile root like install does, with
// each flag overriding its own default under the profile home.
func (options *routeOptions) paths() (string, string, error) {
	bindings, profiles := options.bindings, options.profiles
	if bindings != "" && profiles != "" {
		return bindings, profiles, nil
	}
	home, err := selectedHome()
	if err != nil {
		return "", "", err
	}
	if bindings == "" {
		bindings = filepath.Join(home, "bindings", "local.yaml")
	}
	if profiles == "" {
		profiles = filepath.Join(home, "profiles")
	}
	return bindings, profiles, nil
}

func readBindingsFile(path string) (installfs.Snapshot, error) {
	snapshot, err := installfs.SnapshotFile(path)
	if err != nil {
		return installfs.Snapshot{}, fmt.Errorf("read bindings %s: %w", path, err)
	}
	if !snapshot.Exists {
		return installfs.Snapshot{}, fmt.Errorf("read bindings %s: file does not exist (%s)", path, bindingsCreateHint(path))
	}
	return snapshot, nil
}

// routeInputs reads and strictly validates the bindings file for list and show.
func (options *routeOptions) routeInputs() (profilemango.Bindings, []byte, string, map[string][]string, error) {
	bindingsPath, profilesRoot, err := options.paths()
	if err != nil {
		return profilemango.Bindings{}, nil, "", nil, err
	}
	snapshot, err := readBindingsFile(bindingsPath)
	if err != nil {
		return profilemango.Bindings{}, nil, "", nil, err
	}
	bindings, diagnostics := profilemango.ParseBindings(snapshot.Content)
	if err := diagnostics.Err(); err != nil {
		return profilemango.Bindings{}, nil, "", nil, fmt.Errorf("bindings %s are invalid:\n%w", bindingsPath, err)
	}
	return bindings, snapshot.Content, bindingsPath, profilesByRoute(profilesRoot), nil
}

// profilesByRoute maps each route name to the profiles whose resolved route
// is that route. Profiles that do not resolve are left out.
func profilesByRoute(root string) map[string][]string {
	profiles, _ := profilemango.LoadProfiles(root)
	result := map[string][]string{}
	for name := range profiles {
		resolved, diagnostics := profilemango.Resolve(profiles, name)
		if diagnostics.HasErrors() {
			continue
		}
		result[resolved.RouteRef] = append(result[resolved.RouteRef], name)
	}
	for route := range result {
		slices.Sort(result[route])
	}
	return result
}

type routeListEntry struct {
	Name     string                    `json:"name"`
	Route    profilemango.RouteBinding `json:"route"`
	Profiles []string                  `json:"profiles"`
}

func runRouteList(output io.Writer, options *routeOptions) error {
	bindings, _, _, users, err := options.routeInputs()
	if err != nil {
		return err
	}
	entries := make([]routeListEntry, 0, len(bindings.Routes))
	for _, name := range slices.Sorted(maps.Keys(bindings.Routes)) {
		entries = append(entries, routeListEntry{Name: name, Route: bindings.Routes[name], Profiles: nonNil(users[name])})
	}
	if options.jsonOutput {
		return writeRouteJSON(output, entries)
	}
	table := tabwriter.NewWriter(output, 0, 2, 2, ' ', 0)
	_, _ = fmt.Fprintln(table, "ROUTE\tPROVIDER\tMODEL\tEFFORT\tPROFILES")
	for _, entry := range entries {
		profiles := strings.Join(entry.Profiles, ", ")
		if profiles == "" {
			profiles = "-"
		}
		_, _ = fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", entry.Name, entry.Route.Provider, entry.Route.Model, entry.Route.Effort, profiles)
	}
	if err := table.Flush(); err != nil {
		return fmt.Errorf("writing routes: %w", err)
	}
	return nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func writeRouteJSON(output io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	if _, err := fmt.Fprintf(output, "%s\n", data); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	return nil
}
