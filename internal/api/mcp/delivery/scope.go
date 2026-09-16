package delivery

import (
	"context"
	"strings"
	"sync"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	codeProjectNotSelected = "project_not_selected"
	codeProjectNotReady    = "project_not_ready"
	codeProjectScopeFixed  = "project_scope_fixed"
)

type projectScope struct {
	mu       sync.Mutex
	id       string
	fixed    bool
	verified bool
}

func (c *core) currentProjectId() (string, error) {
	c.scope.mu.Lock()
	defer c.scope.mu.Unlock()
	if strings.TrimSpace(c.scope.id) == "" {
		return "", apperror.NewWithCode(apperror.KindValidation, codeProjectNotSelected, "No Project is selected for this MCP connection")
	}
	return c.scope.id, nil
}

func (c *core) requireEnvironment(ctx context.Context) (string, environmentdto.View, error) {
	projectId, err := c.currentProjectId()
	if err != nil {
		return "", environmentdto.View{}, err
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectId)
	if err != nil {
		if apperror.IsKind(err, apperror.KindNotFound) {
			return "", environmentdto.View{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
		}
		return "", environmentdto.View{}, err
	}
	return projectId, environment, nil
}

func (c *core) requireReadyEnvironment(ctx context.Context) (string, environmentdto.View, error) {
	projectId, environment, err := c.requireEnvironment(ctx)
	if err != nil {
		return "", environmentdto.View{}, err
	}
	if !environmentProbeSucceeded(environment) {
		return "", environmentdto.View{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
	}
	return projectId, environment, nil
}

func (c *core) requireReadyGateway(ctx context.Context) (string, gatewaydto.GatewayView, error) {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return "", gatewaydto.GatewayView{}, err
	}
	gateway, err := c.readyGateway(ctx, projectId)
	if err != nil {
		return "", gatewaydto.GatewayView{}, err
	}
	return projectId, gateway, nil
}

func (c *core) readyGateway(ctx context.Context, projectId string) (gatewaydto.GatewayView, error) {
	gateways, err := c.deps.Gateway.ListGateways(ctx, c.deps.ActorUserId, projectId, 1, 1, "")
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if gateways.Total == 0 || len(gateways.Items) == 0 || gateways.Items[0].DefaultService == nil {
		return gatewaydto.GatewayView{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project gateway is not ready")
	}
	return gateways.Items[0], nil
}

func (c *core) bindReadyProject(ctx context.Context, projectId string) (map[string]any, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	c.scope.mu.Lock()
	fixed := c.scope.fixed
	current := c.scope.id
	c.scope.mu.Unlock()
	if fixed && current != "" && current != projectId {
		return nil, apperror.NewWithCode(apperror.KindForbidden, codeProjectScopeFixed, "Project scope cannot be changed for this MCP connection")
	}
	projects, err := c.deps.Project.ListByMember(ctx, c.deps.ActorUserId)
	if err != nil {
		return nil, err
	}
	var selected model.Project
	for _, project := range projects {
		if project.Id == projectId {
			selected = project
			break
		}
	}
	if selected.Id == "" {
		return nil, apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectId)
	if err != nil {
		if apperror.IsKind(err, apperror.KindNotFound) {
			return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
		}
		return nil, err
	}
	if !environmentProbeSucceeded(environment) {
		return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
	}
	gateway, err := c.readyGateway(ctx, projectId)
	if err != nil {
		return nil, err
	}
	c.scope.mu.Lock()
	c.scope.id = projectId
	c.scope.verified = true
	c.scope.mu.Unlock()
	return currentProjectOutput(selected, environment, gateway), nil
}

func (c *core) currentProjectSummary(ctx context.Context) (map[string]any, error) {
	projectId, err := c.currentProjectId()
	if err != nil {
		return nil, err
	}
	projects, err := c.deps.Project.ListByMember(ctx, c.deps.ActorUserId)
	if err != nil {
		return nil, err
	}
	var selected model.Project
	for _, project := range projects {
		if project.Id == projectId {
			selected = project
			break
		}
	}
	if selected.Id == "" {
		return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotSelected, "No Project is selected for this MCP connection")
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectId)
	if err != nil {
		return nil, err
	}
	gateway, err := c.readyGateway(ctx, projectId)
	if err != nil {
		return nil, err
	}
	return currentProjectOutput(selected, environment, gateway), nil
}

func (c *core) applicationInScope(ctx context.Context, applicationId string) (model.Application, error) {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Application{}, err
	}
	return c.deps.Application.ApplicationForUser(ctx, c.deps.ActorUserId, projectId, applicationId)
}

func (c *core) versionInScope(ctx context.Context, versionId string) error {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return err
	}
	_, err = c.deps.Application.VersionForUser(ctx, c.deps.ActorUserId, projectId, versionId)
	return err
}

func (c *core) serviceInScope(ctx context.Context, serviceId string) error {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return err
	}
	_, err = c.deps.Service.GetService(ctx, c.deps.ActorUserId, projectId, serviceId)
	return err
}

func (c *core) gatewayInScope(ctx context.Context, gatewayId string) (gatewaydto.GatewayView, error) {
	projectId, _, err := c.requireReadyGateway(ctx)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	gateway, err := c.deps.Gateway.GatewayForUser(ctx, c.deps.ActorUserId, projectId, gatewayId)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	return gateway, nil
}

func (c *core) routeInScope(ctx context.Context, routeId string) (model.Route, error) {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Route{}, err
	}
	route, err := c.deps.Route.RouteForUser(ctx, c.deps.ActorUserId, projectId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	return route, nil
}

func (c *core) deploymentInScope(ctx context.Context, deploymentId string) (model.Deployment, error) {
	projectId, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Deployment{}, err
	}
	return c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, projectId, deploymentId)
}

func environmentProbeSucceeded(view environmentdto.View) bool {
	if view.LastProbeRevision == nil || *view.LastProbeRevision != view.TargetRevision || view.LastProbeStatus == nil || *view.LastProbeStatus != model.EnvironmentProbeStatusSucceeded {
		return false
	}
	if view.TargetType == model.EnvironmentTargetTypeLocal {
		return true
	}
	if view.SSH == nil {
		return false
	}
	fingerprint := strings.TrimSpace(view.SSH.HostKeyFingerprint)
	return strings.HasPrefix(fingerprint, "SHA256:") && len(fingerprint) > len("SHA256:")
}

func currentProjectOutput(project model.Project, environment environmentdto.View, gateway gatewaydto.GatewayView) map[string]any {
	output := map[string]any{
		"project":     projectOutput(project),
		"environment": environmentOutput(environment),
		"gateway":     gatewayOutput(gateway),
	}
	if gateway.DefaultService != nil {
		output["default_service"] = gatewayServiceOutput(*gateway.DefaultService)
	}
	return output
}
