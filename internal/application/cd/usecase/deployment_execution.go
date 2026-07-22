package cdsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string, forceRecreate bool) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	versionId := ""
	if deployment.VersionId != nil {
		versionId = strings.TrimSpace(*deployment.VersionId)
	}
	if versionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if deployment.EnvironmentId == nil || strings.TrimSpace(*deployment.EnvironmentId) == "" {
		err := fmt.Errorf("deployment %s missing environment_id", deployment.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	opts := parseDeployOptions(deployment.OptionsJSON)
	if forceRecreate {
		opts.ForceRecreate = true
	}
	instanceKey := strings.TrimSpace(opts.InstanceKey)
	if instanceKey == "" {
		instanceKey = "default"
	}

	version, err := s.executionStore.Version(ctx, versionId)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.Environment(ctx, *deployment.EnvironmentId)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if app.ProjectId == nil || *app.ProjectId != env.ProjectId {
		err := fmt.Errorf("application and environment must belong to the same project")
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	components, err := s.executionStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	exposes, err := s.executionStore.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}

	svc, err := s.upsertServiceDeploying(ctx, app.Id, env.Id, instanceKey, version.Id)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := s.renderAndDeployWithOptions(ctx, app, version, components, exposes, env, svc, deployment.Id, opts.ForceRecreate, opts.RuntimeConfig); err != nil {
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id, svc.LastSuccessfulVersionId)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	last := version.Id
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id, &last); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	version, err := s.executionStore.Version(ctx, svc.VersionId)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	components, err := s.executionStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	exposes, err := s.executionStore.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}

	if err := s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusDeploying); err != nil {
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	restartOpts := parseDeployOptions(deployment.OptionsJSON)
	if err := s.renderAndDeployWithOptions(ctx, app, version, components, exposes, env, svc, deployment.Id, false, restartOpts.RuntimeConfig); err != nil {
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id, svc.LastSuccessfulVersionId)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	last := version.Id
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id, &last); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationStop(ctx context.Context, applicationId string, deploymentId string, removeVolumes bool) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.Environment(ctx, svc.EnvironmentId)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	serviceDir := s.workspace.ServiceDir(app.Code, env.Code, svc.InstanceKey)
	logPath := s.workspace.DeploymentLogPath(app.Code, env.Code, svc.InstanceKey, deployment.Id)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, env.Code, svc.InstanceKey)
	command := stopComposeCommand(projectName, removeVolumes)
	if err := s.runner.Run(ctx, serviceDir, logWriter, command.Name, command.Args...); err != nil {
		_ = s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusFaulted)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusStopped); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) loadDeploymentExecution(ctx context.Context, applicationId string, deploymentId string) (model.Application, model.Deployment, error) {
	app, err := s.executionStore.Application(ctx, applicationId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	deployment, err := s.executionStore.Deployment(ctx, deploymentId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	return app, deployment, nil
}

func (s Service) resolveServiceFromDeployment(ctx context.Context, applicationId string, deployment model.Deployment) (model.Service, error) {
	if deployment.ServiceId != nil && strings.TrimSpace(*deployment.ServiceId) != "" {
		return s.executionStore.Service(ctx, *deployment.ServiceId)
	}
	opts := parseDeployOptions(deployment.OptionsJSON)
	instanceKey := strings.TrimSpace(opts.InstanceKey)
	if instanceKey == "" {
		instanceKey = "default"
	}
	if deployment.EnvironmentId == nil || strings.TrimSpace(*deployment.EnvironmentId) == "" {
		return model.Service{}, fmt.Errorf("deployment %s missing environment_id", deployment.Id)
	}
	return s.executionStore.ServiceByKey(ctx, applicationId, *deployment.EnvironmentId, instanceKey)
}

func (s Service) upsertServiceDeploying(ctx context.Context, applicationId string, environmentId string, instanceKey string, versionId string) (model.Service, error) {
	existing, err := s.executionStore.ServiceByKey(ctx, applicationId, environmentId, instanceKey)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, err
		}
		svc := model.Service{
			Id:            idutil.NewId(),
			ApplicationId: applicationId,
			EnvironmentId: environmentId,
			InstanceKey:   instanceKey,
			VersionId:     versionId,
			Status:        status.ServiceStatusDeploying,
		}
		if err := s.executionStore.UpsertService(ctx, svc); err != nil {
			return model.Service{}, err
		}
		return s.executionStore.ServiceByKey(ctx, applicationId, environmentId, instanceKey)
	}
	existing.VersionId = versionId
	existing.Status = status.ServiceStatusDeploying
	if err := s.executionStore.UpsertService(ctx, existing); err != nil {
		return model.Service{}, err
	}
	return s.executionStore.ServiceByKey(ctx, applicationId, environmentId, instanceKey)
}

