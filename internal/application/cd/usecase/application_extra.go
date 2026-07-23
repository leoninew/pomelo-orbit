package cdsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ImportApplication(ctx context.Context, userId string, input cdto.ApplicationImportInput) (model.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	name, code, kind, imagePullPolicy, err := normalizeApplicationImportInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, code); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: kind, ImagePullPolicy: imagePullPolicy}
	if err := s.store.CreateApplication(ctx, app); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to import application", err)
	}
	label := strings.TrimSpace(input.VersionLabel)
	if label == "" {
		label = "v1"
	}
	if _, err := s.CreateVersion(ctx, userId, cdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         label,
		EnvJSON:       input.VersionEnvJSON,
		Note:          input.VersionNote,
		Components:    input.Components,
		Exposes:       input.Exposes,
	}); err != nil {
		_ = s.store.DeleteApplication(ctx, app.Id)
		return model.Application{}, err
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ExportApplication(ctx context.Context, userId string, applicationId string) (cdto.ApplicationExport, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return cdto.ApplicationExport{}, err
	}
	versions, err := s.ListVersions(ctx, userId, app.Id)
	if err != nil {
		return cdto.ApplicationExport{}, err
	}
	services, err := s.ListServicesByApplication(ctx, userId, app.Id)
	if err != nil {
		return cdto.ApplicationExport{}, err
	}
	return cdto.ApplicationExport{Application: app, Versions: versions, Services: services}, nil
}

