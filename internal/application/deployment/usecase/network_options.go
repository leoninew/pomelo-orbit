package deploymentsvc

import (
	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func setPlanJoinTraefikNetwork(plan *model.EffectiveServicePlan, requested *bool) {
	join := requested == nil || *requested
	if isGatewayCarrier(*plan) {
		join = true
	}
	plan.JoinTraefikNetwork = &join
}

func deploymentJoinTraefikNetwork(plan model.EffectiveServicePlan) *bool {
	join := plan.JoinsTraefikNetwork()
	return &join
}

func previewJoinTraefikNetwork(input deploymentdto.PreviewComposeInput) *bool {
	return input.JoinTraefikNetwork
}
