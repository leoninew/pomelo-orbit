package deploymentsvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestDeploymentPullPolicyIsSavedAndExecuted(t *testing.T) {
	for _, gateway := range []bool{false, true} {
		for _, policy := range []string{"", "missing", "always", "never"} {
			name := "service/" + policy
			if gateway {
				name = "gateway/" + policy
			}
			t.Run(name, func(t *testing.T) {
				executor, store, runtime, _ := independentDeploymentTestService(t, gateway)
				description := &recordingLifecycleCommands{}
				service := Service{commands: description, commandStore: store, dispatcher: directoryDispatcher{}, targetResolver: executor.targetResolver, transactionRunner: testDeploymentTransaction{}}
				config := &model.GatewayConfig{ApplicationId: "gateway-1", NetworkName: "traefik"}
				if gateway {
					config = testGatewayEnrichmentPlan("", "", "").Gateway
					config.ApplicationId = store.application.Id
				}
				service.gatewayCoordinator = &gatewayDeploymentCoordinatorFake{gateway: config}
				store.components[0].PullPolicy = "always"
				restartPolicy := "unless-stopped"
				store.components[0].RestartPolicy = &restartPolicy
				input := deploymentdto.DeployServiceInput{
					DeploymentDirectory: filepath.ToSlash(t.TempDir()), EnvironmentTargetRevision: 1,
				}
				want := policy
				if policy == "" {
					want = "missing"
				} else {
					input.PullPolicy = &policy
				}
				result, err := service.DeployService(context.Background(), "user-1", "project-1", store.service.Id, input)
				if err != nil {
					t.Fatal(err)
				}
				options, err := parseDeployOptions(store.deployment.OptionsJSON)
				if err != nil || options.PullPolicy != want {
					t.Fatalf("saved pull policy = %q, want %q; error = %v", options.PullPolicy, want, err)
				}
				if store.deployment.CommandText != "deployment command" {
					t.Fatalf("saved command = %s", store.deployment.CommandText)
				}
				if err := executor.ExecuteApplicationDeploy(context.Background(), "project-1", store.application.Id, result.DeploymentId, false); err != nil {
					t.Fatal(err)
				}
				if !runtime.deployed || runtime.settings.PullPolicy != want || runtime.settings.ForceRecreate != options.ForceRecreate || runtime.scope.Project != store.service.Code {
					t.Fatalf("executed settings = %+v, saved options = %+v", runtime.settings, options)
				}
				if description.settings != runtime.settings || description.project != runtime.scope.Project {
					t.Fatalf("described=%+v executed=%+v", description, runtime.settings)
				}
				if len(runtime.staged) != 1 || !strings.Contains(runtime.staged[0].Compose, "restart: unless-stopped") {
					t.Fatal("deployment did not preserve the component restart policy")
				}
			})
		}
	}
}

type recordingLifecycleCommands struct {
	lifecycleCommandsFake
	project  string
	settings deploymentport.DeploySettings
}

func (r *recordingLifecycleCommands) DeployDescription(project string, settings deploymentport.DeploySettings) string {
	r.project, r.settings = project, settings
	return "deployment command"
}

func TestDeploymentRejectsInvalidPullPolicyBeforeChangingConfiguration(t *testing.T) {
	executor, store, _, _ := independentDeploymentTestService(t, false)
	service := Service{commands: lifecycleCommandsFake{}, commandStore: store, dispatcher: directoryDispatcher{}, targetResolver: executor.targetResolver, transactionRunner: testDeploymentTransaction{}}
	policy := "unless-stopped"
	_, err := service.DeployService(context.Background(), "user-1", "project-1", store.service.Id, deploymentdto.DeployServiceInput{
		DeploymentDirectory: "/new/api", EnvironmentTargetRevision: 1, PullPolicy: &policy,
	})
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if store.service.DeploymentDirectory != "/custom/demo" || store.deployment.Id != "deployment-1" {
		t.Fatal("invalid pull policy changed the service or created a deployment")
	}
}
