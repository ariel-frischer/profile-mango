package install

import (
	"fmt"
	"slices"
	"sort"
	"strings"
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

// ResolveTarget parses "name" or "name@version". A bare name resolves to the
// target's single installable registered version; an explicit version is kept
// as given so an unqualified version still blocks during planning.
func (registry *Registry) ResolveTarget(value string) (Target, error) {
	selector, err := ParseTargetSelector(value)
	if err != nil || selector.Version != "" {
		return selector, err
	}
	var qualified, names []string
	var result Target
	for _, target := range registry.Targets() {
		if !slices.Contains(names, target.Name) {
			names = append(names, target.Name)
		}
		adapter, _ := registry.Lookup(target)
		if target.Name == selector.Name && adapter.Metadata().Installable {
			qualified = append(qualified, target.String())
			result = target
		}
	}
	switch {
	case !slices.Contains(names, selector.Name):
		return Target{}, fmt.Errorf("unknown target %q; valid targets: %s", selector.Name, strings.Join(names, ", "))
	case len(qualified) == 1:
		return result, nil
	case len(qualified) == 0:
		return Target{}, fmt.Errorf("target %s has no qualified version; use %s@<version>", selector.Name, selector.Name)
	default:
		return Target{}, fmt.Errorf("target %s has several qualified versions (%s); use target@version", selector.Name, strings.Join(qualified, ", "))
	}
}

func DefaultRegistry() *Registry {
	return NewRegistry(
		claudeCodeAdapter{},
		codexAdapter{},
		hermesAdapter{},
		ohMyPiAdapter{},
		openClawAdapter{},
		piAdapter{},
		openCodeAdapter{},
	)
}
