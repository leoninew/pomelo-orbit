package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComponentPreservesDependencyConditions(t *testing.T) {
	service, _, err := renderVersionComponentService(model.VersionComponent{
		Name: "api", Image: "nginx",
		Dependencies: []model.VersionComponentDependency{
			{Name: "mysql", Condition: "service_healthy"},
			{Name: "redis", Condition: "service_started"},
		},
	}, "demo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	depends, ok := service["depends_on"].(map[string]map[string]string)
	if !ok {
		t.Fatalf("depends type = %T", service["depends_on"])
	}
	if depends["mysql"]["condition"] != "service_healthy" {
		t.Fatalf("mysql condition = %#v", depends["mysql"])
	}
}

func TestRenderComposeDeclaresNamedVolumes(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version: model.Version{Id: "version-1"},
		Components: []model.VersionComponent{{
			Name:  "db",
			Image: "postgres:16",
			Mounts: []model.VersionComponentMount{{
				SourceType: mountSourceNamedVolume,
				Source:     "postgres-data",
				Target:     "/var/lib/postgresql/data",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compose, "volumes:\n    postgres-data: {}") {
		t.Fatalf("compose must declare named volume:\n%s", compose)
	}
}
