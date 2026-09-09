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

func TestDeploymentTargetSnapshotForLocalTargetHasNoSSHCredential(t *testing.T) {
	revision := int64(2)
	target := environmentport.Target{Environment: model.Environment{
		Id: "environment-local", ProjectId: "project-1", TargetType: model.EnvironmentTargetTypeLocal, TargetRevision: revision,
	}}
	deployment := model.Deployment{Id: "deployment-1"}
	applyDeploymentTargetSnapshot(&deployment, target, nil)
	if deployment.SSHCredentialId != nil || deployment.SSHCredentialRevision != nil {
		t.Fatalf("local deployment SSH snapshot = %#v", deployment)
	}
	if err := verifyDeploymentTargetSnapshot(deployment, deploymentdto.DeployOptionsJSON{}, target); err != nil {
		t.Fatal(err)
	}
}

func deploymentTestTarget(revision int64) environmentport.Target {
	return environmentport.Target{Environment: model.Environment{
		Id: "environment-1", ProjectId: "project-1", TargetRevision: revision,
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &model.EnvironmentSSHTarget{
			CredentialId: "credential-1", CredentialRevision: revision,
		},
	}}
}
