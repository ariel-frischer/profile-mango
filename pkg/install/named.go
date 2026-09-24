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

// namedProfileUser is implemented by adapters whose agent has native named profiles.
type namedProfileUser interface {
	// NamedProfileUse is the command that starts the agent with the named profile.
	NamedProfileUse(name string) string
}

// namedProfileInstaller locates a named profile's file from its name alone.
type namedProfileInstaller interface {
	namedProfileUser
	// NamedProfileFile names the profile's file: absolute, or relative to the main config
	// directory without escaping it.
	NamedProfileFile(name string) (string, error)
}

// namedProfilePathResolver locates a named profile's file from the resolved main config
// path, for agents whose profiles live beside rather than below that config.
type namedProfilePathResolver interface {
	namedProfileUser
	// NamedProfilePath returns the profile file's absolute path, or configPath itself when
	// the agent selects its default config for that name.
	NamedProfilePath(configPath, name string) (string, error)
}

// installModeFor picks named-profile mode when the adapter supports it; named agent
// destinations keep their own mode and report none.
func installModeFor(adapter Adapter, request Request, target TargetRequest) *InstallMode {
	if !target.Agent.Empty() {
		return nil
	}
	named, found := adapter.(namedProfileUser)
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
	name, err := namedProfileFile(adapter, mode.ProfileName, config.Path)
	if err != nil {
		return installfs.Snapshot{}, err
	}
	if _, resolved := adapter.(namedProfilePathResolver); resolved && name == config.Path {
		return installfs.Snapshot{}, nil
	}
	path, err := patchPath(config.Path, name)
	if err != nil || path == config.Path {
		return installfs.Snapshot{}, fmt.Errorf("named profile file must be a separate file inside the config directory or an absolute path")
	}
	snapshot, err := installfs.SnapshotFile(path)
	if err != nil {
		return snapshot, fmt.Errorf("inspect named profile file: %w", err)
	}
	return snapshot, nil
}

func namedProfileFile(adapter Adapter, profile, configPath string) (string, error) {
	if resolver, found := adapter.(namedProfilePathResolver); found {
		path, err := resolver.NamedProfilePath(configPath, profile)
		if err == nil && !filepath.IsAbs(path) {
			return "", fmt.Errorf("named profile path must be absolute")
		}
		return path, err
	}
	return adapter.(namedProfileInstaller).NamedProfileFile(profile)
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
