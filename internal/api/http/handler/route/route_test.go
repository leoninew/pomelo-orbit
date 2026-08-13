package routehandler

import (
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
)

func TestTraefikRouteListResponseMapsPortView(t *testing.T) {
	entrypoints := []string{"websecure"}
	response := traefikRouteListResponse([]routeport.TraefikRouter{{
		Name:        "api@docker",
		Provider:    "docker",
		Status:      "enabled",
		Rule:        "Host(`api.example.test`)",
		Service:     "api-service",
		Entrypoints: entrypoints,
		TLS:         true,
	}})

	if response.Total != 1 || len(response.Items) != 1 {
		t.Fatalf("unexpected route list response: %+v", &response)
	}
	router := response.Items[0]
	if router.Name != "api@docker" || router.Provider != "docker" || router.Status != "enabled" || router.Rule != "Host(`api.example.test`)" || router.Service != "api-service" || len(router.Entrypoints) != 1 || router.Entrypoints[0] != "websecure" || !router.Tls {
		t.Fatalf("unexpected router response: %+v", router)
	}

	entrypoints[0] = "web"
	if router.Entrypoints[0] != "websecure" {
		t.Fatalf("expected generated proto response to be independent from application view, got %+v", router)
	}
}
