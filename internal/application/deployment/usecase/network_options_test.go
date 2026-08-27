package deploymentsvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestSetPlanJoinTraefikNetworkForcesGatewayCarrier(t *testing.T) {
	disabled := false
	plan := model.EffectiveServicePlan{
		Application:        model.Application{Id: "gateway-app"},
		Gateway:            &model.GatewayConfig{ApplicationId: "gateway-app"},
		JoinTraefikNetwork: &disabled,
	}

	setPlanJoinTraefikNetwork(&plan, &disabled)

	if !plan.JoinsTraefikNetwork() {
		t.Fatal("Gateway carrier must join the Traefik network")
	}
}
