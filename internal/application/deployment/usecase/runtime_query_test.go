package deploymentsvc

import (
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
)

func TestApplyContainerComponentIds(t *testing.T) {
	t.Parallel()
	containers := []deploymentdto.RuntimeContainer{
		{Service: "web"},
		{Service: "worker"},
	}

	applyContainerComponentIds(containers, "version-1", map[string]string{
		"web": "component-web",
	})

	if got, want := containers[0].VersionId, "version-1"; got != want {
		t.Fatalf("VersionID: got %q want %q", got, want)
	}
	if got, want := containers[0].ComponentId, "component-web"; got != want {
		t.Fatalf("ComponentID: got %q want %q", got, want)
	}
	if got, want := containers[1].VersionId, "version-1"; got != want {
		t.Fatalf("VersionID: got %q want %q", got, want)
	}
	if containers[1].ComponentId != "" {
		t.Fatalf("unexpected ComponentID for %+v", containers[1])
	}
}
