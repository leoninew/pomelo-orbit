package servicehandler

import (
	"testing"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestServiceViewResponseUsesBoundComponentImage(t *testing.T) {
	response := serviceViewResponse(servicedto.ServiceView{
		Service: model.Service{Id: "service-1"},
		Env:     []model.ServiceEnv{{Key: "SHARED_VALUE", Value: "value"}},
		Components: []model.ServiceComponent{{
			Id: "service-component-1", SourceVersionComponentId: "version-component-1",
		}},
		ComponentDefinitions: []model.VersionComponent{{
			Id: "version-component-1", Image: "nginx:latest",
		}},
	})
	if len(response.Components) != 1 {
		t.Fatalf("component responses = %d, want 1", len(response.Components))
	}
	if got := response.Components[0].Image; got != "nginx:latest" {
		t.Fatalf("component image = %q, want nginx:latest", got)
	}
	if len(response.Env) != 1 || response.Env[0].Key != "SHARED_VALUE" || response.Env[0].Value != "value" {
		t.Fatalf("service environment = %#v", response.Env)
	}
}
