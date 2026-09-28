package routehandler

import (
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	routev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/route"
)

func TestRouteSyncChangesParsesManualCertificate(t *testing.T) {
	pem := "-----BEGIN CERTIFICATE-----\nYQ==\n-----END CERTIFICATE-----\n-----BEGIN PRIVATE KEY-----\nYg==\n-----END PRIVATE KEY-----\n"
	changes, err := routeSyncChanges([]*routev1.RouteSyncChange{{
		RouteId: "route-1", Certificate: &routev1.RouteSyncCertificateChange{Mode: "manual", Pem: pem},
	}})
	if err != nil || len(changes) != 1 || changes[0].Enabled != nil || changes[0].Certificate == nil || changes[0].Certificate.CertPEM == "" || changes[0].Certificate.CertKey == "" {
		t.Fatalf("mapped changes = %+v, err=%v", changes, err)
	}
	_, err = routeSyncChanges([]*routev1.RouteSyncChange{{
		RouteId: "route-1", Certificate: &routev1.RouteSyncCertificateChange{Mode: "manual", Pem: "invalid"},
	}})
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("invalid PEM error = %v", err)
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
