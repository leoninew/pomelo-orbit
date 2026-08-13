package gatewaysvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestBuildManagedGatewayComponentExposesDashboardApiOnLoopback(t *testing.T) {
	component, err := buildManagedGatewayComponent(
		model.GatewayConfig{},
		[]model.VersionComponent{{
			Id:    "traefik-component",
			Name:  "traefik",
			Image: "traefik:3.6",
		}},
		nil,
		"traefik",
		"traefik",
	)
	if err != nil {
		t.Fatalf("build managed gateway component: %v", err)
	}

	var api *model.VersionComponentEndpoint
	for index := range component.Endpoints {
		if component.Endpoints[index].Protocol == "http" && component.Endpoints[index].ContainerPort == 8080 {
			api = &component.Endpoints[index]
			break
		}
	}
	if api == nil {
		t.Fatal("managed gateway does not declare an API endpoint")
	}
	if api.Protocol != "http" || api.ContainerPort != 8080 || api.Mode != "local" {
		t.Fatalf("unexpected API endpoint: %#v", *api)
	}
	if api.BindAddress == nil || *api.BindAddress != "127.0.0.1" {
		t.Fatalf("API endpoint must bind loopback, got %#v", api.BindAddress)
	}
	if api.ListenPort == nil || *api.ListenPort != 8080 {
		t.Fatalf("API endpoint must listen on 8080, got %#v", api.ListenPort)
	}
}
