package servicehandler

import (
	"testing"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestServiceViewResponseUsesBoundComponentImage(t *testing.T) {
	listenPort := 18080
	response := serviceViewResponse(servicedto.ServiceView{
		Service:          model.Service{Id: "service-1", Code: "ragflow-default"},
		ApplicationCode:  "ragflow",
		Env:              []model.ServiceEnv{{Key: "SHARED_VALUE", Value: "value"}},
		EffectiveError:   "component mysql environment MYSQL_DATABASE: requires service environment MYSQL_DATABASE",
		ActiveDeployment: true,
		Components: []model.ServiceComponent{{
			Id: "service-component-1", ComponentName: "api", SourceVersionComponentId: "version-component-1",
		}},
		ComponentDefinitions: []model.VersionComponent{{
			Id: "version-component-1", Image: "nginx:latest",
		}},
		EffectiveComponents: []model.EffectiveServiceComponent{{
			ServiceComponentId: "service-component-1",
			Endpoints: []model.VersionComponentEndpoint{{
				Name: "http", Protocol: "http", ContainerPort: 8080, Mode: "local", ListenPort: &listenPort,
			}},
		}},
	})
	if len(response.Components) != 1 {
		t.Fatalf("component responses = %d, want 1", len(response.Components))
	}
	if got := response.Components[0].Image; got != "nginx:latest" {
		t.Fatalf("component image = %q, want nginx:latest", got)
	}
	if got := response.Components[0].ContainerName; got != "ragflow-api" {
		t.Fatalf("container name = %q, want ragflow-api", got)
	}
	if len(response.Components[0].EffectiveEndpoints) != 1 {
		t.Fatalf("effective endpoints = %#v", response.Components[0].EffectiveEndpoints)
	}
	endpoint := response.Components[0].EffectiveEndpoints[0]
	if endpoint.Name != "http" || endpoint.Protocol != "http" || endpoint.ContainerPort != 8080 || endpoint.Mode != "local" || endpoint.ListenPort == nil || *endpoint.ListenPort != 18080 {
		t.Fatalf("effective endpoint = %#v", endpoint)
	}
	if len(response.Env) != 1 || response.Env[0].Key != "SHARED_VALUE" || response.Env[0].Value != "value" {
		t.Fatalf("service environment = %#v", response.Env)
	}
	if response.EffectiveError == "" {
		t.Fatal("service effective error is missing")
	}
	if !response.ActiveDeployment {
		t.Fatal("service active deployment is missing")
	}
	if response.Code != "ragflow-default" {
		t.Fatalf("service code = %q", response.Code)
	}
}

func TestServiceComponentDetailResponseReturnsVersionAndServiceValues(t *testing.T) {
	pullPolicy := "never"
	restartPolicy := "no"
	response := serviceComponentDetailResponse(servicedto.ServiceComponentDetail{
		ServiceComponent: model.ServiceComponent{
			Id: "service-component-1", Entrypoint: []string{"/custom-entrypoint"}, Command: []string{"--serve"},
			PullPolicy: &pullPolicy, RestartPolicy: &restartPolicy,
		},
		VersionComponent: model.VersionComponent{
			Id: "version-component-1", Name: "mysql", Image: "mysql:8",
			Entrypoint: []string{"/version-entrypoint"}, Command: []string{"--default"},
			PullPolicy: "always",
		},
	})
	if response.ServiceComponent == nil || response.VersionComponent == nil {
		t.Fatal("expected Version and Service Component source views")
	}
	if response.ServiceComponent.Entrypoint == nil || *response.ServiceComponent.Entrypoint != "/custom-entrypoint" ||
		response.ServiceComponent.Command == nil || *response.ServiceComponent.Command != "--serve" ||
		response.ServiceComponent.PullPolicy == nil || *response.ServiceComponent.PullPolicy != pullPolicy ||
		response.ServiceComponent.RestartPolicy == nil || *response.ServiceComponent.RestartPolicy != restartPolicy {
		t.Fatalf("service component values = %#v", response.ServiceComponent)
	}
	if response.VersionComponent.Entrypoint != "/version-entrypoint" ||
		response.VersionComponent.Command != "--default" ||
		response.VersionComponent.PullPolicy != "always" {
		t.Fatalf("version component values = %#v", response.VersionComponent)
	}
}

func TestServiceComponentMountOverlayMapsHostPathMode(t *testing.T) {
	source := "D:/SourceCodes/mywork/PomeloOrbit-go/data/backup/bge-m3"
	sourceIsHostPath := true
	input := serviceComponentOverlayInput(&servicev1.ServiceComponentOverlayUpdateReq{
		Mounts: []*servicev1.ServiceComponentMountOverlay{{
			Target: "/data", Source: &source, SourceIsHostPath: &sourceIsHostPath, State: string(model.ServiceComponentOverlayOverride),
		}},
	})
	if len(input.Mounts) != 1 || input.Mounts[0].SourceIsHostPath == nil || !*input.Mounts[0].SourceIsHostPath {
		t.Fatalf("mount input lost host path mode: %#v", input.Mounts)
	}
	response := serviceComponentResponse(model.ServiceComponent{Mounts: input.Mounts})
	if len(response.Mounts) != 1 || response.Mounts[0].SourceIsHostPath == nil || !*response.Mounts[0].SourceIsHostPath {
		t.Fatalf("mount response lost host path mode: %#v", response.Mounts)
	}
}
