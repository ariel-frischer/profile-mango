package install

import (
	"fmt"
	"sort"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
)

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	registry := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		_ = registry.Register(adapter)
	}
	return registry
}

func (registry *Registry) Register(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("adapter is required")
	}
	metadata := adapter.Metadata()
	if metadata.Target == "" || metadata.Version == "" {
		return fmt.Errorf("adapter metadata requires target and version")
	}
	if registry.adapters == nil {
		registry.adapters = make(map[string]Adapter)
	}
	key := Target{Name: metadata.Target, Version: metadata.Version}.String()
	if _, found := registry.adapters[key]; found {
		return fmt.Errorf("adapter already registered: %s", key)
	}
	registry.adapters[key] = adapter
	return nil
}

func (registry *Registry) Lookup(target Target) (Adapter, bool) {
	if registry == nil {
		return nil, false
	}
	adapter, found := registry.adapters[target.String()]
	return adapter, found
}

func (registry *Registry) Targets() []Target {
	if registry == nil {
		return nil
	}
	result := make([]Target, 0, len(registry.adapters))
	for _, adapter := range registry.adapters {
		metadata := adapter.Metadata()
		result = append(result, Target{Name: metadata.Target, Version: metadata.Version})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

type blockedAdapter struct{ metadata AdapterMetadata }

func (adapter blockedAdapter) Metadata() AdapterMetadata { return adapter.metadata }

func (adapter blockedAdapter) Plan(AdapterInput) (Patch, error) {
	return Patch{}, fmt.Errorf("target %s is install-blocked", adapter.metadata.Target)
}

func DefaultRegistry() *Registry {
	blockedReason := "production target installation is blocked pending disposable-target validation; no target state is read or written"
	return NewRegistry(
		claudeCodeAdapter{},
		blockedAdapter{metadata: blockedMetadata(codex.TargetName, codex.TargetVersion, codex.AdapterVersion, codex.EvidenceSHA256, blockedReason)},
		hermesAdapter{},
		blockedAdapter{metadata: blockedMetadata(ohmypi.TargetName, ohmypi.TargetVersion, ohmypi.AdapterVersion, ohmypi.EvidenceSHA256, blockedReason)},
		blockedAdapter{metadata: blockedMetadata(openclaw.TargetName, openclaw.TargetVersion, openclaw.AdapterVersion, openclaw.EvidenceSHA256, blockedReason)},
		piAdapter{},
		openCodeAdapter{},
	)
}

func blockedMetadata(target, version, adapterVersion, evidence, reason string) AdapterMetadata {
	return AdapterMetadata{Target: target, Version: version, AdapterVersion: adapterVersion, EvidenceSHA256: evidence, Installable: false, Status: StatusBlocked, Reason: reason}
}
