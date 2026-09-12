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
	codeProjectNotSelected   = "project_not_selected"
	codeProjectNotReady      = "project_not_ready"
	codeProjectScopeMismatch = "project_scope_mismatch"
	codeProjectScopeFixed    = "project_scope_fixed"
)

type projectScope struct {
	mu       sync.Mutex
	id       string
	fixed    bool
	verified bool
}

func (c *core) currentProjectID() (string, error) {
	c.scope.mu.Lock()
	defer c.scope.mu.Unlock()
	if strings.TrimSpace(c.scope.id) == "" {
		return "", apperror.NewWithCode(apperror.KindValidation, codeProjectNotSelected, "No Project is selected for this MCP connection")
	}
	return c.scope.id, nil
}

func (c *core) requireActiveEnvironment(ctx context.Context) (string, environmentdto.View, error) {
	projectID, err := c.currentProjectID()
	if err != nil {
		return "", environmentdto.View{}, err
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectID)
	if err != nil {
		if apperror.IsKind(err, apperror.KindNotFound) {
			return "", environmentdto.View{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
		}
		return "", environmentdto.View{}, err
	}
	if environment.State != model.EnvironmentStateActive {
		return "", environmentdto.View{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
	}
	return projectID, environment, nil
}

func (c *core) requireReadyEnvironment(ctx context.Context) (string, environmentdto.View, error) {
	projectID, environment, err := c.requireActiveEnvironment(ctx)
	if err != nil {
		return "", environmentdto.View{}, err
	}
	if !environmentProbeSucceeded(environment) {
		return "", environmentdto.View{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
	}
	return projectID, environment, nil
}

func (c *core) requireReadyGateway(ctx context.Context) (string, gatewaydto.GatewayView, error) {
	projectID, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return "", gatewaydto.GatewayView{}, err
	}
	gateway, err := c.readyGateway(ctx, projectID)
	if err != nil {
		return "", gatewaydto.GatewayView{}, err
	}
	return projectID, gateway, nil
}

func (c *core) readyGateway(ctx context.Context, projectID string) (gatewaydto.GatewayView, error) {
	gateways, err := c.deps.Gateway.ListGateways(ctx, c.deps.ActorUserId, projectID, 1, 1, "")
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if gateways.Total == 0 || len(gateways.Items) == 0 || gateways.Items[0].DefaultService == nil {
		return gatewaydto.GatewayView{}, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project gateway is not ready")
	}
	return gateways.Items[0], nil
}

func (c *core) bindReadyProject(ctx context.Context, projectID string) (map[string]any, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	c.scope.mu.Lock()
	fixed := c.scope.fixed
	current := c.scope.id
	c.scope.mu.Unlock()
	if fixed && current != "" && current != projectID {
		return nil, apperror.NewWithCode(apperror.KindForbidden, codeProjectScopeFixed, "Project scope cannot be changed for this MCP connection")
	}
	projects, err := c.deps.Project.ListByMember(ctx, c.deps.ActorUserId)
	if err != nil {
		return nil, err
	}
	var selected model.Project
	for _, project := range projects {
		if project.Id == projectID {
			selected = project
			break
		}
	}
	if selected.Id == "" {
		return nil, apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectID)
	if err != nil {
		if apperror.IsKind(err, apperror.KindNotFound) {
			return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
		}
		return nil, err
	}
	if environment.State != model.EnvironmentStateActive || !environmentProbeSucceeded(environment) {
		return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotReady, "Project environment is not ready")
	}
	gateway, err := c.readyGateway(ctx, projectID)
	if err != nil {
		return nil, err
	}
	c.scope.mu.Lock()
	c.scope.id = projectID
	c.scope.verified = true
	c.scope.mu.Unlock()
	return currentProjectOutput(selected, environment, gateway), nil
}

func (c *core) currentProjectSummary(ctx context.Context) (map[string]any, error) {
	projectID, err := c.currentProjectID()
	if err != nil {
		return nil, err
	}
	projects, err := c.deps.Project.ListByMember(ctx, c.deps.ActorUserId)
	if err != nil {
		return nil, err
	}
	var selected model.Project
	for _, project := range projects {
		if project.Id == projectID {
			selected = project
			break
		}
	}
	if selected.Id == "" {
		return nil, apperror.NewWithCode(apperror.KindValidation, codeProjectNotSelected, "No Project is selected for this MCP connection")
	}
	environment, err := c.deps.Environment.EnvironmentForUser(ctx, c.deps.ActorUserId, projectID)
	if err != nil {
		return nil, err
	}
	gateway, err := c.readyGateway(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return currentProjectOutput(selected, environment, gateway), nil
}

func (c *core) applicationInScope(ctx context.Context, applicationID string) (model.Application, error) {
	projectID, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Application{}, err
	}
	application, err := c.deps.Application.ApplicationForUser(ctx, c.deps.ActorUserId, applicationID)
	if err != nil {
		return model.Application{}, err
	}
	if err := assertResourceProject(projectID, derefString(application.ProjectId)); err != nil {
		return model.Application{}, err
	}
	return application, nil
}

func (c *core) versionInScope(ctx context.Context, versionID string) error {
	version, err := c.deps.Application.VersionForUser(ctx, c.deps.ActorUserId, versionID)
	if err != nil {
		return err
	}
	_, err = c.applicationInScope(ctx, version.Version.ApplicationId)
	return err
}

func (c *core) serviceInScope(ctx context.Context, serviceID string) error {
	service, err := c.deps.Service.GetService(ctx, c.deps.ActorUserId, serviceID)
	if err != nil {
		return err
	}
	_, err = c.applicationInScope(ctx, service.Service.ApplicationId)
	return err
}

func (c *core) gatewayInScope(ctx context.Context, gatewayID string) (gatewaydto.GatewayView, error) {
	projectID, _, err := c.requireReadyGateway(ctx)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	gateway, err := c.deps.Gateway.GatewayForUser(ctx, c.deps.ActorUserId, gatewayID)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if err := assertResourceProject(projectID, derefString(gateway.Application.ProjectId)); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	return gateway, nil
}

func (c *core) routeInScope(ctx context.Context, routeID string) (model.Route, error) {
	projectID, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Route{}, err
	}
	route, err := c.deps.Route.RouteForUser(ctx, c.deps.ActorUserId, routeID)
	if err != nil {
		return model.Route{}, err
	}
	if err := assertResourceProject(projectID, derefString(route.ProjectId)); err != nil {
		return model.Route{}, err
	}
	return route, nil
}

func (c *core) deploymentInScope(ctx context.Context, deploymentID string) (model.Deployment, error) {
	projectID, _, err := c.requireReadyEnvironment(ctx)
	if err != nil {
		return model.Deployment{}, err
	}
	deployment, err := c.deps.Deployment.DeploymentForUser(ctx, c.deps.ActorUserId, deploymentID)
	if err != nil {
		return model.Deployment{}, err
	}
	if err := assertResourceProject(projectID, derefString(deployment.ProjectId)); err != nil {
		return model.Deployment{}, err
	}
	return deployment, nil
}

func assertResourceProject(scopeID, resourceProjectID string) error {
	if strings.TrimSpace(resourceProjectID) == "" || resourceProjectID != scopeID {
		return apperror.NewWithCode(apperror.KindForbidden, codeProjectScopeMismatch, "Resource does not belong to the selected project")
	}
	return nil
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

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
