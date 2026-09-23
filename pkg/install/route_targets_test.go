package install

import (
	"reflect"
	"sort"
	"testing"

	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/arieljcode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"gitlab.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"gitlab.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestRouteTargetsMatchAdapterNames(t *testing.T) {
	t.Parallel()
	names := []string{arieljcode.TargetName, claudecode.TargetName, codex.TargetName, hermes.TargetName, ohmypi.TargetName, openclaw.TargetName, opencode.TargetName, pi.TargetName}
	sort.Strings(names)
	if !reflect.DeepEqual(names, profilemango.RouteTargets) {
		t.Fatalf("adapter targets %v, route targets %v", names, profilemango.RouteTargets)
	}
}
