package routehandler

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRouteResponseIncludesTargetServiceCode(t *testing.T) {
	serviceId, serviceCode := "service-1", "api-main"
	view := model.RouteView{Route: model.Route{Id: "route-1", ServiceId: &serviceId}, ServiceCode: &serviceCode}
	response := routeResponse(view)
	encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&response)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["service_id"] != serviceId || payload["service_code"] != serviceCode {
		t.Fatalf("route response = %s", encoded)
	}
	view.ServiceCode = nil
	response = routeResponse(view)
	if response.ServiceCode != nil || response.GetServiceId() != serviceId {
		t.Fatalf("deleted service response = %+v", &response)
	}
}

func TestRouteSyncPreviewResponseKeepsRulePartsSeparate(t *testing.T) {
	response := routeSyncPreviewResponse(routedto.RouteSyncPreview{Items: []routedto.RouteSyncPlanItem{{
		Action: "publish", RouteId: "a", RouteName: "api",
		BusinessHash: "item-business", PublicationHash: "item-publication",
		Rule: &routedto.RouteSyncRule{Protocol: "https", Match: "Host(`api.example.test`)", Target: "http://api:8080"},
	}}})
	if len(response.Items) != 1 || response.Items[0].Rule == nil || response.Items[0].Action != "publish" || response.Items[0].RouteId != "a" ||
		response.Items[0].Rule.Protocol != "https" || response.Items[0].Rule.Match != "Host(`api.example.test`)" || response.Items[0].Rule.Target != "http://api:8080" ||
		response.Items[0].BusinessHash != "item-business" || response.Items[0].PublicationHash != "item-publication" {
		t.Fatalf("preview response = %+v", response.Items)
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
