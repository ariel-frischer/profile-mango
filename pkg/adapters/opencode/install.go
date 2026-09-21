package opencode

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// ModelPatch is the only install-capable OpenCode configuration effect.
type ModelPatch struct {
	Content []byte
	Before  string
	After   string
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
