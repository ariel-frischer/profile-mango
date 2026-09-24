package install

import (
	"fmt"
	"path/filepath"

	"gitlab.com/ariel-frischer/profile-mango/internal/installfs"
)

// Install modes: a native named profile the user selects, or the agent's default settings.
const (
	InstallModeNamedProfile  = "named-profile"
	InstallModeDefaultConfig = "default-config"
)

// NoProfilesNote is shown for agents that have no named profiles yet.
const NoProfilesNote = "this agent has no profiles; installed as its default settings"

// InstallMode reports where a target puts the profile and how to use it.
type InstallMode struct {
	Mode        string `json:"mode"`
	ProfileName string `json:"profileName,omitempty"`
	UseCommand  string `json:"useCommand,omitempty"`
	// SetsDefault reports that --default also makes the named profile the agent's default.
	SetsDefault bool `json:"setsDefault,omitempty"`
}

// namedProfileInstaller is implemented by adapters whose agent has native named profiles.
type namedProfileInstaller interface {
	// NamedProfileFile names the profile's file, relative to the main config directory.
	NamedProfileFile(name string) (string, error)
	// NamedProfileUse is the command that starts the agent with the named profile.
	NamedProfileUse(name string) string
}

// installModeFor picks named-profile mode when the adapter supports it; named agent
// destinations keep their own mode and report none.
func installModeFor(adapter Adapter, request Request, target TargetRequest) *InstallMode {
	if !target.Agent.Empty() {
		return nil
	}
	named, found := adapter.(namedProfileInstaller)
	if !found {
		return &InstallMode{Mode: InstallModeDefaultConfig}
	}
	return &InstallMode{Mode: InstallModeNamedProfile, ProfileName: request.ProfileName, UseCommand: named.NamedProfileUse(request.ProfileName), SetsDefault: request.Default}
}

// snapshotNamedFile reads the named profile's current file next to the main config.
func snapshotNamedFile(adapter Adapter, mode *InstallMode, config installfs.Snapshot) (installfs.Snapshot, error) {
	if mode == nil || mode.Mode != InstallModeNamedProfile {
		return installfs.Snapshot{}, nil
	}
	name, err := adapter.(namedProfileInstaller).NamedProfileFile(mode.ProfileName)
	if err != nil {
		return installfs.Snapshot{}, err
	}
	if filepath.Base(name) != name {
		return installfs.Snapshot{}, fmt.Errorf("named profile file must stay in the config directory")
	}
	snapshot, err := installfs.SnapshotFile(filepath.Join(filepath.Dir(config.Path), name))
	if err != nil {
		return snapshot, fmt.Errorf("inspect named profile file: %w", err)
	}
	return snapshot, nil
}

// guardUnchangedConfig makes apply refuse a plan whose main config changed after it was
// read, when the plan itself does not rewrite that config.
func guardUnchangedConfig(targetPlan *TargetPlan, config installfs.Snapshot) {
	for _, check := range targetPlan.checks {
		if check.Path == config.Path {
			return
		}
	}
	targetPlan.checks = append(targetPlan.checks, installfs.Change{Path: config.Path, Before: config})
}
