package gatewaysvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestManagedGatewayCodeIsProductIdentity(t *testing.T) {
	if got := ManagedGatewayCode(); got != "traefik" {
		t.Fatalf("managed gateway code = %q", got)
	}
}

func TestManagedGatewayNameIsProductIdentity(t *testing.T) {
	if got := ManagedGatewayName(); got != "Traefik" {
		t.Fatalf("managed gateway name = %q", got)
	}
}

func TestBuildGatewayExposureItemUsesComponentServiceHost(t *testing.T) {
	item, err := buildGatewayExposureItem(
		model.Application{Id: "app-1", Code: "ragflow"},
		model.Service{Id: "service-1", Code: "ragflow-default"},
		model.EffectiveServiceComponent{Name: "minio"},
		model.VersionComponentEndpoint{Protocol: "http", ContainerPort: 9000, Mode: "gateway"},
		&model.GatewayConfig{BaseDomain: "example.test"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if item.PublicHost != "minio.ragflow-default.example.test" {
		t.Fatalf("public host = %q", item.PublicHost)
	}
	if item.ClientHint != "http://minio.ragflow-default.example.test" {
		t.Fatalf("client hint = %q", item.ClientHint)
	}
}
