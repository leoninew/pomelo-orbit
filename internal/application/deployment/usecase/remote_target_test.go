package deploymentsvc

import (
	"strings"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestDeploymentTargetSnapshotRejectsChangedEnvironment(t *testing.T) {
	target := deploymentTestTarget(2)
	deployment := model.Deployment{Id: "deployment-1"}
	applyDeploymentTargetSnapshot(&deployment, target, nil)
	changed := deploymentTestTarget(3)

	err := verifyDeploymentTargetSnapshot(deployment, deploymentdto.DeployOptionsJSON{}, changed)
	if err == nil || !strings.Contains(err.Error(), "changed after deployment was queued") {
		t.Fatalf("verifyDeploymentTargetSnapshot error = %v", err)
	}
}

func TestDeploymentTargetSnapshotRequiresFormalFields(t *testing.T) {
	target := deploymentTestTarget(2)
	err := verifyDeploymentTargetSnapshot(model.Deployment{Id: "deployment-1"}, deploymentdto.DeployOptionsJSON{}, target)
	if err == nil || !strings.Contains(err.Error(), "missing environment target snapshot") {
		t.Fatalf("verifyDeploymentTargetSnapshot error = %v", err)
	}
}

func TestDeploymentTargetSnapshotPinsGatewayApplication(t *testing.T) {
	target := deploymentTestTarget(2)
	gateway := &model.GatewayConfig{ApplicationId: "gateway-app-1"}
	deployment := model.Deployment{Id: "deployment-1"}
	applyDeploymentTargetSnapshot(&deployment, target, gateway)

	if err := verifyDeploymentTargetSnapshot(deployment, deploymentdto.DeployOptionsJSON{GatewayConfig: gateway}, target); err != nil {
		t.Fatalf("verifyDeploymentTargetSnapshot returned error: %v", err)
	}
	if deployment.GatewayApplicationId == nil || *deployment.GatewayApplicationId != gateway.ApplicationId {
		t.Fatalf("gateway snapshot = %#v", deployment.GatewayApplicationId)
	}
}

func deploymentTestTarget(revision int64) environmentport.SSHTarget {
	return environmentport.SSHTarget{Environment: model.Environment{
		Id: "environment-1", ProjectId: "project-1", TargetRevision: revision,
		SSHCredentialId: "credential-1", SSHCredentialRevision: revision,
	}}
}
