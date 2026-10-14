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
	compose, err := RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application:        model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Service:            model.Service{Code: "demo-default"},
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

func TestRenderComposeBracketsIPv6LoopbackPort(t *testing.T) {
	disabled := false
	compose, err := RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application:        model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		JoinTraefikNetwork: &disabled,
		Components: []model.EffectiveServiceComponent{{
			Name: "web", Image: "nginx:latest",
			Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "local", BindAddress: stringPointer("::1"), ListenPort: intPointer(8080)}},
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(compose, "[::1]:8080:80") {
		t.Fatalf("IPv6 binding is not bracketed:\n%s", compose)
	}
}

func TestRenderComposeJoinsSharedGatewayNetworkByDefault(t *testing.T) {
	compose, err := RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Gateway:     &model.GatewayConfig{NetworkName: "traefik"},
		Components:  []model.EffectiveServiceComponent{{Name: "api", Image: "nginx:latest"}},
	}})
	if err != nil {
		t.Fatalf("RenderCompose() error = %v", err)
	}
	if !strings.Contains(compose, "name: traefik") || !strings.Contains(compose, "external: true") {
		t.Fatalf("shared Gateway network missing from compose:\n%s", compose)
	}
}

func TestRenderComposeRejectsMissingEnvironmentGatewayNetwork(t *testing.T) {
	_, err := RenderCompose(context.Background(), RenderInput{Plan: model.EffectiveServicePlan{
		Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Components:  []model.EffectiveServiceComponent{{Name: "api", Image: "nginx:latest"}},
	}})
	if err == nil || !strings.Contains(err.Error(), "project gateway network is required") {
		t.Fatalf("RenderCompose() error = %v", err)
	}
}
