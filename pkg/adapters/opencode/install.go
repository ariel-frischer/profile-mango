package opencode

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ModelPatch is the model-only OpenCode configuration effect.
type ModelPatch struct {
	Content []byte
	Before  string
	After   string
}

// ConfigPatch is the lossless OpenCode configuration patch for one install.
type ConfigPatch struct {
	Content       []byte
	ModelBefore   string
	ModelAfter    string
	SkillsBefore  []string
	SkillsAfter   []string
	SkillsAdded   []string
	SkillsRemoved bool
}

// PatchConfig applies the qualified model field and optional skill root field.
func PatchConfig(source []byte, route profilemango.RouteBinding, skillPaths []string) (ConfigPatch, error) {
	model, err := PatchModel(source, route)
	if err != nil {
		return ConfigPatch{}, err
	}
	result := ConfigPatch{Content: model.Content, ModelBefore: model.Before, ModelAfter: model.After}
	if len(skillPaths) == 0 {
		return result, nil
	}
	skills, err := PatchSkillsJSONC(model.Content, skillPaths)
	if err != nil {
		return ConfigPatch{}, err
	}
	result.Content = skills.Content
	result.SkillsBefore = skills.Before
	result.SkillsAfter = skills.After
	result.SkillsAdded = skills.Added
	return result, nil
}

// PatchConfigRemoveSkillPath removes one target-owned skills.paths entry.
func PatchConfigRemoveSkillPath(source []byte, route profilemango.RouteBinding, skillPath string) (ConfigPatch, error) {
	model, err := PatchModel(source, route)
	if err != nil {
		return ConfigPatch{}, err
	}
	skills, err := RemoveSkillPathJSONC(model.Content, skillPath)
	if err != nil {
		return ConfigPatch{}, err
	}
	return ConfigPatch{
		Content:       skills.Content,
		ModelBefore:   model.Before,
		ModelAfter:    model.After,
		SkillsBefore:  skills.Before,
		SkillsAfter:   skills.After,
		SkillsRemoved: skills.Removed,
	}, nil
}

// PatchModel changes only the top-level JSONC model field.
func PatchModel(source []byte, route profilemango.RouteBinding) (ModelPatch, error) {
	model, err := validateInstallRoute(route)
	if err != nil {
		return ModelPatch{}, err
	}
	if len(source) == 0 {
		content := []byte("{\n  \"model\": " + strconv.Quote(model) + "\n}\n")
		return ModelPatch{Content: content, After: model}, nil
	}
	parsed, err := PatchModelJSONC(source, model)
	if err != nil {
		return ModelPatch{}, err
	}
	return ModelPatch{Content: parsed.Content, Before: parsed.Before, After: model}, nil
}

func routeModel(route profilemango.RouteBinding) (string, error) {
	if route.Transport != "native" {
		return "", fmt.Errorf("OpenCode model install requires native transport")
	}
	if route.Provider == "" || route.Model == "" {
		return "", fmt.Errorf("OpenCode model install requires provider and model")
	}
	if strings.Contains(route.Provider, "/") || unsafeRoutePart(route.Provider) || unsafeRoutePart(route.Model) {
		return "", fmt.Errorf("OpenCode provider/model contains unsupported characters")
	}
	return route.Provider + "/" + route.Model, nil
}

// RoleModel is the provider/model selector a role subagent file's model names
// (packages/core/src/v1/config/agent.ts model).
func RoleModel(provider, model string) (string, error) {
	return routeModel(profilemango.RouteBinding{Transport: "native", Provider: provider, Model: model})
}

func validateInstallRoute(route profilemango.RouteBinding) (string, error) {
	model, err := routeModel(route)
	if err != nil {
		return "", err
	}
	if route.Authentication == "" {
		return "", fmt.Errorf("OpenCode model install requires an authentication mode")
	}
	if !validInstallEffort(route.Effort) {
		return "", fmt.Errorf("OpenCode model install rejects unsupported effort %q", route.Effort)
	}
	return model, nil
}

func validInstallEffort(value string) bool {
	switch value {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra":
		return true
	default:
		return false
	}
}
func unsafeRoutePart(value string) bool {
	return strings.IndexFunc(value, func(character rune) bool {
		return character == 0 || unicode.IsSpace(character) || unicode.IsControl(character)
	}) >= 0
}
