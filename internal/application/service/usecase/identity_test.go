package servicesvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestNormalizeIdentityOverlayPreservesExplicitClears(t *testing.T) {
	user, empty := "1000:1000", ""
	declaration := model.VersionComponent{User: &user, GroupAdd: []string{"988"}}
	component := model.ServiceComponent{User: &user, GroupAdd: []string{"988"}}
	if err := normalizeOverlay(&component, declaration); err != nil {
		t.Fatal(err)
	}
	if component.User != nil || component.GroupAdd != nil {
		t.Fatal("identical nonempty overrides did not inherit")
	}
	component.User, component.GroupAdd = &empty, []string{}
	if err := normalizeOverlay(&component, model.VersionComponent{}); err != nil {
		t.Fatal(err)
	}
	if component.User == nil || component.GroupAdd == nil {
		t.Fatal("explicit empty overrides were collapsed")
	}
}
