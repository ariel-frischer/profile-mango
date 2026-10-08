package opencode

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// variantEfforts are the effort names OpenCode 1.18.31 uses as built-in model
// variant keys (packages/opencode/src/provider/transform.ts variants). An agent's
// variant applies only when its configured model defines that variant.
var variantEfforts = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// EffortSupport reports whether effort is installable as a named agent variant,
// and why not otherwise.
func EffortSupport(effort string) (bool, string) {
	if slices.Contains(variantEfforts, effort) {
		return true, ""
	}
	return false, "OpenCode " + TargetVersion + " built-in model variants are only " + strings.Join(variantEfforts, ", ")
}

// AgentDefinition renders a native named Markdown definition. OpenCode trims its
// prompt. A supported route effort becomes the agent's default model variant.
func AgentDefinition(route profilemango.RouteBinding, mode string, instructions []string) ([]byte, error) {
	model, err := validateInstallRoute(route)
	if err != nil {
		return nil, fmt.Errorf("render OpenCode agent model: %w", err)
	}
	if mode != "primary" && mode != "subagent" {
		return nil, fmt.Errorf("OpenCode agent mode must be primary or subagent")
	}
	var body strings.Builder
	body.WriteString("---\nmode: " + mode + "\nmodel: " + strconv.Quote(model) + "\n")
	if supported, _ := EffortSupport(route.Effort); supported {
		body.WriteString("variant: " + strconv.Quote(route.Effort) + "\n")
	}
	body.WriteString("---\n")
	for _, instruction := range instructions {
		if strings.TrimSpace(instruction) == "" {
			return nil, fmt.Errorf("OpenCode agent instruction must not be blank")
		}
		body.WriteString("\n")
		body.WriteString(strings.TrimSpace(instruction))
		body.WriteString("\n")
	}
	return []byte(body.String()), nil
}
