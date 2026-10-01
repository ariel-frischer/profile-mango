package install

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

// Version check statuses recorded in a target plan.
const (
	VersionInRange    = "in-range"
	VersionOutOfRange = "out-of-range"
	VersionNotFound   = "not-found"
	VersionUnknown    = "unknown"
)

var versionPattern = regexp.MustCompile(`\d+\.\d+\.\d+`)

// VersionRange is a half-open [Min, Below) range of numeric major.minor.patch
// versions. CalVer year.month.patch versions compare the same way.
type VersionRange struct {
	Min   [3]int
	Below [3]int
}

// ParseVersion extracts the first major.minor.patch number from text such as
// "codex-cli 0.155.1" or "2.1.280 (Claude Code)".
func ParseVersion(text string) ([3]int, bool) {
	match := versionPattern.FindString(text)
	if match == "" {
		return [3]int{}, false
	}
	var result [3]int
	for index, part := range strings.SplitN(match, ".", 3) {
		value, err := strconv.Atoi(part)
		if err != nil {
			return [3]int{}, false
		}
		result[index] = value
	}
	return result, true
}

// TildeRange allows patch updates of a qualified version: >= qualified and
// < the next minor. For CalVer year.month.patch it keeps the same year.month.
func TildeRange(qualified string) (VersionRange, error) {
	version, ok := ParseVersion(qualified)
	if !ok {
		return VersionRange{}, fmt.Errorf("qualified version %q has no major.minor.patch number", qualified)
	}
	return VersionRange{Min: version, Below: [3]int{version[0], version[1] + 1, 0}}, nil
}

// ParseVersionRange parses the ">=A <B" form produced by VersionRange.String.
func ParseVersionRange(value string) (VersionRange, error) {
	fields := strings.Fields(value)
	if len(fields) != 2 || !strings.HasPrefix(fields[0], ">=") || !strings.HasPrefix(fields[1], "<") {
		return VersionRange{}, fmt.Errorf("version range %q must use \">=A <B\" syntax", value)
	}
	minimum, minOK := ParseVersion(strings.TrimPrefix(fields[0], ">="))
	below, belowOK := ParseVersion(strings.TrimPrefix(fields[1], "<"))
	if !minOK || !belowOK || compareVersions(minimum, below) >= 0 {
		return VersionRange{}, fmt.Errorf("version range %q must name a non-empty major.minor.patch range", value)
	}
	return VersionRange{Min: minimum, Below: below}, nil
}

// CompatibleRange returns an adapter's tested version range: its explicit
// CompatibleRange when set, otherwise the tilde range of its qualified version.
func CompatibleRange(metadata AdapterMetadata) (VersionRange, error) {
	if metadata.CompatibleRange != "" {
		return ParseVersionRange(metadata.CompatibleRange)
	}
	return TildeRange(metadata.Version)
}

func (versionRange VersionRange) String() string {
	return ">=" + formatVersion(versionRange.Min) + " <" + formatVersion(versionRange.Below)
}

// Contains reports whether version lies inside the range.
func (versionRange VersionRange) Contains(version [3]int) bool {
	return compareVersions(version, versionRange.Min) >= 0 && compareVersions(version, versionRange.Below) < 0
}

func formatVersion(version [3]int) string {
	return fmt.Sprintf("%d.%d.%d", version[0], version[1], version[2])
}

func compareVersions(left, right [3]int) int {
	for index := range left {
		if left[index] != right[index] {
			if left[index] < right[index] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// VersionDetection is what a VersionDetector observed for one target's
// command. Output is the raw "--version" text; Err reports a failed probe.
type VersionDetection struct {
	Binary string
	Path   string
	Found  bool
	Output string
	Err    error
}

// VersionDetector looks up a target's installed command version. It is
// injected by the caller so planning itself never runs a process.
type VersionDetector func(Target) VersionDetection

// VersionCheck records how the installed agent version relates to the
// adapter's tested range. It is part of the plan and its plan ID.
type VersionCheck struct {
	Binary    string `json:"binary"`
	Qualified string `json:"qualified"`
	Range     string `json:"range"`
	Detected  string `json:"detected,omitempty"`
	Status    string `json:"status"`
}

// CheckVersion classifies one detection against an adapter's tested range.
func CheckVersion(metadata AdapterMetadata, detection VersionDetection) (VersionCheck, error) {
	versionRange, err := CompatibleRange(metadata)
	if err != nil {
		return VersionCheck{}, err
	}
	check := VersionCheck{Binary: detection.Binary, Qualified: metadata.Version, Range: versionRange.String(), Status: VersionUnknown}
	if !detection.Found {
		check.Status = VersionNotFound
		return check, nil
	}
	detected, ok := ParseVersion(detection.Output)
	if detection.Err != nil || !ok {
		return check, nil
	}
	check.Detected = formatVersion(detected)
	check.Status = VersionOutOfRange
	if versionRange.Contains(detected) {
		check.Status = VersionInRange
	}
	return check, nil
}

// attachVersionCheck records the installed agent version after adapter and
// conflict checks, before manifest serialization. Missing, unreadable, or
// out-of-range versions add a warning without blocking installation.
func attachVersionCheck(targetPlan *TargetPlan, detect VersionDetector) {
	if detect == nil {
		return
	}
	check, err := CheckVersion(targetPlan.Metadata, detect(targetPlan.Target))
	if err != nil {
		targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.version_range_invalid", "target", err.Error(), 0, 0)
		return
	}
	targetPlan.VersionCheck = &check
	if message := versionWarning(check); message != "" {
		targetPlan.Diagnostics.Add(profilemango.SeverityWarning, "install.version_"+strings.ReplaceAll(check.Status, "-", "_"), "target", message, 0, 0)
	}
}

// installedTarget identifies the installed agent without changing the
// qualified target used to select adapters and generate settings.
func (targetPlan TargetPlan) installedTarget() Target {
	target := targetPlan.Target
	if check := targetPlan.VersionCheck; check != nil && check.Detected != "" {
		target.Version = check.Detected
	}
	return target
}

func versionWarning(check VersionCheck) string {
	switch check.Status {
	case VersionOutOfRange:
		return fmt.Sprintf("installed %s %s is outside the tested range %s; settings are written for %s and may not be honored", check.Binary, check.Detected, check.Range, check.Qualified)
	case VersionNotFound:
		return fmt.Sprintf("%s not found on PATH; installing settings for %s", check.Binary, check.Qualified)
	case VersionUnknown:
		return fmt.Sprintf("could not read the installed %s version; installing settings for %s", check.Binary, check.Qualified)
	default:
		return ""
	}
}
