package routesvc

import (
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestCompareSyncStateIncludesCustomRouteFields(t *testing.T) {
	routes := []model.Route{{
		Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://api:8080",
		Enabled: true, HTTPSEnabled: true, CertType: certTypeLetsEncrypt, AcmeChallenge: acmeChallengeDNS,
	}}
	routers := []routeport.TraefikRouter{{
		Name: "api-route@rest", Provider: "rest", Rule: "Host(`api.example.test`)", Service: "api-service@rest",
		Entrypoints: []string{"web"}, TLS: false, TLSConfig: `{"certResolver":"letsencrypt"}`,
	}}
	services := []routeport.TraefikService{{
		Name: "api-service@rest", Provider: "rest", Protocol: "http", Servers: []string{"http://old-api:8080"},
	}}

	preview := compareSyncState(routes, routers, services)
	if preview.Matched {
		t.Fatal("expected target and protocol differences")
	}
	if len(preview.Differences) != 1 {
		t.Fatalf("differences = %+v, want one route difference", preview.Differences)
	}
	difference := preview.Differences[0]
	if difference.Action != "modified" || difference.RouteName != "api" || difference.Field != "route" || difference.BusinessValue != "HTTPS Host(`api.example.test`) -> http://api:8080" || difference.TraefikValue != "HTTP Host(`api.example.test`) -> http://old-api:8080" {
		t.Fatalf("difference = %+v, want one readable route difference", difference)
	}

	matchingRouters := []routeport.TraefikRouter{{
		Name: "api-route@rest", Provider: "rest", Rule: "Host(`api.example.test`)", Service: "api-service@rest",
		Entrypoints: []string{"websecure"}, TLS: true, TLSConfig: `{"certResolver":"letsencrypt-dns","options":"default"}`,
	}}
	matchingServices := []routeport.TraefikService{{
		Name: "api-service@rest", Provider: "rest", Protocol: "http", Servers: []string{"http://api:8080"},
	}}
	if preview = compareSyncState(routes, matchingRouters, matchingServices); !preview.Matched || len(preview.Differences) != 0 {
		t.Fatalf("matching preview = %+v", preview)
	}
}

func TestCompareSyncStateIgnoresRouteCertificatesAndIncludesUnmanagedTraefikEntries(t *testing.T) {
	certPEM := "CERT"
	certKey := "KEY"
	routes := []model.Route{{
		Name: "secure", Protocol: "http", Domain: "secure.example.test", PathPrefix: "/", TargetUrl: "http://secure:8080",
		Enabled: true, HTTPSEnabled: true, CertType: "manual", CertPEM: &certPEM, CertKey: &certKey,
	}}
	routers := []routeport.TraefikRouter{{
		Name: "secure-route@rest", Provider: "rest", Rule: "Host(`secure.example.test`)", Service: "secure-service@rest",
		Entrypoints: []string{"websecure"}, TLS: true, TLSConfig: `{"certResolver":"letsencrypt-dns","options":"default"}`,
	}, {
		Name: "external-route@rest", Provider: "rest", Rule: "Host(`external.example.test`)", Service: "external-service@rest",
		Entrypoints: []string{"websecure"}, TLS: true,
	}}
	services := []routeport.TraefikService{{
		Name: "secure-service@rest", Provider: "rest", Protocol: "http", Servers: []string{"http://secure:8080"},
	}, {
		Name: "external-service@rest", Provider: "rest", Protocol: "http", Servers: []string{"http://external:8080"},
	}}
	preview := compareSyncState(routes, routers, services)
	if preview.Matched || len(preview.Differences) != 1 {
		t.Fatalf("preview = %+v, want one unmanaged route difference", preview)
	}
	difference := preview.Differences[0]
	if difference.Action != "removed" || difference.RouteName != "external" || difference.Field != "route" || difference.BusinessValue != "" || difference.TraefikValue != "HTTPS Host(`external.example.test`) -> http://external:8080" {
		t.Fatalf("difference = %+v, want one readable unmanaged route difference", difference)
	}
}

func TestCompareSyncStateIncludesTCPListenPort(t *testing.T) {
	listenPort := 16379
	routes := []model.Route{{
		Name: "redis", Protocol: routeProtocolTCP, Domain: "redis.example.test", TargetAddress: "redis", TargetPort: 6379,
		Enabled: true, ListenPort: &listenPort,
	}}
	routers := []routeport.TraefikRouter{{
		Name: "redis-route@rest", Provider: "rest", Rule: "HostSNI(`*`)", Service: "redis-service@rest",
		Entrypoints: []string{"tcp16380"},
	}}
	services := []routeport.TraefikService{{
		Name: "redis-service@rest", Provider: "rest", Protocol: routeProtocolTCP, Servers: []string{"redis:6379"},
	}}

	preview := compareSyncState(routes, routers, services)
	if preview.Matched || len(preview.Differences) != 1 || preview.Differences[0].Action != "modified" || preview.Differences[0].Field != "route" || preview.Differences[0].BusinessValue != "TCP :16379 -> redis:6379" || preview.Differences[0].TraefikValue != "TCP :16380 -> redis:6379" {
		t.Fatalf("TCP listen-port preview = %+v", preview)
	}
}

func TestCompareSyncStateShowsAddedRouteWithBusinessDataOnly(t *testing.T) {
	routes := []model.Route{{
		Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://api:8080", Enabled: true,
	}}

	preview := compareSyncState(routes, nil, nil)
	if preview.Matched || len(preview.Differences) != 1 {
		t.Fatalf("preview = %+v, want one added route difference", preview)
	}
	difference := preview.Differences[0]
	if difference.Action != "added" || difference.RouteName != "api" || difference.Field != "route" || difference.BusinessValue != "HTTP Host(`api.example.test`) -> http://api:8080" || difference.TraefikValue != "" {
		t.Fatalf("difference = %+v, want business route data only", difference)
	}
}

func TestCompareSyncStateIgnoresTraefikServiceImplementationName(t *testing.T) {
	routes := []model.Route{{
		Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://api:8080", Enabled: true,
	}}
	routers := []routeport.TraefikRouter{{
		Name: "api-route@rest", Provider: "rest", Rule: "Host(`api.example.test`)", Service: "legacy-service@rest",
	}}
	services := []routeport.TraefikService{{
		Name: "legacy-service@rest", Provider: "rest", Protocol: "http", Servers: []string{"http://api:8080"},
	}}

	preview := compareSyncState(routes, routers, services)
	if !preview.Matched || len(preview.Differences) != 0 {
		t.Fatalf("preview = %+v, want matching route data", preview)
	}
	if preview.BusinessHash == preview.TraefikHash {
		t.Fatalf("hashes unexpectedly match: %+v", preview)
	}
}
