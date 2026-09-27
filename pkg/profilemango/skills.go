package profilemango

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// SkillFile is the file every skills entry names; its folder is the installed skill.
const SkillFile = "SKILL.md"

// MaxSkillFiles caps how many regular files one skill folder may hold.
const MaxSkillFiles = 256

// Skill file modes: a file with any executable bit installs as SkillModeExec.
const (
	SkillModeExec  fs.FileMode = 0o755
	SkillModePlain fs.FileMode = 0o644
)

var (
	// skillNamePattern is the skill folder name every qualified agent accepts.
	skillNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	commitPattern    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	sha256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// SkillRef is one skills entry: a <dir>/SKILL.md path beneath the package root and,
// for a skill vendored from another repository, its pinned provenance.
type SkillRef struct {
	Path   string
	Source *SkillSource
}

// SkillSource records where a vendored skill folder came from. SHA256 is the folder's
// tree digest (see SkillTreeDigest), checked offline on every validation and install.
type SkillSource struct {
	Repo   string `yaml:"repo" json:"repo"`
	Commit string `yaml:"commit" json:"commit"`
	Path   string `yaml:"path,omitempty" json:"path,omitempty"`
	SHA256 string `yaml:"sha256" json:"sha256"`
}

// skillRefFields and skillSourceFields are the only keys the mapping forms accept.
var (
	skillRefFields    = []string{"path", "source"}
	skillSourceFields = []string{"repo", "commit", "path", "sha256"}
)

// UnmarshalYAML accepts a scalar path or a {path, source} mapping, rejecting unknown keys.
func (ref *SkillRef) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		var single string
		if err := node.Decode(&single); err != nil {
			return fmt.Errorf("decode skill path: %w", err)
		}
		*ref = SkillRef{Path: single}
		return nil
	}
	if err := requireKnownKeys(node, "skills item", skillRefFields); err != nil {
		return err
	}
	var decoded struct {
		Path string `yaml:"path"`
	}
	if err := node.Decode(&decoded); err != nil {
		return fmt.Errorf("decode skills item: %w", err)
	}
	*ref = SkillRef{Path: decoded.Path}
	source := mappingValue(node, "source")
	if source == nil {
		return nil
	}
	if err := requireKnownKeys(source, "skills item source", skillSourceFields); err != nil {
		return err
	}
	ref.Source = &SkillSource{}
	if err := source.Decode(ref.Source); err != nil {
		return fmt.Errorf("decode skills item source: %w", err)
	}
	return nil
}

// requireKnownKeys fails closed on a mapping key outside allowed.
func requireKnownKeys(node *yaml.Node, label string, allowed []string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%s must be a map", label)
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		key := node.Content[index].Value
		if !containsString(allowed, key) {
			return fmt.Errorf("%s: unknown field %q; expected one of %s", label, key, strings.Join(allowed, ", "))
		}
	}
	return nil
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// MarshalJSON writes a skill without provenance as its path string, so the string form
// keeps its resolved-profile JSON, and a vendored skill as {path, source}.
func (ref SkillRef) MarshalJSON() ([]byte, error) {
	if ref.Source == nil {
		return json.Marshal(ref.Path)
	}
	return json.Marshal(struct {
		Path   string       `json:"path"`
		Source *SkillSource `json:"source"`
	}{ref.Path, ref.Source})
}

// Dir is the skill folder path beneath the package root, e.g. skills/review.
func (ref SkillRef) Dir() string { return path.Dir(path.Clean(ref.Path)) }

// Name is the skill folder name the agents see, e.g. review.
func (ref SkillRef) Name() string { return path.Base(ref.Dir()) }

// SkillPaths returns each entry's SKILL.md path in order.
func SkillPaths(refs []SkillRef) []string {
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.Path)
	}
	return paths
}

// skillRefsFromPaths wraps plain path strings, e.g. from the deprecated wrapped form.
func skillRefsFromPaths(paths *[]string) *[]SkillRef {
	if paths == nil {
		return nil
	}
	refs := make([]SkillRef, 0, len(*paths))
	for _, value := range *paths {
		refs = append(refs, SkillRef{Path: value})
	}
	return &refs
}

// validateSkills checks every skills entry offline: a <dir>/SKILL.md path beneath the
// package root, a valid and unique folder name, and well-formed provenance.
func validateSkills(skills *[]SkillRef, diagnostics *Diagnostics) {
	if skills == nil {
		return
	}
	seen := make(map[string]int, len(*skills))
	for index, ref := range *skills {
		at := fmt.Sprintf("skills[%d]", index)
		if !validateSkillPath(ref.Path, at, diagnostics) {
			continue
		}
		if first, duplicate := seen[ref.Name()]; duplicate {
			diagnostics.Add(SeverityError, "profile.skill_duplicate", at, fmt.Sprintf("skill %q is already listed at skills[%d]; agents key skills by folder name", ref.Name(), first), 0, 0)
		} else {
			seen[ref.Name()] = index
		}
		if ref.Source != nil {
			validateSkillSource(*ref.Source, at+".source", diagnostics)
		}
	}
}

