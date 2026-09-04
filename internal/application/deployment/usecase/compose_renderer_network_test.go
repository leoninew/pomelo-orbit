package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRenderComposeSkipsTraefikNetworkWhenDisabled(t *testing.T) {
	disabled := false
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application:        model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Service:            model.Service{Code: "demo-default", InstanceKey: "default"},
		JoinTraefikNetwork: &disabled,
		Components: []model.EffectiveServiceComponent{{
			Name: "api", Image: "nginx:latest",
			Endpoints: []model.VersionComponentEndpoint{{
				Protocol: "http", ContainerPort: 80, Mode: "gateway",
			}},
		}},
	}})
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}
	if strings.Contains(compose, "traefik") {
		t.Fatalf("disabled Traefik network leaked into compose:\n%s", compose)
	}
}

func TestRenderComposeJoinsEnvironmentGatewayNetworkByDefault(t *testing.T) {
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Gateway:     &model.GatewayConfig{NetworkName: "orbit-demo-traefik"},
		Components:  []model.EffectiveServiceComponent{{Name: "api", Image: "nginx:latest"}},
	}})
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}
	if !strings.Contains(compose, "name: orbit-demo-traefik") || !strings.Contains(compose, "external: true") {
		t.Fatalf("environment Gateway network missing from compose:\n%s", compose)
	}
}

func TestRenderComposeRejectsMissingEnvironmentGatewayNetwork(t *testing.T) {
	_, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Components:  []model.EffectiveServiceComponent{{Name: "api", Image: "nginx:latest"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "project gateway network is required") {
		t.Fatalf("RenderCompose() error = %v", err)
	}
}
