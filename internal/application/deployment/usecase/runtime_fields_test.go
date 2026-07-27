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
	tmpfsJSON := `[{"target":"/tmp","size_bytes":1048576,"mode":"1777"}]`
	ulimitsJSON := `[{"name":"memlock","soft":-1,"hard":-1}]`
	service := Service{}
	compose, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version: model.Version{Id: "version-1"},
		Service: model.Service{InstanceKey: "default"},
		Components: []model.VersionComponent{{
			Name: "web", Image: "nginx", RestartPolicy: &restartPolicy, TmpfsJSON: &tmpfsJSON, UlimitsJSON: &ulimitsJSON,
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
