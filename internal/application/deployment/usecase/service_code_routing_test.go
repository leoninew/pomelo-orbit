package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeUsesComponentServiceHosts(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "ragflow", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "ragflow-default"},
		Gateway:     &model.GatewayConfig{BaseDomain: "example.test", DefaultEntrypoint: "web"},
		Components: []model.EffectiveServiceComponent{
			{Name: "minio", Image: "minio:latest", Endpoints: []model.VersionComponentEndpoint{{Name: "api", Protocol: "http", ContainerPort: 9000, Mode: "gateway_http"}}},
			{Name: "ragflow", Image: "ragflow:latest", Endpoints: []model.VersionComponentEndpoint{{Name: "web", Protocol: "http", ContainerPort: 80, Mode: "gateway_http"}}},
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, host := range []string{"minio.ragflow-default.example.test", "ragflow.ragflow-default.example.test"} {
		if !strings.Contains(compose, "Host(`"+host+"`)") {
			t.Fatalf("compose does not route %s:\n%s", host, compose)
		}
	}
}

func TestRenderComposeUsesComponentServiceSNI(t *testing.T) {
	entrypoint := "tcp3306"
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "mysql", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "mysql-default"},
		Gateway:     &model.GatewayConfig{BaseDomain: "example.test", TLSMode: "tls"},
		Components: []model.EffectiveServiceComponent{{
			Name: "mysql", Image: "mysql:8",
			Endpoints: []model.VersionComponentEndpoint{{Name: "mysql", Protocol: "tcp", ContainerPort: 3306, Mode: "gateway_tcp", Entrypoint: &entrypoint}},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compose, "HostSNI(`mysql.mysql-default.example.test`)") {
		t.Fatalf("compose does not use component SNI:\n%s", compose)
	}
}

func TestRenderComposeRejectsDuplicateGatewayHTTPRoute(t *testing.T) {
	_, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "api", Kind: status.ApplicationKindStandard},
		Service:     model.Service{Code: "api-default"},
		Gateway:     &model.GatewayConfig{BaseDomain: "example.test", DefaultEntrypoint: "web"},
		Components: []model.EffectiveServiceComponent{{
			Name: "api", Image: "api:latest",
			Endpoints: []model.VersionComponentEndpoint{
				{Name: "http", Protocol: "http", ContainerPort: 80, Mode: "gateway_http"},
				{Name: "admin", Protocol: "http", ContainerPort: 8080, Mode: "gateway_http"},
			},
		}},
	}})
	if err == nil || !strings.Contains(err.Error(), "duplicates route") {
		t.Fatalf("render error = %v, want duplicate route error", err)
	}
}
