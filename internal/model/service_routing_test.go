package model

import "testing"

func TestDeriveServiceComponentHost(t *testing.T) {
	host, err := DeriveServiceComponentHost(
		&GatewayConfig{InternalDomain: "example.test"},
		Service{Code: "ragflow-default"},
		"minio",
	)
	if err != nil {
		t.Fatal(err)
	}
	if host != "minio.ragflow-default.example.test" {
		t.Fatalf("host = %q", host)
	}
}

func TestDeriveServiceComponentHostRejectsInvalidLabels(t *testing.T) {
	for _, test := range []struct {
		name      string
		gateway   *GatewayConfig
		service   Service
		component string
	}{
		{name: "missing gateway", service: Service{Code: "ragflow-default"}, component: "minio"},
		{name: "invalid service code", gateway: &GatewayConfig{InternalDomain: "example.test"}, service: Service{Code: "ragflow_default"}, component: "minio"},
		{name: "invalid component", gateway: &GatewayConfig{InternalDomain: "example.test"}, service: Service{Code: "ragflow-default"}, component: "minio_console"},
		{name: "invalid base domain", gateway: &GatewayConfig{InternalDomain: "example_test"}, service: Service{Code: "ragflow-default"}, component: "minio"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DeriveServiceComponentHost(test.gateway, test.service, test.component); err == nil {
				t.Fatal("expected invalid host error")
			}
		})
	}
}

func TestRuntimeContainerName(t *testing.T) {
	if got, want := RuntimeContainerName("RAGFlow_Prod", "minio_api"), "ragflow-prod-minio-api"; got != want {
		t.Fatalf("container name = %q, want %q", got, want)
	}
}

func TestGatewayNetworkName(t *testing.T) {
	if got, want := GatewayNetworkName(), "traefik"; got != want {
		t.Fatalf("GatewayNetworkName() = %q, want %q", got, want)
	}
}

func TestIsExternalDomainSuffix(t *testing.T) {
	for _, domain := range []string{"example.com", "sub.example.com", "123.sub.example.com", "example.xn--p1ai"} {
		if !IsExternalDomainSuffix(domain) {
			t.Errorf("expected valid external domain suffix %q", domain)
		}
	}
	for _, domain := range []string{"example", "example.123", "1.2.3.4", "bad_name.example.com", "-sub.example.com"} {
		if IsExternalDomainSuffix(domain) {
			t.Errorf("expected invalid external domain suffix %q", domain)
		}
	}
}
