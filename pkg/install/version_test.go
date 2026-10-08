package install

import (
	"errors"
	"strings"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/codex"
)

func TestTildeRangeTable(t *testing.T) {
	tests := map[string]struct {
		qualified string
		wantRange string
		in        []string
		out       []string
	}{
		"semver":         {qualified: "2.1.278", wantRange: ">=2.1.278 <2.2.0", in: []string{"2.1.278", "2.1.280", "2.1.999"}, out: []string{"2.1.277", "2.2.0", "3.0.0", "1.9.999"}},
		"zero major":     {qualified: "0.157.1", wantRange: ">=0.157.1 <0.158.0", in: []string{"0.157.1", "0.157.7"}, out: []string{"0.155.1", "0.153.9", "1.154.0"}},
		"calver":         {qualified: "2026.9.5", wantRange: ">=2026.9.5 <2026.10.0", in: []string{"2026.9.5", "2026.9.30"}, out: []string{"2026.9.4", "2026.10.1", "2027.9.5"}},
		"calver dec":     {qualified: "2026.12.1", wantRange: ">=2026.12.1 <2026.13.0", in: []string{"2026.12.9"}, out: []string{"2027.1.0"}},
		"dev suffix":     {qualified: "0.83.909-dev (ca8017a3a)", wantRange: ">=0.83.909 <0.84.0", in: []string{"0.83.910"}, out: []string{"0.84.0"}},
		"multi digit":    {qualified: "1.18.31", wantRange: ">=1.18.31 <1.19.0", in: []string{"1.18.100"}, out: []string{"1.18.4", "1.2.40"}},
		"minor rollover": {qualified: "1.9.0", wantRange: ">=1.9.0 <1.10.0", in: []string{"1.9.9"}, out: []string{"1.10.0"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			versionRange, err := TildeRange(test.qualified)
			if err != nil || versionRange.String() != test.wantRange {
				t.Fatalf("TildeRange(%q) = %q, %v; want %q", test.qualified, versionRange, err, test.wantRange)
			}
			for _, value := range test.in {
				assertRangeContains(t, versionRange, value, true)
			}
			for _, value := range test.out {
				assertRangeContains(t, versionRange, value, false)
			}
		})
	}
}

func assertRangeContains(t *testing.T, versionRange VersionRange, value string, want bool) {
	t.Helper()
	version, ok := ParseVersion(value)
	if !ok {
		t.Fatalf("ParseVersion(%q) failed", value)
	}
	if got := versionRange.Contains(version); got != want {
		t.Fatalf("%s contains %s = %v, want %v", versionRange, value, got, want)
	}
}

