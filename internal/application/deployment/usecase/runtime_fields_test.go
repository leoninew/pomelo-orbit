package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeIncludesStructuredRuntimeFields(t *testing.T) {
	restartPolicy := "unless-stopped"
	service := Service{}
	compose, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version: model.Version{Id: "version-1"},
		Service: model.Service{InstanceKey: "default"},
		Components: []model.VersionComponent{{
			Name: "web", Image: "nginx", RestartPolicy: &restartPolicy,
			Tmpfs:   []model.VersionComponentTmpfs{{Target: "/tmp", SizeBytes: 1048576, Mode: "1777"}},
			Ulimits: []model.VersionComponentUlimit{{Name: "memlock", Soft: -1, Hard: -1}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"restart: unless-stopped", "/tmp:size=1048576,mode=1777", "memlock:", "soft: -1"} {
		if !strings.Contains(compose, expected) {
			t.Fatalf("compose missing %q:\n%s", expected, compose)
		}
	}
}
