package profilemango

// APIVersion identifies the deprecated wrapped profile form and the resolved model.
const APIVersion = "profilemango.dev/v1alpha1"

// Route defaults applied by ParseBindings when a base route omits the field.
const (
	DefaultTransport      = "native"
	DefaultAuthentication = "oauth"
)

const (
	KindPolicyProfile = "PolicyProfile"
	PlanVersion       = "profilemango.dev/plan/v1alpha1"
	ManifestVersion   = "profilemango.dev/manifest/v1alpha1"
)

// PolicyProfile is the strict, flat YAML input document. Name is optional in
// YAML and defaults to the profile folder name; see ParseProfileAt.
type PolicyProfile struct {
	Name         string            `yaml:"name,omitempty" json:"name,omitempty"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels       map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Extends      string            `yaml:"extends,omitempty" json:"extends,omitempty"`
	Route        string            `yaml:"route,omitempty" json:"route,omitempty"`
	Permissions  *PermissionPolicy `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	Tools        *AccessRules      `yaml:"tools,omitempty" json:"tools,omitempty"`
	Instructions InstructionsSpec  `yaml:"instructions,omitempty" json:"instructions,omitempty"`
	// Skills lists skill folders by their SKILL.md path, each optionally with the
	// provenance of a skill vendored from another repository.
	Skills *[]SkillRef `yaml:"skills,omitempty" json:"skills,omitempty"`
	// GlobalInstructions maps a target name to whole global instruction files
	// (file name -> resource path, or a list of fragment paths composed in order),
	// e.g. codex: {AGENTS.md: instructions/codex.md}.
	GlobalInstructions map[string]map[string]ResourceList `yaml:"globalInstructions,omitempty" json:"globalInstructions,omitempty"`
	// AgentFiles maps a target name to whole native subagent files installed verbatim
	// in its subagent directory (file name -> resource path), e.g. oh-my-pi: {scout.md: agents/scout.md}.
	AgentFiles map[string]map[string]string `yaml:"agentFiles,omitempty" json:"agentFiles,omitempty"`
	// Roles describes roles keyed by canonical identifier; models for them
	// come from the bound route's roles.
	Roles map[string]RoleDefinition `yaml:"roles,omitempty" json:"roles,omitempty"`
}

// Metadata is the resolved identity carried by ResolvedProfile.
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type PermissionPolicy struct {
	Mode    *string `yaml:"mode,omitempty" json:"mode,omitempty"`
	Network *string `yaml:"network,omitempty" json:"network,omitempty"`
	Shell   *string `yaml:"shell,omitempty" json:"shell,omitempty"`
}

type AccessRules struct {
	Allow *[]string `yaml:"allow,omitempty" json:"allow,omitempty"`
	Deny  *[]string `yaml:"deny,omitempty" json:"deny,omitempty"`
}

type InstructionsSpec struct {
	Append *[]string `yaml:"append,omitempty" json:"append,omitempty"`
}

// RoleDefinition describes one portable role. Instructions is an optional
// resource path beneath the package root.
type RoleDefinition struct {
	Description  string  `yaml:"description" json:"description"`
	Instructions *string `yaml:"instructions,omitempty" json:"instructions,omitempty"`
}

// Bindings contains machine-local route identity without credentials.
type Bindings struct {
	Routes map[string]RouteBinding `yaml:"routes" json:"routes"`
}

// RouteBinding is a base route plus optional explicit per-target overrides.
// Targets is empty in every effective route returned by RouteFor. ParseBindings
// fills an omitted Transport with DefaultTransport and Authentication with
// DefaultAuthentication. Roles maps role identifiers to their own route;
// the base route is the default role.
// SubagentMaxEffort caps the effort a caller may request for one subagent spawn.
// A target override may change bound Roles through targets.<agent>.roles but
// never SubagentMaxEffort.
type RouteBinding struct {
	Provider          string                   `yaml:"provider" json:"provider"`
	Transport         string                   `yaml:"transport" json:"transport"`
	Authentication    string                   `yaml:"authentication" json:"authentication"`
	Model             string                   `yaml:"model" json:"model"`
	Effort            string                   `yaml:"effort" json:"effort"`
	SubagentMaxEffort string                   `yaml:"subagentMaxEffort,omitempty" json:"subagentMaxEffort,omitempty"`
	Targets           map[string]RouteOverride `yaml:"targets,omitempty" json:"targets,omitempty"`
	Roles             map[string]RoleRoute     `yaml:"roles,omitempty" json:"roles,omitempty"`
}

// RouteOverride replaces any non-empty base route field for one target. Roles
// replaces non-empty fields of roles the base route already binds.
type RouteOverride struct {
	Provider       string                  `yaml:"provider,omitempty" json:"provider,omitempty"`
	Transport      string                  `yaml:"transport,omitempty" json:"transport,omitempty"`
	Authentication string                  `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	Model          string                  `yaml:"model,omitempty" json:"model,omitempty"`
	Effort         string                  `yaml:"effort,omitempty" json:"effort,omitempty"`
	Roles          map[string]RoleOverride `yaml:"roles,omitempty" json:"roles,omitempty"`
}

// RoleOverride replaces any non-empty field of one base role route for one target.
type RoleOverride struct {
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Model    string `yaml:"model,omitempty" json:"model,omitempty"`
	Effort   string `yaml:"effort,omitempty" json:"effort,omitempty"`
}

// RoleRoute is the provider, model, and optional effort for one portable role.
type RoleRoute struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
	Effort   string `yaml:"effort,omitempty" json:"effort,omitempty"`
}

// ReservedRoleDefault names the role the base route already defines.
const ReservedRoleDefault = "default"

// ResolvedProfile is the deterministic canonical model after inheritance.
type ResolvedProfile struct {
	APIVersion   string            `json:"apiVersion"`
	Kind         string            `json:"kind"`
	Metadata     Metadata          `json:"metadata"`
	RouteRef     string            `json:"routeRef"`
	Permissions  *PermissionPolicy `json:"permissions,omitempty"`
	Tools        *ResolvedRules    `json:"tools,omitempty"`
	Instructions []string          `json:"instructions,omitempty"`
	Skills       []SkillRef        `json:"skills,omitempty"`
	// GlobalInstructions maps a target name to whole global instruction files it owns.
	GlobalInstructions map[string]map[string]ResourceList `json:"globalInstructions,omitempty"`
	// AgentFiles maps a target name to whole native subagent files it owns.
	AgentFiles map[string]map[string]string `json:"agentFiles,omitempty"`
	// Roles is the resolved portable role definitions.
	Roles  map[string]RoleDefinition `json:"roles,omitempty"`
	Parent string                    `json:"parent,omitempty"`
}

type ResolvedRules struct {
	Allow   []string `json:"allow,omitempty"`
	Deny    []string `json:"deny,omitempty"`
	Managed bool     `json:"managed"`
	Closed  bool     `json:"closed"`
}

type ResourceDigest struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Plan is a versioned, inert description of future compilation work.
type Plan struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Profile    string           `json:"profile"`
	Target     string           `json:"target"`
	Route      RouteBinding     `json:"route"`
	Resources  []ResourceDigest `json:"resources"`
}

// Manifest records ownership metadata without applying it.
type Manifest struct {
	APIVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Owner      string           `json:"owner"`
	Generation uint64           `json:"generation"`
	Profile    string           `json:"profile"`
	Target     string           `json:"target"`
	Resources  []ResourceDigest `json:"resources"`
}