// validateSkillPath reports a path that is not <dir>/SKILL.md beneath the package root
// or whose folder name agents cannot load; it returns false when it reported one.
func validateSkillPath(value, at string, diagnostics *Diagnostics) bool {
	clean := path.Clean(value)
	switch {
	case strings.TrimSpace(value) == "" || !safeGlobalResource(value) || strings.Contains(value, "\\"):
		diagnostics.Add(SeverityError, "profile.skill_path_invalid", at, "skill path must be relative and remain beneath the package root", 0, 0)
	case path.Base(clean) != SkillFile || path.Dir(clean) == ".":
		diagnostics.Add(SeverityError, "profile.skill_path_invalid", at, "skill path must name a skill folder's SKILL.md, e.g. skills/review/SKILL.md", 0, 0)
	case !skillNamePattern.MatchString(path.Base(path.Dir(clean))):
		diagnostics.Add(SeverityError, "profile.skill_name_invalid", at, fmt.Sprintf("skill folder %q must be 1-64 lowercase letters, digits, or hyphens, starting with a letter or digit", path.Base(path.Dir(clean))), 0, 0)
	default:
		return true
	}
	return false
}

func validateSkillSource(source SkillSource, at string, diagnostics *Diagnostics) {
	if parsed, err := url.Parse(source.Repo); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		diagnostics.Add(SeverityError, "profile.skill_source_invalid", at+".repo", "repo must be the source repository URL, e.g. https://github.com/owner/repo", 0, 0)
	}
	if !commitPattern.MatchString(source.Commit) {
		diagnostics.Add(SeverityError, "profile.skill_source_invalid", at+".commit", "commit must be the full 40-character lowercase hex commit the skill was copied from", 0, 0)
	}
	if source.Path != "" && (!safeGlobalResource(source.Path) || strings.Contains(source.Path, "\\")) {
		diagnostics.Add(SeverityError, "profile.skill_source_invalid", at+".path", "path must be relative within the source repository", 0, 0)
	}
	if !sha256Pattern.MatchString(source.SHA256) {
		diagnostics.Add(SeverityError, "profile.skill_source_invalid", at+".sha256", "sha256 must be the skill folder's 64-character lowercase hex tree digest", 0, 0)
	}
}

// SkillTreeFile is one file of a skill folder as the tree digest sees it.
type SkillTreeFile struct {
	Path   string // slash-separated, relative to the skill folder
	Mode   fs.FileMode
	SHA256 string
}

// SkillMode is the installed mode of a skill file: SkillModeExec when perm has any
// executable bit, else SkillModePlain.
func SkillMode(perm fs.FileMode) fs.FileMode {
	if perm&0o111 != 0 {
		return SkillModeExec
	}
	return SkillModePlain
}

// SkillTreeDigest is the sha256 over the sorted lines "<relpath>\0<mode>\0<file sha256>\n",
// with mode written as 0755 or 0644. It names a skill folder's exact installed content.
func SkillTreeDigest(files []SkillTreeFile) string {
	sorted := append([]SkillTreeFile(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	hash := sha256.New()
	for _, file := range sorted {
		_, _ = fmt.Fprintf(hash, "%s\x00%04o\x00%s\n", file.Path, SkillMode(file.Mode), file.SHA256) // hash writes never fail
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// ValidateSkillDocument checks a SKILL.md for the rules every qualified agent shares:
// valid UTF-8, YAML frontmatter with a non-empty description, and, when the frontmatter
// sets name, a name equal to the skill folder name. It returns the frontmatter name.
func ValidateSkillDocument(folder string, content []byte) (string, error) {
	if !utf8.Valid(content) {
		return "", fmt.Errorf("SKILL.md is not valid UTF-8")
	}
	front, err := skillFrontmatter(content)
	if err != nil {
		return "", err
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(front, &metadata); err != nil {
		return "", fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}
	if strings.TrimSpace(metadata.Description) == "" {
		return "", fmt.Errorf("SKILL.md frontmatter needs a non-empty description")
	}
	if metadata.Name != "" && metadata.Name != folder {
		return "", fmt.Errorf("SKILL.md frontmatter name %q must match its folder name %q", metadata.Name, folder)
	}
	return metadata.Name, nil
}

// skillFrontmatter returns the YAML between the leading and closing --- lines.
func skillFrontmatter(content []byte) ([]byte, error) {
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("SKILL.md needs YAML frontmatter starting with ---")
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return []byte(strings.Join(lines[1:index], "\n")), nil
		}
	}
	return nil, fmt.Errorf("SKILL.md frontmatter is not closed with ---")
}
