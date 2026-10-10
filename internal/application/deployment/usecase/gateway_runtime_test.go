package deploymentsvc

import (
	"context"
	"io"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestSelectGatewayVersionForDeploymentUsesRequiredCoordinator(t *testing.T) {
	coordinator := &gatewayDeploymentCoordinatorFake{
		selected: model.Service{Id: "gateway-service", VersionId: "dns-version"},
	}
	service := Service{gatewayCoordinator: coordinator}

	selected, err := service.selectGatewayVersionForDeployment(
		context.Background(),
		"project-1",
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

func TestGatewayDeploymentOverwritesStaticConfigAndRecreatesContainer(t *testing.T) {
	for _, gateway := range []bool{true, false} {
		t.Run(map[bool]string{true: "gateway", false: "ordinary-service"}[gateway], func(t *testing.T) {
			workspace := &gatewayDeploymentRuntimeFake{workspaceFake: testWorkspace(t.TempDir())}
			service := Service{runtime: workspace, logStore: workspace}
			plan := testGatewayEnrichmentPlan("", "", "")
			plan.Service.Code = "traefik-default"
			plan.Service.DeploymentDirectory = "/custom/traefik"
			deploymentId := "deployment-1"
			plan.Service.CurrentDeploymentId = &deploymentId
			service.executionStore = &independentDeploymentStore{runtimeQueryStore: &runtimeQueryStore{service: plan.Service}, deployment: model.Deployment{Id: deploymentId, ServiceId: &plan.Service.Id, Status: status.WorkStatusRunning}}
			plan.Components[0].Image = "traefik:3.6"
			plan.Components[0].Mounts = plan.Components[0].Mounts[:1]
			plan.Components[0].Mounts[0].Content = "providers:\n  rest:\n    insecure: true\n"
			plan.Components[0].Mounts[0].IgnoreIfExists = true
			if !gateway {
				plan.Gateway = nil
				plan.JoinTraefikNetwork = new(bool)
			}
			if err := service.renderAndDeployWithOptions(context.Background(), environmentport.Target{}, plan, "deployment-1", false); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(workspace.command, "--force-recreate") != gateway {
				t.Fatalf("deployment command = %s", workspace.command)
			}
			if len(workspace.staged) != 1 || len(workspace.staged[0].Files) != 1 {
				t.Fatalf("staged workspace = %+v", workspace.staged)
			}
			file := workspace.staged[0].Files[0]
			if gateway && (file.IgnoreIfExists || strings.Contains(string(file.Content), "rest:") || !strings.Contains(string(file.Content), "watch: true")) {
				t.Fatalf("staged Gateway config = %s, ignore_if_exists=%v", file.Content, file.IgnoreIfExists)
			}
			if !gateway && (!file.IgnoreIfExists || !strings.Contains(string(file.Content), "rest:")) {
				t.Fatal("ordinary Service configuration was overwritten")
			}
		})
	}
}

type gatewayDeploymentRuntimeFake struct {
	*workspaceFake
	command string
}

func (f *gatewayDeploymentRuntimeFake) Run(_ context.Context, _ environmentport.Target, _ deploymentport.ServiceLocation, _ io.Writer, command string, args ...string) error {
	f.command = command + " " + strings.Join(args, " ")
	return nil
}

type gatewayDeploymentCoordinatorFake struct {
	selected    model.Service
	selectCalls int
	gateway     *model.GatewayConfig
}

func (f *gatewayDeploymentCoordinatorFake) GatewayForDeployment(context.Context, string, model.Application, model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	return f.gateway, nil
}

func (f *gatewayDeploymentCoordinatorFake) SelectGatewayDeploymentVersion(_ context.Context, _ string, _ model.Application, service model.Service) (model.Service, error) {
	f.selectCalls++
	if f.selected.Id == "" {
		return service, nil
	}
	return f.selected, nil
}
