package install

const ohMyPiPrimaryPresetGolden = "modelPresets:\n  mango-primary:\n    modelRoles:\n" +
	"      commit: \"openai/gpt-5.6:high\"\n      default: \"openai/gpt-5.6:high\"\n      plan: \"openai/gpt-5.6:high\"\n      slow: \"openai/gpt-5.6:high\"\n      smol: \"openai/gpt-5.6:high\"\n      task: \"openai/gpt-5.6:high\"\n      tiny: \"openai/gpt-5.6:high\"\n    defaultThinkingLevel: \"high\"\n"

const ohMyPiRolesPresetGolden = "modelPresets:\n  mango-primary:\n    modelRoles:\n" +
	"      commit: \"opencode-go/glm-5.3-flash:low\"\n      default: \"anthropic/claude-opus-5-5:medium\"\n      plan: \"anthropic/claude-opus-5-5:high\"\n      slow: \"anthropic/claude-opus-5-5:high\"\n      smol: \"opencode-go/gpt-6-luna:high\"\n      task: \"anthropic/claude-opus-5-5:medium\"\n      tiny: \"opencode-go/glm-5.3-flash:low\"\n    defaultThinkingLevel: \"medium\"\n"

const ohMyPiRolesPresetJSON = `{"defaultThinkingLevel":"medium","modelRoles":{"commit":"opencode-go/glm-5.3-flash:low","default":"anthropic/claude-opus-5-5:medium","plan":"anthropic/claude-opus-5-5:high","slow":"anthropic/claude-opus-5-5:high","smol":"opencode-go/gpt-6-luna:high","task":"anthropic/claude-opus-5-5:medium","tiny":"opencode-go/glm-5.3-flash:low"}}`
