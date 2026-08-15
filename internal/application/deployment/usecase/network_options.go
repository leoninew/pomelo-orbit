package deploymentsvc

import (
	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func setPlanJoinTraefikNetwork(plan *model.EffectiveServicePlan, requested *bool) {
	join := requested == nil || *requested
	if plan.Application.Kind == status.ApplicationKindGateway {
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
