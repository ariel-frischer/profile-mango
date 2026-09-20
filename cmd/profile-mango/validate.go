package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func newValidateCmd() *cobra.Command {
	var bindingsPath string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "validate <profile.yaml>",
		Short: "Validate one PolicyProfile without accessing agent homes or the network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileData, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read profile: %w", err)
			}
			profile, diagnostics := profilemango.ParseProfile(profileData)
			if bindingsPath != "" {
				bindingsData, readErr := os.ReadFile(bindingsPath)
				if readErr != nil {
					return fmt.Errorf("read bindings: %w", readErr)
				}
				bindings, bindingDiagnostics := profilemango.ParseBindings(bindingsData)
				diagnostics = append(diagnostics, bindingDiagnostics...)
				if profile.Spec.RouteRef != "" {
					if _, found := bindings.Routes[profile.Spec.RouteRef]; !found {
						diagnostics.Add(profilemango.SeverityError, "binding.route_missing", "spec.routeRef", "routeRef is not present in bindings", 0, 0)
					}
				}
			}
			if jsonOutput {
				return printValidationJSON(cmd, profile, diagnostics)
			}
			for _, diagnostic := range diagnostics.Sorted() {
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), diagnosticLine(diagnostic)); err != nil {
					return fmt.Errorf("write diagnostic: %w", err)
				}
			}
			if diagnostics.HasErrors() {
				return fmt.Errorf("validation failed")
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s is valid\n", profile.Metadata.Name); err != nil {
				return fmt.Errorf("write validation result: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bindingsPath, "bindings", "", "optional local route bindings file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "emit a stable JSON result")
	return cmd
}

func printValidationJSON(cmd *cobra.Command, profile profilemango.PolicyProfile, diagnostics profilemango.Diagnostics) error {
	result := struct {
		Valid       bool                     `json:"valid"`
		ProfileName string                   `json:"profileName,omitempty"`
		Diagnostics profilemango.Diagnostics `json:"diagnostics"`
	}{Valid: !diagnostics.HasErrors(), ProfileName: profile.Metadata.Name, Diagnostics: diagnostics.Sorted()}
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
