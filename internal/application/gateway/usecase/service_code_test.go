package gatewaysvc

import (
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildGatewayExposureItemUsesComponentServiceHost(t *testing.T) {
	item, err := buildGatewayExposureItem(
		model.Application{Id: "app-1", Code: "ragflow"},
		model.Service{Id: "service-1", Code: "ragflow-default"},
		model.EffectiveServiceComponent{Name: "minio"},
		model.VersionComponentEndpoint{Name: "api", Protocol: "http", ContainerPort: 9000, Mode: "gateway_http"},
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
