package gatewaysvc

import (
	"testing"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestFindGatewayDashboardRouteAcceptsCustomAndManagedTargets(t *testing.T) {
	applicationCode := "traefik"
	runtimeServiceID := "traefik-service"
	custom := model.Route{
		Id: "custom-dashboard", Protocol: "http", Name: "traefik",
		TargetUrl: gatewayDashboardTargetURL(applicationCode),
	}
	managed := model.Route{
		Id: "managed-dashboard", Protocol: "http", Name: "dashboard",
		TargetUrl: "traefik-default/traefik/http8080", ServiceId: stringRef(runtimeServiceID),
		ComponentName: stringRef("traefik"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(8080),
	}

	foundCustom, err := findGatewayDashboardRoute([]model.Route{custom, ordinaryHTTPRoute()}, applicationCode, runtimeServiceID)
	if err != nil {
		t.Fatal(err)
	}
	if foundCustom.Id != custom.Id {
		t.Fatalf("custom dashboard = %#v", foundCustom)
	}

	foundManaged, err := findGatewayDashboardRoute([]model.Route{ordinaryHTTPRoute(), managed}, applicationCode, runtimeServiceID)
	if err != nil {
		t.Fatal(err)
	}
	if foundManaged.Id != managed.Id || foundManaged.ServiceId == nil || foundManaged.TargetUrl != managed.TargetUrl {
		t.Fatalf("managed dashboard = %#v", foundManaged)
	}
}

func TestFindGatewayDashboardRouteRejectsMissingAndDuplicateTargets(t *testing.T) {
	applicationCode := "traefik"
	runtimeServiceID := "traefik-service"
	custom := model.Route{
		Id: "custom-dashboard", Protocol: "http", TargetUrl: gatewayDashboardTargetURL(applicationCode),
	}
	managed := model.Route{
		Id: "managed-dashboard", Protocol: "http", ServiceId: stringRef(runtimeServiceID),
		ComponentName: stringRef("traefik"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(8080),
	}

	if _, err := findGatewayDashboardRoute([]model.Route{ordinaryHTTPRoute()}, applicationCode, runtimeServiceID); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("missing dashboard error = %v", err)
	}
	if _, err := findGatewayDashboardRoute([]model.Route{custom, managed}, applicationCode, runtimeServiceID); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("duplicate dashboard error = %v", err)
	}
}

func TestIsGatewayDashboardRouteIgnoresOtherRouteTypes(t *testing.T) {
	applicationCode := "traefik"
	runtimeServiceID := "traefik-service"
	cases := []model.Route{
		ordinaryHTTPRoute(),
		{Id: "tcp", Protocol: "tcp", ServiceId: stringRef(runtimeServiceID), ComponentName: stringRef("traefik"), EndpointProtocol: stringRef("tcp"), EndpointContainerPort: intRef(8080)},
		{Id: "other-service", Protocol: "http", ServiceId: stringRef("other-service"), ComponentName: stringRef("traefik"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(8080)},
		{Id: "other-component", Protocol: "http", ServiceId: stringRef(runtimeServiceID), ComponentName: stringRef("app"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(8080)},
		{Id: "other-port", Protocol: "http", ServiceId: stringRef(runtimeServiceID), ComponentName: stringRef("traefik"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(80)},
		{Id: "custom-other-host", Protocol: "http", TargetUrl: "http://other-traefik:8080"},
	}
	for _, route := range cases {
		if isGatewayDashboardRoute(route, applicationCode, runtimeServiceID) {
			t.Fatalf("route %#v must not be the Gateway dashboard", route)
		}
	}
}

func ordinaryHTTPRoute() model.Route {
	return model.Route{
		Id: "web", Name: "web", Protocol: "http", Domain: "web.example.test",
		TargetUrl: "web-default/web/http80", ServiceId: stringRef("web-service"),
		ComponentName: stringRef("web"), EndpointProtocol: stringRef("http"), EndpointContainerPort: intRef(80),
	}
}
