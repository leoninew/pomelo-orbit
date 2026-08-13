package gatewaysvc

import (
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