func TestParseVersionTable(t *testing.T) {
	tests := map[string]struct {
		text string
		want [3]int
		ok   bool
	}{
		"codex output":  {text: "codex-cli 0.155.1", want: [3]int{0, 155, 1}, ok: true},
		"claude output": {text: "2.1.280 (Claude Code)", want: [3]int{2, 1, 280}, ok: true},
		"multi line":    {text: "tool\nversion 1.2.3\nbuild 9.9.9", want: [3]int{1, 2, 3}, ok: true},
		"no version":    {text: "unknown", ok: false},
		"two parts":     {text: "v1.2", ok: false},
		"empty":         {text: "", ok: false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := ParseVersion(test.text)
			if ok != test.ok || got != test.want {
				t.Fatalf("ParseVersion(%q) = %v, %v; want %v, %v", test.text, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestParseVersionRangeTable(t *testing.T) {
	tests := map[string]struct {
		value   string
		want    string
		wantErr bool
	}{
		"valid":          {value: ">=1.2.3 <2.0.0", want: ">=1.2.3 <2.0.0"},
		"extra spaces":   {value: "  >=1.2.3   <1.3.0 ", want: ">=1.2.3 <1.3.0"},
		"empty range":    {value: ">=1.2.3 <1.2.3", wantErr: true},
		"inverted":       {value: ">=2.0.0 <1.0.0", wantErr: true},
		"missing upper":  {value: ">=1.2.3", wantErr: true},
		"wrong operator": {value: ">1.2.3 <2.0.0", wantErr: true},
		"not a version":  {value: ">=a <b", wantErr: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseVersionRange(test.value)
			if (err != nil) != test.wantErr {
				t.Fatalf("ParseVersionRange(%q) error = %v, wantErr %v", test.value, err, test.wantErr)
			}
			if err == nil && got.String() != test.want {
				t.Fatalf("ParseVersionRange(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestCompatibleRangeHonorsAdapterOverride(t *testing.T) {
	versionRange, err := CompatibleRange(AdapterMetadata{Version: "1.2.3", CompatibleRange: ">=1.2.0 <1.4.0"})
	if err != nil || versionRange.String() != ">=1.2.0 <1.4.0" {
		t.Fatalf("override range = %q, %v", versionRange, err)
	}
	if _, err := CompatibleRange(AdapterMetadata{Version: "1.2.3", CompatibleRange: "any"}); err == nil {
		t.Fatal("invalid override accepted")
	}
}

func TestCheckVersionTable(t *testing.T) {
	claude := claudeCodeAdapter{}.Metadata()
	codexMetadata := codexAdapter{}.Metadata()
	tests := map[string]struct {
		metadata  AdapterMetadata
		detection VersionDetection
		want      VersionCheck
	}{
		"claude patch update in range": {metadata: claude, detection: VersionDetection{Binary: "claude", Found: true, Output: "2.1.280 (Claude Code)"},
			want: VersionCheck{Binary: "claude", Qualified: claudecode.TargetVersion, Range: ">=2.1.278 <2.2.0", Detected: "2.1.280", Status: VersionInRange}},
		"codex minor update out of range": {metadata: codexMetadata, detection: VersionDetection{Binary: "codex", Found: true, Output: "codex-cli 0.155.1"},
			want: VersionCheck{Binary: "codex", Qualified: codex.TargetVersion, Range: ">=0.157.1 <0.158.0", Detected: "0.155.1", Status: VersionOutOfRange}},
		"not found": {metadata: codexMetadata, detection: VersionDetection{Binary: "codex"},
			want: VersionCheck{Binary: "codex", Qualified: codex.TargetVersion, Range: ">=0.157.1 <0.158.0", Status: VersionNotFound}},
		"probe failed": {metadata: codexMetadata, detection: VersionDetection{Binary: "codex", Found: true, Output: "codex-cli 0.157.1", Err: errors.New("exit status 1")},
			want: VersionCheck{Binary: "codex", Qualified: codex.TargetVersion, Range: ">=0.157.1 <0.158.0", Status: VersionUnknown}},
		"unparseable output": {metadata: codexMetadata, detection: VersionDetection{Binary: "codex", Found: true, Output: "hello"},
			want: VersionCheck{Binary: "codex", Qualified: codex.TargetVersion, Range: ">=0.157.1 <0.158.0", Status: VersionUnknown}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := CheckVersion(test.metadata, test.detection)
			if err != nil || got != test.want {
				t.Fatalf("CheckVersion = %#v, %v; want %#v", got, err, test.want)
			}
		})
	}
}

func fakeDetector(output string, found bool) VersionDetector {
	return func(target Target) VersionDetection {
		return VersionDetection{Binary: target.Name, Found: found, Output: output}
	}
}

func TestBuildPlanRecordsVersionCheckWithoutBlocking(t *testing.T) {
	tests := map[string]struct {
		detect      VersionDetector
		wantStatus  string
		wantWarning string
	}{
		"in range":     {detect: fakeDetector("codex-cli 0.157.3", true), wantStatus: VersionInRange},
		"out of range": {detect: fakeDetector("codex-cli 0.155.1", true), wantStatus: VersionOutOfRange, wantWarning: "install.version_out_of_range"},
		"not found":    {detect: fakeDetector("", false), wantStatus: VersionNotFound, wantWarning: "install.version_not_found"},
		"unreadable":   {detect: fakeDetector("garbage", true), wantStatus: VersionUnknown, wantWarning: "install.version_unknown"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request, _ := codexTestRequest(t)
			request.Registry = DefaultRegistry()
			request.DetectVersion = test.detect
			plan, err := BuildPlan(request)
			if err != nil {
				t.Fatal(err)
			}
			target := plan.Targets[0]
			if plan.Status != StatusReady || target.Status != StatusReady {
				t.Fatalf("version check changed status: plan=%s target=%s", plan.Status, target.Status)
			}
			if target.VersionCheck == nil || target.VersionCheck.Status != test.wantStatus {
				t.Fatalf("version check = %#v, want status %s", target.VersionCheck, test.wantStatus)
			}
			assertVersionWarning(t, target, test.wantWarning)
			data, err := plan.JSON()
			if err != nil || !strings.Contains(string(data), `"status": "`+test.wantStatus+`"`) {
				t.Fatalf("plan JSON lacks version status %s: %v\n%s", test.wantStatus, err, data)
			}
		})
	}
}

func assertVersionWarning(t *testing.T, target TargetPlan, code string) {
	t.Helper()
	for _, diagnostic := range target.Diagnostics {
		if strings.HasPrefix(diagnostic.Code, "install.version_") && diagnostic.Code != code {
			t.Fatalf("unexpected version diagnostic %#v", diagnostic)
		}
	}
	if code != "" && !hasDiagnostic(target.Diagnostics, code) {
		t.Fatalf("missing %s warning: %#v", code, target.Diagnostics)
	}
}

func TestPlanIDIsDeterministicForIdenticalDetection(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Registry = DefaultRegistry()
	request.DetectVersion = fakeDetector("codex-cli 0.155.1", true)
	first, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanID != second.PlanID {
		t.Fatalf("identical inputs produced different plan IDs: %s != %s", first.PlanID, second.PlanID)
	}
	request.DetectVersion = fakeDetector("codex-cli 0.157.1", true)
	changed, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if changed.PlanID == first.PlanID {
		t.Fatal("a different installed version kept the same plan ID")
	}
}

func TestBuildPlanSkipsDetectionForBlockedTargets(t *testing.T) {
	request, _ := codexTestRequest(t)
	request.Registry = DefaultRegistry()
	request.Targets[0].Target.Version = "0.155.1"
	called := false
	request.DetectVersion = func(Target) VersionDetection { called = true; return VersionDetection{} }
	plan, err := BuildPlan(request)
	if err != nil {
		t.Fatal(err)
	}
	if called || plan.Targets[0].VersionCheck != nil || plan.Targets[0].Status == StatusReady {
		t.Fatalf("explicit unqualified version must stay exact and blocked without detection: called=%v plan=%#v", called, plan.Targets[0])
	}
}
