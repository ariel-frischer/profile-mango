package install

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ariel-frischer/profile-mango/pkg/adapters/claudecode"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/codex"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/hermes"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/jcodefork"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/ohmypi"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/openclaw"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/opencode"
	"github.com/ariel-frischer/profile-mango/pkg/adapters/pi"
	"github.com/ariel-frischer/profile-mango/pkg/profilemango"
)

func TestRouteTargetsMatchAdapterNames(t *testing.T) {
	t.Parallel()
	names := []string{jcodefork.TargetName, claudecode.TargetName, codex.TargetName, hermes.TargetName, ohmypi.TargetName, openclaw.TargetName, opencode.TargetName, pi.TargetName}
	sort.Strings(names)
	if !reflect.DeepEqual(names, profilemango.RouteTargets) {
		t.Fatalf("adapter targets %v, route targets %v", names, profilemango.RouteTargets)
	}
}
