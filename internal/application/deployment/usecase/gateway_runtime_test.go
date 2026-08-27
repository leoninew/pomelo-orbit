package deploymentsvc

import (
	"context"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestSelectGatewayVersionForDeploymentUsesRequiredCoordinator(t *testing.T) {
	coordinator := &gatewayDeploymentCoordinatorFake{
		selected: model.Service{Id: "gateway-service", VersionId: "dns-version"},
	}
	service := Service{gatewayCoordinator: coordinator}

	selected, err := service.selectGatewayVersionForDeployment(
		context.Background(),
		model.Application{Id: "gateway-app"},
		model.Service{Id: "gateway-service", VersionId: "base-version"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if coordinator.selectCalls != 1 {
		t.Fatalf("SelectGatewayDeploymentVersion calls = %d, want 1", coordinator.selectCalls)
	}
	if selected.VersionId != "dns-version" {
		t.Fatalf("selected Version = %q, want dns-version", selected.VersionId)
	}
}

type gatewayDeploymentCoordinatorFake struct {
	selected    model.Service
	selectCalls int
}

func (f *gatewayDeploymentCoordinatorFake) GatewayForDeployment(context.Context, model.Application, model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	return nil, nil
}

func (f *gatewayDeploymentCoordinatorFake) SelectGatewayDeploymentVersion(_ context.Context, _ model.Application, _ model.Service) (model.Service, error) {
	f.selectCalls++
	return f.selected, nil
}