func (s Service) StopApplication(ctx context.Context, userId string, applicationId string, input cdto.ApplicationServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	svc, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	if svc.Status != status.ServiceStatusRunning && svc.Status != status.ServiceStatusFaulted {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法停止")
	}
	env, err := s.store.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	deployment := newApplicationDeployment(app, "stop")
	deployment.ServiceId = &svc.Id
	deployment.VersionId = &svc.VersionId
	deployment.EnvironmentId = &svc.EnvironmentId
	opts := cdto.DeployOptionsJSON{InstanceKey: svc.InstanceKey, RemoveVolumes: input.RemoveVolumes}
	raw, _ := json.Marshal(opts)
	text := string(raw)
	deployment.OptionsJSON = &text
	deployment.CommandText = stopComposeCommand(composeProjectName(app.Code, env.Code, svc.InstanceKey), input.RemoveVolumes).String()
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchApplicationStop(ctx, cdto.ApplicationStopDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) RestartApplication(ctx context.Context, userId string, applicationId string, input cdto.ApplicationServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	svc, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	if svc.Status != status.ServiceStatusRunning && svc.Status != status.ServiceStatusFaulted {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法重启")
	}
	version, err := s.store.Version(ctx, svc.VersionId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := validateVersionComponents(components); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	env, err := s.store.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	deployment := newApplicationDeployment(app, "restart")
	deployment.ServiceId = &svc.Id
	deployment.VersionId = &version.Id
	deployment.EnvironmentId = &svc.EnvironmentId
	opts := cdto.DeployOptionsJSON{InstanceKey: svc.InstanceKey}
	raw, _ := json.Marshal(opts)
	text := string(raw)
	deployment.OptionsJSON = &text
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, env.Code, svc.InstanceKey), app.ImagePullPolicy, false).String()
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchApplicationRestart(ctx, cdto.ApplicationRestartDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ApplicationStatus(ctx context.Context, userId string, applicationId string, input cdto.ApplicationServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	svc, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	env, err := s.store.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	projectName := composeProjectName(app.Code, env.Code, svc.InstanceKey)
	command := containerPsCommand(projectName)
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, env.Code, svc.InstanceKey), command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

func (s Service) ApplicationLogs(ctx context.Context, userId string, applicationId string, tail int, input cdto.ApplicationServiceTargetInput, component string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
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
	svc, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	env, err := s.store.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	projectName := composeProjectName(app.Code, env.Code, svc.InstanceKey)
	var command composeCommand
	if component != "" {
		command = containerLogsTailCommand(projectName, strconv.Itoa(tail), component)
	} else {
		command = containerLogsTailCommand(projectName, strconv.Itoa(tail))
	}
	output, err := s.queryRunner.Run(ctx, s.workspace.ServiceDir(app.Code, env.Code, svc.InstanceKey), command.Name, command.Args...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

// normalizeComposeServiceName validates an optional compose service/component name for log filtering.
func normalizeComposeServiceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	// Reject values that could be interpreted as docker CLI flags or paths.
	if strings.HasPrefix(name, "-") || strings.ContainsAny(name, "/\\ \t\n") {
		return "", apperror.New(apperror.KindValidation, "invalid component name")
	}
	return name, nil
}

func (s Service) DeploymentContainerLog(ctx context.Context, userId string, deploymentId string, tail int) (cdto.DeploymentContainerLog, error) {
	if tail < 1 || tail > 1000 {
		return cdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return cdto.DeploymentContainerLog{}, err
	}
	if deployment.OperationType == "stop" {
		return cdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "container logs are not available for stop deployments")
	}
	if deployment.Status == status.WorkStatusWaitingToRun {
		return cdto.DeploymentContainerLog{Source: "pending", IsRealtimeSupported: true}, nil
	}
	if deployment.ApplicationId == nil || strings.TrimSpace(*deployment.ApplicationId) == "" {
		return cdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.store.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return cdto.DeploymentContainerLog{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return cdto.DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		return cdto.DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	env, err := s.store.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		return cdto.DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	serviceDir := s.workspace.ServiceDir(app.Code, env.Code, svc.InstanceKey)
	projectName := composeProjectName(app.Code, env.Code, svc.InstanceKey)
	sinceCommand := containerLogsSinceCommand(projectName, deployment.StartedAt.UTC().Format(time.RFC3339))
	output, err := s.queryRunner.Run(ctx, serviceDir, sinceCommand.Name, sinceCommand.Args...)
	if err == nil {
		return cdto.DeploymentContainerLog{Logs: output, Source: "since", IsRealtimeSupported: true}, nil
	}
	tailCommand := containerLogsTailCommand(projectName, strconv.Itoa(tail))
	output, tailErr := s.queryRunner.Run(ctx, serviceDir, tailCommand.Name, tailCommand.Args...)
	if tailErr != nil {
		return cdto.DeploymentContainerLog{}, apperror.New(apperror.KindInternal, outputOrError(output, tailErr))
	}
	return cdto.DeploymentContainerLog{Logs: output, Source: "tail", IsRealtimeSupported: true}, nil
}

func (s Service) resolveServiceTarget(ctx context.Context, applicationId string, input cdto.ApplicationServiceTargetInput) (model.Service, error) {
	if strings.TrimSpace(input.ServiceId) != "" {
		svc, err := s.store.Service(ctx, strings.TrimSpace(input.ServiceId))
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
			}
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
		}
		if svc.ApplicationId != applicationId {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return svc, nil
	}
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if instanceKey == "" {
		instanceKey = "default"
	}
	environmentId := strings.TrimSpace(input.EnvironmentId)
	if environmentId == "" {
		services, err := s.store.ListServicesByApplication(ctx, applicationId)
		if err != nil {
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
		}
		if len(services) == 0 {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		if len(services) == 1 {
			return services[0], nil
		}
		return model.Service{}, apperror.New(apperror.KindValidation, "environment_id is required when multiple services exist")
	}
	svc, err := s.store.ServiceByKey(ctx, applicationId, environmentId, instanceKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return svc, nil
}

func normalizeApplicationImportInput(input cdto.ApplicationImportInput) (string, string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	kind, err := normalizeApplicationKind(input.Kind)
	if err != nil {
		return "", "", "", "", err
	}
	imagePullPolicy := strings.TrimSpace(input.ImagePullPolicy)
	if imagePullPolicy == "" {
		imagePullPolicy = "missing"
	}
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) || !validImagePullPolicy(imagePullPolicy) {
		return "", "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, kind, imagePullPolicy, nil
}

func (s Service) ensureApplicationCodeAvailable(ctx context.Context, code string) error {
	existing, err := s.store.ApplicationByCode(ctx, code)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	return nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}
