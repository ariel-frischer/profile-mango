package claudecode

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

const effortKey = "effortLevel"

// installEfforts are the top-level settings effortLevel values the exact 2.1.278
// settings schema accepts and whose consumption was observed natively. Other
// values (max is session-only) are dropped by Claude Code's schema, so they are
// never written.
var installEfforts = []string{"low", "medium", "high", "xhigh"}

// EffortSupport reports whether effort is installable as settings effortLevel,
// and why not otherwise.
func EffortSupport(effort string) (bool, string) {
	if slices.Contains(installEfforts, effort) {
		return true, ""
	}
	return false, "Claude Code " + TargetVersion + " settings effortLevel accepts only " + strings.Join(installEfforts, ", ")
}

// agentEfforts are the named effort values a ~/.claude/agents/*.md frontmatter
// effort accepts: the binary validates it against ["low","medium","high","xhigh","max"]
// (installed 2.1.281 binary, "Agent file ... has invalid effort"). Integers are
// also accepted there but are not portable efforts.
var agentEfforts = []string{"low", "medium", "high", "xhigh", "max"}

// AgentEffortSupport reports whether effort is writable as agent file frontmatter effort.
func AgentEffortSupport(effort string) (bool, string) {
	if slices.Contains(agentEfforts, effort) {
		return true, ""
	}
	return false, "Claude Code agent file effort accepts only " + strings.Join(agentEfforts, ", ")
}

// SettingsPatch is the model plus, when installable, the effortLevel change.
type SettingsPatch struct {
	Content      []byte
	ModelBefore  string
	ModelAfter   string
	EffortBefore string
	EffortAfter  string
}

// PatchSettings writes the route model and, when EffortSupport allows it, the
// route effort as top-level effortLevel. Unsupported efforts are left untouched.
// The effort is patched first so an inserted model precedes it.
func PatchSettings(source []byte, route profilemango.RouteBinding) (SettingsPatch, error) {
	model, err := validateInstallRoute(route)
	if err != nil {
		return SettingsPatch{}, err
	}
	var patch SettingsPatch
	if supported, _ := EffortSupport(route.Effort); supported {
		if len(source) == 0 {
			source = []byte("{}\n")
		}
		effort, err := patchEffortJSON(source, route.Effort)
		if err != nil {
			return SettingsPatch{}, err
		}
		source, patch.EffortBefore, patch.EffortAfter = effort.Content, effort.Before, effort.After
	}
	modelPatch, err := PatchModelJSON(source, model)
	if err != nil {
		return SettingsPatch{}, err
	}
	patch.Content, patch.ModelBefore, patch.ModelAfter = modelPatch.Content, modelPatch.Before, modelPatch.After
	return patch, nil
}

func patchEffortJSON(source []byte, effort string) (ModelPatch, error) {
	scan, err := inspectSettings(source)
	if err != nil {
		return ModelPatch{}, fmt.Errorf("patch Claude Code effortLevel: %w", err)
	}
	literal := strconv.Quote(effort)
	if scan.effort == nil {
		return ModelPatch{Content: insertTopLevelKey(source, scan, effortKey, literal), After: effort, Inserted: true}, nil
	}
	if scan.effort.value == effort {
		return ModelPatch{Content: append([]byte(nil), source...), Before: effort, After: effort, Found: true}, nil
	}
	content := replaceJSONSpan(source, scan.effort.start, scan.effort.end, literal)
	return ModelPatch{Content: content, Before: scan.effort.value, After: effort, Found: true}, nil
}
