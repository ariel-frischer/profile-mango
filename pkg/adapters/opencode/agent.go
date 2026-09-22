package opencode

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// AgentDefinition renders a native named Markdown definition. OpenCode trims its prompt.
func AgentDefinition(route profilemango.RouteBinding, mode string, instructions []string) ([]byte, error) {
	model, err := validateInstallRoute(route)
	if err != nil {
		return nil, fmt.Errorf("render OpenCode agent model: %w", err)
	}
	if mode != "primary" && mode != "subagent" {
		return nil, fmt.Errorf("OpenCode agent mode must be primary or subagent")
	}
	var body strings.Builder
	body.WriteString("---\nmode: " + mode + "\nmodel: " + strconv.Quote(model) + "\n---\n")
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
