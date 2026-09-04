package gatewaysvc

import (
	"context"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
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

func TestGatewayForDeploymentFindsCarrierEvenWhenNetworkWasRequestedDisabled(t *testing.T) {
	disabled := false
	projectID := "project-1"
	app := model.Application{Id: "gateway-app", ProjectId: &projectID, Kind: status.ApplicationKindStandard}
	plan := model.EffectiveServicePlan{
		Application: app, JoinTraefikNetwork: &disabled,
	}
	service := Service{config: gatewayConfigStore{cfg: model.GatewayConfig{ApplicationId: app.Id, NetworkName: "orbit-project-1-traefik"}}}

	config, err := service.GatewayForDeployment(context.Background(), app, plan)
	if err != nil {
		t.Fatalf("GatewayForDeployment() error = %v", err)
	}
	if config == nil || config.ApplicationId != app.Id || config.NetworkName != "orbit-project-1-traefik" {
		t.Fatalf("GatewayForDeployment() = %#v, want GatewayConfig for %q", config, app.Id)
	}
}

type gatewayConfigStore struct {
	cfg model.GatewayConfig
}

func (s gatewayConfigStore) GatewayConfig(_ context.Context, applicationID string) (model.GatewayConfig, error) {
	if applicationID != s.cfg.ApplicationId {
		return model.GatewayConfig{}, repository.ErrNotFound
	}
	return s.cfg, nil
}

func (s gatewayConfigStore) GatewayConfigByProject(_ context.Context, projectID string) (model.GatewayConfig, error) {
	if projectID == "" {
		return model.GatewayConfig{}, repository.ErrNotFound
	}
	return s.cfg, nil
}

func (gatewayConfigStore) ListGatewayApplications(context.Context, string) ([]model.Application, error) {
	return nil, nil
}

func (gatewayConfigStore) UpsertGatewayConfig(context.Context, model.GatewayConfig) error {
	return nil
}

func (gatewayConfigStore) ReplaceGatewayVersionBindings(context.Context, string, []model.GatewayVersionBinding) error {
	return nil
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
