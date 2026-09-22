// OpenClaw v2026.9.5 source-native field-consumption probe.
// Run only from a disposable sandbox with the exact source mounted at /opt/openclaw.

const sourceRoot = "/opt/openclaw";

const [{ parseConfigJson5 }, { validateConfigObject }, scope, thinking] = await Promise.all([
  import(`${sourceRoot}/src/config/io.read-helpers.ts`),
  import(`${sourceRoot}/src/config/validation-core.ts`),
  import(`${sourceRoot}/src/agents/agent-scope.ts`),
  import(`${sourceRoot}/src/agents/model-thinking-default-core.ts`),
]);

function assertEqual(actual, expected, label) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    throw new Error(`${label}: got ${JSON.stringify(actual)}, want ${JSON.stringify(expected)}`);
  }
}

function assert(condition, label) {
  if (!condition) {
    throw new Error(label);
  }
}

const raw = `{
  agents: {
    defaults: {
      model: { primary: "synthetic/global-primary", fallbacks: ["synthetic/global-fallback"] },
      thinkingDefault: "high",
    },
    entries: { main: {} },
  },
}\n`;
const parsed = parseConfigJson5(raw);
assert(parsed.ok, `source JSON5 parser rejected the candidate: ${parsed.error ?? "unknown error"}`);
const accepted = validateConfigObject(parsed.parsed);
assert(accepted.ok, `source config validator rejected the candidate: ${JSON.stringify(accepted.issues)}`);

const unknown = validateConfigObject({
  agents: {
    defaults: {
      model: { primary: "synthetic/global-primary" },
      thinkingDefault: "high",
      unknownProperty: true,
    },
  },
});
assert(!unknown.ok, "source config validator accepted an unknown agents.defaults property");

const invalidThinking = validateConfigObject({
  agents: { defaults: { model: { primary: "synthetic/global-primary" }, thinkingDefault: "turbo" } },
});
assert(!invalidThinking.ok, "source config validator accepted an unsupported thinkingDefault");

const globalConfig = accepted.config;
const globalPrimary = scope.resolveAgentEffectiveModelPrimary(globalConfig, "main");
const globalExplicitAgentPrimary = scope.resolveAgentExplicitModelPrimary(globalConfig, "main");
const globalThinking = thinking.resolveConfiguredThinkingDefaultCore({
  cfg: globalConfig,
  provider: "synthetic",
  model: "global-primary",
});
const inheritedFallbackAvailability = scope.resolveModelFallbackAvailability({
  cfg: globalConfig,
  agentId: "main",
  hasSessionModelOverride: false,
});
const inheritedFallbackProjection = scope.resolveEffectiveModelFallbacks({
  cfg: globalConfig,
  agentId: "main",
  hasSessionModelOverride: false,
});

assertEqual(globalPrimary, "synthetic/global-primary", "global primary getter");
assertEqual(globalExplicitAgentPrimary, undefined, "global explicit-agent primary getter");
assertEqual(globalThinking, "high", "global thinking getter");
assertEqual(inheritedFallbackAvailability, {
  kind: "active",
  models: ["synthetic/global-fallback"],
  source: "inherited",
}, "global fallback availability");
assertEqual(inheritedFallbackProjection, undefined, "inherited fallback projection");

const agentOverrideConfig = {
  agents: {
    defaults: {
      model: { primary: "synthetic/global-primary", fallbacks: ["synthetic/global-fallback"] },
      thinkingDefault: "high",
    },
    entries: {
      main: {
        model: { primary: "synthetic/agent-primary", fallbacks: ["synthetic/agent-fallback"] },
        thinkingDefault: "low",
      },
    },
  },
};
const agentPrimary = scope.resolveAgentEffectiveModelPrimary(agentOverrideConfig, "main");
const agentThinking = thinking.resolveConfiguredThinkingDefaultCore({
  cfg: agentOverrideConfig,
  agentId: "main",
  provider: "synthetic",
  model: "agent-primary",
});
const agentFallbackAvailability = scope.resolveModelFallbackAvailability({
  cfg: agentOverrideConfig,
  agentId: "main",
  hasSessionModelOverride: false,
});
const agentFallbackProjection = scope.resolveEffectiveModelFallbacks({
  cfg: agentOverrideConfig,
  agentId: "main",
  hasSessionModelOverride: false,
});
const userOverrideFallbackAvailability = scope.resolveModelFallbackAvailability({
  cfg: globalConfig,
  agentId: "main",
  hasSessionModelOverride: true,
  modelOverrideSource: "user",
});

assertEqual(agentPrimary, "synthetic/agent-primary", "per-agent primary override");
assertEqual(agentThinking, "low", "per-agent thinking override");
assertEqual(agentFallbackAvailability, {
  kind: "active",
  models: ["synthetic/agent-fallback"],
  source: "explicit",
}, "per-agent fallback override");
assertEqual(agentFallbackProjection, ["synthetic/agent-fallback"], "per-agent fallback projection");
assertEqual(userOverrideFallbackAvailability, {
  kind: "disabled_by_model_override",
}, "user model override fallback limit");

const agentPrimaryWithoutFallbacks = {
  agents: {
    defaults: {
      model: { primary: "synthetic/global-primary", fallbacks: ["synthetic/global-fallback"] },
      thinkingDefault: "high",
    },
    entries: { main: { model: { primary: "synthetic/agent-primary" } } },
  },
};
const disabledAgentFallbacks = scope.resolveModelFallbackAvailability({
  cfg: agentPrimaryWithoutFallbacks,
  agentId: "main",
  hasSessionModelOverride: false,
});
assertEqual(disabledAgentFallbacks, {
  kind: "none_configured",
  source: "explicit",
}, "agent primary without fallback inheritance");

process.stdout.write(`${JSON.stringify({
  source: {
    commit: "ec9c1a13db8938e5a3eaa51fca2e981cde2395a9",
    tree: "ac00d08eb2766b4fd114bee710e78a1df6c3cc03",
    modules: [
      "src/config/io.read-helpers.ts#parseConfigJson5",
      "src/config/validation-core.ts#validateConfigObject",
      "src/agents/agent-scope.ts#resolveAgentEffectiveModelPrimary",
      "src/agents/agent-scope.ts#resolveModelFallbackAvailability",
      "src/agents/model-thinking-default-core.ts#resolveConfiguredThinkingDefaultCore",
    ],
  },
  parser: {
    accepted: true,
    unknownDefaultsPropertyRejected: true,
    unsupportedThinkingDefaultRejected: true,
  },
  fieldConsumption: {
    globalPrimary,
    globalThinking,
    inheritedFallbackAvailability,
    inheritedFallbackProjection,
    agentPrimary,
    agentThinking,
    agentFallbackAvailability,
    agentFallbackProjection,
  },
  limits: {
    userOverrideFallbackAvailability,
    agentPrimaryWithoutFallbacks: disabledAgentFallbacks,
  },
}, null, 2)}\n`);
