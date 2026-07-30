package deploymentsvc

import (
	"context"
	"errors"
	"strconv"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
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
	exists, err := s.workspace.ServiceDirExists(app.Code, service.InstanceKey)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to inspect service workspace", err)
	}
	if !exists {
		return []deploymentdto.RuntimeContainer{}, nil
	}
	command := containerPsCommand(composeProjectName(app.Code, service.InstanceKey))
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, service.InstanceKey), command.Name, command.Args...)
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
	projectName := composeProjectName(app.Code, service.InstanceKey)
	command := containerLogsTailCommand(projectName, strconv.Itoa(tail))
	if component != "" {
		command = containerLogsTailCommand(projectName, strconv.Itoa(tail), component)
	}
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, service.InstanceKey), command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

// PreviewService renders a saved service configuration without creating a deployment.
func (s Service) PreviewService(ctx context.Context, userId string, serviceId string) (string, error) {
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
	exposes, err := s.executionStore.ServiceExposesByService(ctx, service.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, app.Code, service.InstanceKey)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to resolve physical service dir", err)
	}
	gateway, err := s.gatewayForDeployment(ctx, app, exposes)
	if err != nil {
		return "", err
	}
	content, err := s.RenderCompose(ctx, RenderInput{
		App: app, Version: version, Components: components, Exposes: exposes,
		Service: service, Gateway: gateway, RuntimeConfig: cloneRuntimeConfig(service.RuntimeConfig), PhysicalSvcDir: physicalDir,
	})
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	return content, nil
}

// DeleteApplication performs the runtime safety checks and optional workspace
// cleanup required before removing an application specification.
func (s Service) DeleteApplication(ctx context.Context, userId string, applicationId string, removeDir bool) error {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	services, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load services", err)
	}
	for _, service := range services {
		switch service.Status {
		case status.ServiceStatusDeploying:
			return apperror.New(apperror.KindValidation, "应用正在部署中, 请稍后再试")
		case status.ServiceStatusRunning:
			return apperror.New(apperror.KindValidation, "应用正在运行中, 请先停止后再删除")
		}
	}
	if removeDir {
		if err := s.workspace.RemoveAppDir(app.Code); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to remove application directory", err)
		}
	}
	if err := s.application.DeleteApplication(ctx, app.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application", err)
	}
	return nil
}
