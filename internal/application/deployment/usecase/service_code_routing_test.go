package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRenderComposeUsesComponentServiceHosts(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "ragflow", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "ragflow-service"},
		Gateway:     &model.GatewayConfig{BaseDomain: "example.test", DefaultEntrypoint: "web", NetworkName: "orbit-routing-test-traefik"},
		Components: []model.EffectiveServiceComponent{
			{Name: "minio", Image: "minio:latest", Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 9000, Mode: "gateway"}}},
			{Name: "ragflow", Image: "ragflow:latest", Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "gateway"}}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"minio.ragflow-service.example.test", "ragflow.ragflow-service.example.test"} {
		if !strings.Contains(compose, "Host(`"+host+"`)") {
			t.Fatalf("compose does not route %s:\n%s", host, compose)
		}
	}
}

func TestRenderComposeDoesNotPublishTCPDeclaration(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "mysql", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "mysql-default"},
		Gateway:     &model.GatewayConfig{NetworkName: "orbit-routing-test-traefik"},
		Components: []model.EffectiveServiceComponent{{
			Name: "mysql", Image: "mysql:8",
			Endpoints: []model.VersionComponentEndpoint{{Protocol: "tcp", ContainerPort: 3306, Mode: "internal"}},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compose, "traefik.tcp.routers") || strings.Contains(compose, "3306:3306") {
		t.Fatalf("compose must not publish TCP declarations directly:\n%s", compose)
	}
	if !strings.Contains(compose, "- mysql-mysql") {
		t.Fatalf("TCP route targets must join the Traefik network with their stable alias:\n%s", compose)
	}
}

func TestRenderComposeRejectsDuplicateGatewayHTTPRoute(t *testing.T) {
	_, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "api", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "api-default"},
		Gateway:     &model.GatewayConfig{BaseDomain: "example.test", DefaultEntrypoint: "web", NetworkName: "orbit-routing-test-traefik"},
		Components: []model.EffectiveServiceComponent{{
			Name: "api", Image: "api:latest",
			Endpoints: []model.VersionComponentEndpoint{
				{Protocol: "http", ContainerPort: 80, Mode: "gateway"},
				{Protocol: "http", ContainerPort: 8080, Mode: "gateway"},
			},
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "duplicates route") {
		t.Fatalf("render error = %v, want duplicate route error", err)
	}
}
