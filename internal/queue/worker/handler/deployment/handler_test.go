package deploymenthandler

import (
	"context"
	"strings"
	"testing"

	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

func TestHandleRejectsInvalidPayload(t *testing.T) {
	handler := NewDeployHandler(&fakeDeployer{})
	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{invalid`})
	if err == nil || !strings.Contains(err.Error(), "parse deployment task payload") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHandleRequiresApplicationAndDeploymentId(t *testing.T) {
	handler := NewDeployHandler(&fakeDeployer{})
	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "project_id, application_id and deployment_id are required") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestHandleDeploysApplication(t *testing.T) {
	deployer := &fakeDeployer{}
	handler := NewDeployHandler(deployer)

	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{"project_id":"project-1","application_id":"app-1","deployment_id":"deploy-1"}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !deployer.deployed {
		t.Fatal("expected deploy to be called")
	}
	if deployer.projectId != "project-1" || deployer.applicationId != "app-1" || deployer.deploymentId != "deploy-1" || deployer.forceRecreate {
		t.Fatalf("unexpected deploy payload: project=%s app=%s deployment=%s force_recreate=%v", deployer.projectId, deployer.applicationId, deployer.deploymentId, deployer.forceRecreate)
	}
}

func TestHandleDeploysApplicationWithForceRecreate(t *testing.T) {
	deployer := &fakeDeployer{}
	handler := NewDeployHandler(deployer)

	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{"project_id":"project-1","application_id":"app-1","deployment_id":"deploy-1","force_recreate":true}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !deployer.deployed {
		t.Fatal("expected deploy to be called")
	}
	if deployer.projectId != "project-1" || deployer.applicationId != "app-1" || deployer.deploymentId != "deploy-1" || !deployer.forceRecreate {
		t.Fatalf("unexpected deploy payload: project=%s app=%s deployment=%s force_recreate=%v", deployer.projectId, deployer.applicationId, deployer.deploymentId, deployer.forceRecreate)
	}
}

func TestHandleRestartsApplication(t *testing.T) {
	deployer := &fakeDeployer{}
	handler := NewRestartHandler(deployer)

	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{"project_id":"project-1","application_id":"app-1","deployment_id":"deploy-1"}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !deployer.restarted {
		t.Fatal("expected restart to be called")
	}
	if deployer.projectId != "project-1" || deployer.applicationId != "app-1" || deployer.deploymentId != "deploy-1" {
		t.Fatalf("unexpected ids: project=%s app=%s deployment=%s", deployer.projectId, deployer.applicationId, deployer.deploymentId)
	}
}

func TestHandleStopsApplication(t *testing.T) {
	deployer := &fakeDeployer{}
	handler := NewStopHandler(deployer)

	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{"project_id":"project-1","application_id":"app-1","deployment_id":"deploy-1","remove_volumes":true}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !deployer.stopped {
		t.Fatal("expected stop to be called")
	}
	if deployer.projectId != "project-1" || deployer.applicationId != "app-1" || deployer.deploymentId != "deploy-1" || !deployer.removeVolumes {
		t.Fatalf("unexpected stop payload: project=%s app=%s deployment=%s remove_volumes=%v", deployer.projectId, deployer.applicationId, deployer.deploymentId, deployer.removeVolumes)
	}
}

type fakeDeployer struct {
	deployed      bool
	restarted     bool
	stopped       bool
	removeVolumes bool
	forceRecreate bool
	projectId     string
	applicationId string
	deploymentId  string
}

func (d *fakeDeployer) ExecuteApplicationDeploy(_ context.Context, projectId string, applicationId string, deploymentId string, forceRecreate bool) error {
	d.deployed = true
	d.forceRecreate = forceRecreate
	d.projectId = projectId
	d.applicationId = applicationId
	d.deploymentId = deploymentId
	return nil
}

func (d *fakeDeployer) ExecuteApplicationRestart(_ context.Context, projectId string, applicationId string, deploymentId string) error {
	d.restarted = true
	d.projectId = projectId
	d.applicationId = applicationId
	d.deploymentId = deploymentId
	return nil
}

func (d *fakeDeployer) ExecuteApplicationStop(_ context.Context, projectId string, applicationId string, deploymentId string, removeVolumes bool) error {
	d.stopped = true
	d.projectId = projectId
	d.removeVolumes = removeVolumes
	d.applicationId = applicationId
	d.deploymentId = deploymentId
	return nil
}
