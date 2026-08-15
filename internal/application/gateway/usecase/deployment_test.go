package gatewaysvc

import (
	"context"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestGatewayForDeploymentDoesNotRequireGatewayForInternalTCPEndpoint(t *testing.T) {
	plan := model.EffectiveServicePlan{
		Application: model.Application{Kind: status.ApplicationKindStandard},
		Components: []model.EffectiveServiceComponent{{
			Name: "redis",
			Endpoints: []model.VersionComponentEndpoint{{
				Protocol: "tcp", ContainerPort: 6379, Mode: "internal",
			}},
		}},
	}
	if hasGatewayEndpoint(plan) {
		t.Fatal("internal TCP endpoint must not be treated as a Gateway endpoint")
	}
}

func TestGatewayForDeploymentSkipsGatewayConfigWhenTraefikNetworkDisabled(t *testing.T) {
	disabled := false
	plan := model.EffectiveServicePlan{
		Application:        model.Application{Kind: status.ApplicationKindStandard},
		JoinTraefikNetwork: &disabled,
		Components: []model.EffectiveServiceComponent{{
			Name: "api",
			Endpoints: []model.VersionComponentEndpoint{{
				Protocol: "http", ContainerPort: 80, Mode: "gateway",
			}},
		}},
	}

	gateway, err := (Service{}).GatewayForDeployment(context.Background(), plan.Application, plan)
	if err != nil {
		t.Fatalf("GatewayForDeployment() error = %v", err)
	}
	if gateway != nil {
		t.Fatalf("GatewayForDeployment() = %#v, want nil", gateway)
	}
}
