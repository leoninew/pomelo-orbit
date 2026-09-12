package deploymentsvc

import (
	"context"
	"errors"
	"path"
	"strconv"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// ApplicationStatus returns normalized container statuses for a resolved runtime service.
func (s Service) ApplicationStatus(ctx context.Context, userId string, applicationId string, input deploymentdto.ServiceTargetInput) ([]deploymentdto.RuntimeContainer, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return nil, err
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		return nil, err
	}
	exists, err := s.runtime.ServiceDirExists(ctx, target, service.Code)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to inspect service workspace", err)
	}
	if !exists {
		return []deploymentdto.RuntimeContainer{}, nil
	}
	command := containerPsCommand(composeProjectName(service.Code))
	output, err := s.runtime.Query(ctx, target, service.Code, command.Name, command.Args...)
	if err != nil {
		return nil, apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	containers, err := parseComposePsOutput(output)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to parse compose status", err)
	}
	if err := s.populateContainerComponentIds(ctx, service.VersionId, containers); err != nil {
		return nil, err
	}
	return containers, nil
}

func (s Service) populateContainerComponentIds(ctx context.Context, versionId string, containers []deploymentdto.RuntimeContainer) error {
	if versionId == "" || len(containers) == 0 {
		return nil
	}
	components, err := s.application.VersionComponentsByVersion(ctx, versionId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list version components", err)
	}
	componentIds := make(map[string]string, len(components))
	for _, component := range components {
		componentIds[component.Name] = component.Id
	}
	applyContainerComponentIds(containers, versionId, componentIds)
	return nil
}

func applyContainerComponentIds(containers []deploymentdto.RuntimeContainer, versionId string, componentIds map[string]string) {
	for index := range containers {
		containers[index].VersionId = versionId
		containers[index].ComponentId = componentIds[containers[index].Service]
	}
}

// ApplicationLogs returns recent compose logs for a resolved runtime service.
func (s Service) ApplicationLogs(ctx context.Context, userId string, applicationId string, tail int, input deploymentdto.ServiceTargetInput, component string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if tail < 1 || tail > 1000 {
		return "", apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		return "", err
	}
	exists, err := s.runtime.ServiceDirExists(ctx, target, service.Code)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to inspect service workspace", err)
	}
	if !exists {
		return "", nil
	}
	projectName := composeProjectName(service.Code)
	command := containerLogsTailCommand(projectName, strconv.Itoa(tail))
	if component != "" {
		command = containerLogsTailCommand(projectName, strconv.Itoa(tail), component)
	}
	output, err := s.runtime.Query(ctx, target, service.Code, command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

// PreviewService renders a saved service configuration without creating a deployment.
func (s Service) PreviewService(ctx context.Context, userId string, serviceId string, input deploymentdto.PreviewComposeInput) (string, error) {
	if s.commandStore == nil || s.executionStore == nil {
		return "", apperror.New(apperror.KindInternal, "deployment stores are not configured")
	}
	service, app, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return "", err
	}
	version, err := s.commandStore.Version(ctx, service.VersionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Version "+service.VersionId+" not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, service.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list service components", err)
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, service.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load service environment", err)
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, service, components, overlays, env, nil)
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	setPlanJoinTraefikNetwork(&plan, previewJoinTraefikNetwork(input))
	return s.renderComposePreview(ctx, app, plan)
}

// PreviewVersion renders version declarations without reading Service runtime configuration.
func (s Service) PreviewVersion(ctx context.Context, userId string, versionId string, input deploymentdto.PreviewComposeInput) (string, error) {
	if s.commandStore == nil {
		return "", apperror.New(apperror.KindInternal, "deployment command store is not configured")
	}
	version, err := s.commandStore.Version(ctx, versionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Version "+versionId+" not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	app, err := s.loadApplicationForUser(ctx, userId, version.ApplicationId)
	if err != nil {
		return "", err
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	plan, err := BuildVersionPreviewPlan(app, version, components, nil)
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	setPlanJoinTraefikNetwork(&plan, previewJoinTraefikNetwork(input))
	return s.renderComposePreview(ctx, app, plan)
}

// renderComposePreview renders desired state only. SSH target validation and
// remote workspace resolution apply to execution, not to a Compose preview.
func (s Service) renderComposePreview(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (string, error) {
	gateway, err := s.gatewayForDeployment(ctx, app, plan)
	if err != nil {
		return "", err
	}
	plan.Gateway = gateway
	content, err := s.RenderCompose(ctx, RenderInput{
		Plan:          plan,
		LogicalSvcDir: path.Join("/.pomelo-orbit-preview", plan.Service.Code),
	})
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	return content, nil
}

// DeleteApplication performs runtime safety checks before removing an
// application specification. Service workspaces are not application-owned.
func (s Service) DeleteApplication(ctx context.Context, userId string, applicationId string) error {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	services, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load services", err)
	}
	if len(services) > 0 {
		return apperror.New(apperror.KindValidation, "应用仍包含服务, 请先删除服务")
	}
	versions, err := s.application.ListVersions(ctx, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load versions", err)
	}
	if len(versions) > 0 {
		return apperror.New(apperror.KindValidation, "应用仍包含版本, 请先删除版本")
	}
	if err := s.application.DeleteApplication(ctx, app.Id); err != nil {
		if errors.Is(err, repository.ErrReferenced) {
			return apperror.New(apperror.KindValidation, "应用包含被引用的版本, 无法删除")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application", err)
	}
	return nil
}
