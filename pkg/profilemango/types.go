package profilemango

// APIVersion is the only profile contract supported by M0.
const APIVersion = "profilemango.dev/v1alpha1"

const (
	KindPolicyProfile = "PolicyProfile"
	PlanVersion       = "profilemango.dev/plan/v1alpha1"
	ManifestVersion   = "profilemango.dev/manifest/v1alpha1"
)

// PolicyProfile is the strict YAML input document.
type PolicyProfile struct {
	APIVersion string      `yaml:"apiVersion" json:"apiVersion"`
	Kind       string      `yaml:"kind" json:"kind"`
	Metadata   Metadata    `yaml:"metadata" json:"metadata"`
	Spec       ProfileSpec `yaml:"spec" json:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type ProfileSpec struct {
	Extends      string            `yaml:"extends,omitempty" json:"extends,omitempty"`
	RouteRef     string            `yaml:"routeRef" json:"routeRef"`
	Permissions  *PermissionPolicy `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	Tools        *AccessRules      `yaml:"tools,omitempty" json:"tools,omitempty"`
	Instructions InstructionsSpec  `yaml:"instructions,omitempty" json:"instructions,omitempty"`
	Skills       *[]string         `yaml:"skills,omitempty" json:"skills,omitempty"`
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

// Bindings contains machine-local route identity without credentials.
type Bindings struct {
	Routes map[string]RouteBinding `yaml:"routes" json:"routes"`
}

// RouteBinding is a base route plus optional explicit per-target overrides.
// Targets is empty in every effective route returned by RouteFor.
type RouteBinding struct {
	Provider       string                   `yaml:"provider" json:"provider"`
	Transport      string                   `yaml:"transport" json:"transport"`
	Authentication string                   `yaml:"authentication" json:"authentication"`
	Model          string                   `yaml:"model" json:"model"`
	Effort         string                   `yaml:"effort" json:"effort"`
	Targets        map[string]RouteOverride `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// RouteOverride replaces any non-empty base route field for one target.
type RouteOverride struct {
	Provider       string `yaml:"provider,omitempty" json:"provider,omitempty"`
	Transport      string `yaml:"transport,omitempty" json:"transport,omitempty"`
	Authentication string `yaml:"authentication,omitempty" json:"authentication,omitempty"`
	Model          string `yaml:"model,omitempty" json:"model,omitempty"`
	Effort         string `yaml:"effort,omitempty" json:"effort,omitempty"`
}

// ResolvedProfile is the deterministic canonical model after inheritance.
type ResolvedProfile struct {
	APIVersion   string            `json:"apiVersion"`
	Kind         string            `json:"kind"`
	Metadata     Metadata          `json:"metadata"`
	RouteRef     string            `json:"routeRef"`
	Permissions  *PermissionPolicy `json:"permissions,omitempty"`
	Tools        *ResolvedRules    `json:"tools,omitempty"`
	Instructions []string          `json:"instructions,omitempty"`
	Skills       []string          `json:"skills,omitempty"`
	Parent       string            `json:"parent,omitempty"`
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
