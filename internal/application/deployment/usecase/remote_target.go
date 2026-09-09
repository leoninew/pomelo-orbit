package deploymentsvc

import (
	"context"
	"fmt"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) resolveProjectTarget(ctx context.Context, app model.Application) (environmentport.Target, error) {
	if app.ProjectId == nil || strings.TrimSpace(*app.ProjectId) == "" {
		return environmentport.Target{}, fmt.Errorf("application %s is missing project scope", app.Id)
	}
	if s.targetResolver == nil {
		return environmentport.Target{}, fmt.Errorf("deployment target resolver is not configured")
	}
	return s.targetResolver.ResolveProjectTarget(ctx, *app.ProjectId)
}

func applyDeploymentTargetSnapshot(deployment *model.Deployment, target environmentport.Target, gateway *model.GatewayConfig) {
	environmentID := target.Environment.Id
	targetType := target.Environment.TargetType
	targetRevision := target.Environment.TargetRevision
	deployment.EnvironmentId = &environmentID
	deployment.EnvironmentTargetType = &targetType
	deployment.EnvironmentTargetRevision = &targetRevision
	deployment.SSHCredentialId = nil
	deployment.SSHCredentialRevision = nil
	if target.Environment.IsSSH() {
		credentialID := target.Environment.SSH.CredentialId
		credentialRevision := target.Environment.SSH.CredentialRevision
		deployment.SSHCredentialId = &credentialID
		deployment.SSHCredentialRevision = &credentialRevision
	}
	deployment.GatewayApplicationId = nil
	if gateway != nil {
		gatewayApplicationID := gateway.ApplicationId
		deployment.GatewayApplicationId = &gatewayApplicationID
	}
}

func verifyDeploymentTargetSnapshot(deployment model.Deployment, options deploymentdto.DeployOptionsJSON, target environmentport.Target) error {
	if deployment.EnvironmentId == nil || strings.TrimSpace(*deployment.EnvironmentId) == "" ||
		deployment.EnvironmentTargetType == nil || strings.TrimSpace(*deployment.EnvironmentTargetType) == "" ||
		deployment.EnvironmentTargetRevision == nil || *deployment.EnvironmentTargetRevision < 1 ||
		(*deployment.EnvironmentTargetType != model.EnvironmentTargetTypeLocal && *deployment.EnvironmentTargetType != model.EnvironmentTargetTypeSSH) {
		return fmt.Errorf("deployment %s is missing environment target snapshot", deployment.Id)
	}
	environment := target.Environment
	if *deployment.EnvironmentId != environment.Id ||
		*deployment.EnvironmentTargetType != environment.TargetType ||
		*deployment.EnvironmentTargetRevision != environment.TargetRevision ||
		(environment.IsSSH() && (deployment.SSHCredentialId == nil || strings.TrimSpace(*deployment.SSHCredentialId) == "" || deployment.SSHCredentialRevision == nil || *deployment.SSHCredentialRevision < 1 || *deployment.SSHCredentialId != environment.SSH.CredentialId || *deployment.SSHCredentialRevision != environment.SSH.CredentialRevision)) ||
		(environment.IsLocal() && (deployment.SSHCredentialId != nil || deployment.SSHCredentialRevision != nil)) {
		return fmt.Errorf("project environment changed after deployment was queued; deploy again")
	}
	if deployment.GatewayApplicationId != nil && strings.TrimSpace(*deployment.GatewayApplicationId) != "" &&
		(options.GatewayConfig == nil || options.GatewayConfig.ApplicationId != *deployment.GatewayApplicationId) {
		return fmt.Errorf("deployment %s has an inconsistent Gateway snapshot", deployment.Id)
	}
	return nil
}
