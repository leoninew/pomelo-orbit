package deploymentsvc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// ApplicationStatus returns the compose status for a resolved runtime service.
func (s Service) ApplicationStatus(ctx context.Context, userID string, applicationID string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userID, applicationID)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	env, err := s.commandStore.Environment(ctx, service.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	command := containerPsCommand(composeProjectName(app.Code, env.Code, service.InstanceKey))
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, env.Code, service.InstanceKey), command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

// ApplicationLogs returns recent compose logs for a resolved runtime service.
func (s Service) ApplicationLogs(ctx context.Context, userID string, applicationID string, tail int, input deploymentdto.ServiceTargetInput, component string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userID, applicationID)
	if err != nil {
		return "", err
	}
	if tail < 1 || tail > 1000 {
		return "", apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	component, err = normalizeComposeServiceName(component)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	env, err := s.commandStore.Environment(ctx, service.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	projectName := composeProjectName(app.Code, env.Code, service.InstanceKey)
	command := containerLogsTailCommand(projectName, strconv.Itoa(tail))
	if component != "" {
		command = containerLogsTailCommand(projectName, strconv.Itoa(tail), component)
	}
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, env.Code, service.InstanceKey), command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

// PreviewVersion renders the deployment compose file without creating a deployment.
func (s Service) PreviewVersion(ctx context.Context, userID string, versionID string, environmentID string, instanceKey string) (string, error) {
	if s.commandStore == nil || s.executionStore == nil {
		return "", apperror.New(apperror.KindInternal, "deployment stores are not configured")
	}
	versionID = strings.TrimSpace(versionID)
	version, err := s.commandStore.Version(ctx, versionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Version "+versionID+" not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	app, err := s.loadApplicationForUser(ctx, userID, version.ApplicationId)
	if err != nil {
		return "", err
	}
	environmentID = strings.TrimSpace(environmentID)
	if environmentID == "" {
		return "", apperror.New(apperror.KindValidation, "environment_id is required")
	}
	env, err := s.commandStore.Environment(ctx, environmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Environment not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	if app.ProjectId == nil || *app.ProjectId != env.ProjectId {
		return "", apperror.New(apperror.KindValidation, "application and environment must belong to the same project")
	}
	instanceKey, err = normalizeInstanceKey(instanceKey)
	if err != nil {
		return "", err
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	exposes, err := s.executionStore.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
	}
	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, app.Code, env.Code, instanceKey)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to resolve physical service dir", err)
	}
	gateway, err := s.gatewayForDeployment(ctx, app, exposes)
	if err != nil {
		return "", err
	}
	content, err := s.RenderCompose(ctx, RenderInput{
		App: app, Version: version, Components: components, Exposes: exposes,
		Env: env, Service: model.Service{InstanceKey: instanceKey}, Gateway: gateway, PhysicalSvcDir: physicalDir,
	})
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	return content, nil
}

// DeleteApplication performs the runtime safety checks and optional workspace
// cleanup required before removing an application specification.
func (s Service) DeleteApplication(ctx context.Context, userID string, applicationID string, removeDir bool) error {
	app, err := s.loadApplicationForUser(ctx, userID, applicationID)
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

func normalizeComposeServiceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	if strings.HasPrefix(name, "-") || strings.ContainsAny(name, "/\\ \\t\\n") {
		return "", apperror.New(apperror.KindValidation, "invalid component name")
	}
	return name, nil
}
