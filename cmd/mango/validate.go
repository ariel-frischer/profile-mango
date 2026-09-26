package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var bindingsPath string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "validate <profile.yaml>",
		Aliases: []string{"v"},
		Short:   "Check a profile.yaml file for errors, without touching any agent's files or the network",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileData, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read profile: %w", err)
			}
			profile, diagnostics := profilemango.ParseProfileAt(profileData, profileFolderName(args[0]))
			if bindingsPath != "" {
				bindingsData, readErr := os.ReadFile(bindingsPath)
				if readErr != nil {
					if os.IsNotExist(readErr) {
						return fmt.Errorf("read bindings %s (%s): %w", bindingsPath, bindingsCreateHint(bindingsPath), readErr)
					}
					return fmt.Errorf("read bindings: %w", readErr)
				}
				bindings, bindingDiagnostics := profilemango.ParseBindings(bindingsData)
				diagnostics = append(diagnostics, bindingDiagnostics...)
				if profile.Route != "" {
					if _, found := bindings.Routes[profile.Route]; !found {
						diagnostics.Add(profilemango.SeverityError, "binding.route_missing", "route", "route is not present in bindings", 0, 0)
					}
				}
			}
			if jsonOutput {
				return printValidationJSON(cmd, profile, diagnostics)
			}
			errStyles := stylesFor(cmd.ErrOrStderr(), true)
			for _, diagnostic := range diagnostics.Sorted() {
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), styledDiagnosticLine(diagnostic, errStyles)); err != nil {
					return fmt.Errorf("write diagnostic: %w", err)
				}
			}
			if diagnostics.HasErrors() {
				return fmt.Errorf("validation failed")
			}
			styles := stylesFor(cmd.OutOrStdout(), true)
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", styles.path(profile.Name), styles.success("is valid")); err != nil {
				return fmt.Errorf("write validation result: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bindingsPath, "bindings", "", "optional local route bindings file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit a stable JSON result")
	return cmd
}

// profileFolderName is the folder that names a profile file when it omits name.
func profileFolderName(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	return filepath.Base(filepath.Dir(absolute))
}

func printValidationJSON(cmd *cobra.Command, profile profilemango.PolicyProfile, diagnostics profilemango.Diagnostics) error {
	result := struct {
		Valid       bool                     `json:"valid"`
		ProfileName string                   `json:"profileName,omitempty"`
		Diagnostics profilemango.Diagnostics `json:"diagnostics"`
	}{Valid: !diagnostics.HasErrors(), ProfileName: profile.Name, Diagnostics: diagnostics.Sorted()}
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode validation result: %w", err)
	}
	if !result.Valid {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func diagnosticLine(diagnostic profilemango.Diagnostic) string {
	path := diagnostic.Path
	if path == "" {
		path = "document"
	}
	return fmt.Sprintf("%s %s: %s (%s)", diagnostic.Severity, path, diagnostic.Message, diagnostic.Code)
}