func (s Service) renderAndDeploy(
	ctx context.Context,
	app model.Application,
	version model.Version,
	components []model.VersionComponent,
	exposes []model.VersionExpose,
	env model.Environment,
	svc model.Service,
	deploymentId string,
	forceRecreate bool,
) error {
	return s.renderAndDeployWithOptions(ctx, app, version, components, exposes, env, svc, deploymentId, forceRecreate, nil)
}

func (s Service) renderAndDeployWithOptions(
	ctx context.Context,
	app model.Application,
	version model.Version,
	components []model.VersionComponent,
	exposes []model.VersionExpose,
	env model.Environment,
	svc model.Service,
	deploymentId string,
	forceRecreate bool,
	runtimeConfig map[string]string,
) error {
	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, app.Code, env.Code, svc.InstanceKey)
	if err != nil {
		return err
	}
	gateway, err := s.optionalGatewayForRender(ctx, components, exposes, app)
	if err != nil {
		return err
	}
	result, err := s.RenderComposeDetailed(ctx, RenderInput{
		App: app, Version: version, Components: components, Exposes: exposes,
		Env: env, Service: svc, Gateway: gateway, RuntimeConfig: runtimeConfig, PhysicalSvcDir: physicalDir,
	})
	if err != nil {
		return err
	}

	serviceDir := s.workspace.ServiceDir(app.Code, env.Code, svc.InstanceKey)
	logPath := s.workspace.DeploymentLogPath(app.Code, env.Code, svc.InstanceKey, deploymentId)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(logWriter, "Rendering version %s (%s) with %d component(s) into env %s instance %s\n",
		version.Label, version.Id, len(components), env.Code, svc.InstanceKey); err != nil {
		return err
	}
	if len(result.ResolvedMounts) > 0 {
		if _, err := fmt.Fprintf(logWriter, "Materializing %d logical mount source(s)\n", countLogicalMounts(result.ResolvedMounts)); err != nil {
			return err
		}
		if err := MaterializeLogicalMountSources(result.ResolvedMounts); err != nil {
			return err
		}
	}
	if err := s.workspace.WriteConfig(app.Code, env.Code, svc.InstanceKey, "docker-compose.yml", result.Compose); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, env.Code, svc.InstanceKey)
	command := deployComposeCommand(projectName, app.ImagePullPolicy, forceRecreate)
	return s.runner.Run(ctx, serviceDir, logWriter, command.Name, command.Args...)
}

func countLogicalMounts(items []ResolvedMount) int {
	n := 0
	for _, item := range items {
		if item.SourceType == mountSourceLogical {
			n++
		}
	}
	return n
}

func parseDeployOptions(raw *string) cdto.DeployOptionsJSON {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return cdto.DeployOptionsJSON{}
	}
	var opts cdto.DeployOptionsJSON
	if err := json.Unmarshal([]byte(*raw), &opts); err != nil {
		return cdto.DeployOptionsJSON{}
	}
	return opts
}

func writeWorkingDirectory(w io.Writer, dir string) error {
	_, err := fmt.Fprintf(w, "Working directory: %s\n", dir)
	return err
}
