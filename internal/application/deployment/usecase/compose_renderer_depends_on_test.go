package deploymentsvc

import (
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComponentPreservesDependencyConditions(t *testing.T) {
	service, _, err := renderVersionComponentService(model.VersionComponent{
		Name: "api", Image: "nginx",
		Dependencies: []model.VersionComponentDependency{
			{Name: "mysql", Condition: "service_healthy"},
			{Name: "redis", Condition: "service_started"},
		},
	}, nil, "demo", "", nil)
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
