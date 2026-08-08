package model

import "testing"

func TestDeriveServiceComponentHost(t *testing.T) {
	host, err := DeriveServiceComponentHost(
		&GatewayConfig{BaseDomain: "example.test"},
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
		{name: "invalid service code", gateway: &GatewayConfig{BaseDomain: "example.test"}, service: Service{Code: "ragflow_default"}, component: "minio"},
		{name: "invalid component", gateway: &GatewayConfig{BaseDomain: "example.test"}, service: Service{Code: "ragflow-default"}, component: "minio_console"},
		{name: "invalid base domain", gateway: &GatewayConfig{BaseDomain: "example_test"}, service: Service{Code: "ragflow-default"}, component: "minio"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DeriveServiceComponentHost(test.gateway, test.service, test.component); err == nil {
				t.Fatal("expected invalid host error")
			}
		})
	}
}
