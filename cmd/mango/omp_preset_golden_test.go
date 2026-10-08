package main

const useRolesPresetGolden = `modelPresets:
  mango-a:
    modelRoles:
      commit: "openai/nano"
      default: "openai/gpt-5.6:high"
      plan: "anthropic/opus:high"
      slow: "anthropic/opus:high"
      smol: "openai/mini:low"
      task: "openai/gpt-5.6:high"
      tiny: "openai/nano"
    defaultThinkingLevel: "high"
  mango-b:
    modelRoles:
      commit: "openai/gpt-5.6:high"
      default: "openai/gpt-5.6:high"
      plan: "openai/gpt-5.6:xhigh"
      slow: "openai/gpt-5.6:xhigh"
      smol: "openai/gpt-5.6:high"
      task: "openai/gpt-5.6:high"
      tiny: "openai/gpt-5.6:high"
    defaultThinkingLevel: "high"
`
