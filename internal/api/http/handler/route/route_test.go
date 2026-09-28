package routehandler

import (
	"testing"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
)

func TestRouteSyncPreviewResponseKeepsRulePartsSeparate(t *testing.T) {
	response := routeSyncPreviewResponse(routedto.RouteSyncPreview{Differences: []routedto.RouteSyncDiff{{
		Action: "added", RouteName: "api", Field: "route",
		Business: &routedto.RouteSyncRule{Match: "HTTPS Host(`api.example.test`)", Target: "http://api:8080"},
	}}})
	if len(response.Differences) != 1 || response.Differences[0].Business == nil ||
		response.Differences[0].Business.Match != "HTTPS Host(`api.example.test`)" ||
		response.Differences[0].Business.Target != "http://api:8080" || response.Differences[0].Traefik != nil {
		t.Fatalf("preview response = %+v", response.Differences)
	}
}

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
