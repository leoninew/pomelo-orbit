package deploymentsvc

import (
	"context"
	"fmt"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) resolveProjectTarget(ctx context.Context, app model.Application) (environmentport.SSHTarget, error) {
	if app.ProjectId == nil || strings.TrimSpace(*app.ProjectId) == "" {
		return environmentport.SSHTarget{}, fmt.Errorf("application %s is missing project scope", app.Id)
	}
	if s.targetResolver == nil {
		return environmentport.SSHTarget{}, fmt.Errorf("deployment target resolver is not configured")
	}
	return s.targetResolver.ResolveProjectTarget(ctx, *app.ProjectId)
}

func applyDeploymentTargetSnapshot(deployment *model.Deployment, target environmentport.SSHTarget, gateway *model.GatewayConfig) {
	environmentID := target.Environment.Id
	targetRevision := target.Environment.TargetRevision
	credentialID := target.Environment.SSHCredentialId
	credentialRevision := target.Environment.SSHCredentialRevision
	deployment.EnvironmentId = &environmentID
	deployment.EnvironmentTargetRevision = &targetRevision
	deployment.SSHCredentialId = &credentialID
	deployment.SSHCredentialRevision = &credentialRevision
	deployment.GatewayApplicationId = nil
	if gateway != nil {
		gatewayApplicationID := gateway.ApplicationId
		deployment.GatewayApplicationId = &gatewayApplicationID
	}
}

func verifyDeploymentTargetSnapshot(deployment model.Deployment, options deploymentdto.DeployOptionsJSON, target environmentport.SSHTarget) error {
	if deployment.EnvironmentId == nil || strings.TrimSpace(*deployment.EnvironmentId) == "" ||
		deployment.EnvironmentTargetRevision == nil || *deployment.EnvironmentTargetRevision < 1 ||
		deployment.SSHCredentialId == nil || strings.TrimSpace(*deployment.SSHCredentialId) == "" ||
		deployment.SSHCredentialRevision == nil || *deployment.SSHCredentialRevision < 1 {
		return fmt.Errorf("deployment %s is missing environment target snapshot", deployment.Id)
	}
	environment := target.Environment
	if *deployment.EnvironmentId != environment.Id ||
		*deployment.EnvironmentTargetRevision != environment.TargetRevision ||
		*deployment.SSHCredentialId != environment.SSHCredentialId ||
		*deployment.SSHCredentialRevision != environment.SSHCredentialRevision {
		return fmt.Errorf("project environment changed after deployment was queued; deploy again")
	}
	if deployment.GatewayApplicationId != nil && strings.TrimSpace(*deployment.GatewayApplicationId) != "" &&
		(options.GatewayConfig == nil || options.GatewayConfig.ApplicationId != *deployment.GatewayApplicationId) {
		return fmt.Errorf("deployment %s has an inconsistent Gateway snapshot", deployment.Id)
	}
	return nil
}
