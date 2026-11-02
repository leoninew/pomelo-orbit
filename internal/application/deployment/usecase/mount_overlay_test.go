package deploymentsvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestServiceMountOverlayChangesEffectivePlanHash(t *testing.T) {
	declaration := model.VersionComponent{Id: "version-component", Name: "web", Mounts: []model.VersionComponentMount{{Target: "/config", SourceType: "controlled_file", Source: "./config", Content: "version", Mode: "0644", IgnoreIfExists: true}}}
	overlay := model.ServiceComponent{Id: "component", ServiceId: "service", SourceVersionComponentId: declaration.Id, ComponentName: declaration.Name}
	version := model.Version{Id: "version"}
	service := model.Service{Id: "service", VersionId: version.Id}
	_, originalHash, err := BuildEffectiveServicePlan(model.Application{}, version, service, []model.VersionComponent{declaration}, []model.ServiceComponent{overlay}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	empty := ""
	disabled := false
	overlay.Mounts = []model.ServiceComponentMount{{Target: "/config", State: model.ServiceComponentOverlayOverride, Content: &empty, IgnoreIfExists: &disabled}}
	plan, updatedHash, err := BuildEffectiveServicePlan(model.Application{}, version, service, []model.VersionComponent{declaration}, []model.ServiceComponent{overlay}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	mount := plan.Components[0].Mounts[0]
	if originalHash == updatedHash || mount.Content != "" || mount.IgnoreIfExists || mount.ReadOnly || mount.Mode != "0644" {
		t.Fatalf("effective mount = %#v, hashes %q %q", mount, originalHash, updatedHash)
	}
}
