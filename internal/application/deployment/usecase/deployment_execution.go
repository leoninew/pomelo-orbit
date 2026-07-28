package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	gatewayport "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	runtimeconfig "gitee.com/leoninew/PomeloOrbit-go/internal/common/runtimeconfig"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string, forceRecreate bool) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if forceRecreate {
		opts.ForceRecreate = true
	}
	if opts.InstanceKey == "" {
		err := fmt.Errorf("deployment %s missing instance_key", deployment.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}

	version, err := s.executionStore.Version(ctx, *deployment.VersionId)
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
	if err := s.ensureSingleRuntime(ctx, app, opts.InstanceKey); err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	preparation, err := s.prepareGatewayDeployment(ctx, app, exposes)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if preparation.RolloutConfig != nil {
		if err := s.deployGatewayInPlace(ctx, preparation.RolloutConfig, deployment.Id); err != nil {
			_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
			return err
		}
	}

	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := s.renderAndDeployWithOptions(ctx, app, version, components, exposes, svc, preparation.RenderConfig, deployment.Id, opts.ForceRecreate, opts.RuntimeConfig); err != nil {
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	restartOpts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
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
	preparation, err := s.prepareGatewayDeployment(ctx, app, exposes)
	if err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if preparation.RolloutConfig != nil {
		if err := s.deployGatewayInPlace(ctx, preparation.RolloutConfig, deployment.Id); err != nil {
			_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
			return err
		}
	}

	if err := s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusDeploying); err != nil {
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := s.renderAndDeployWithOptions(ctx, app, version, components, exposes, svc, preparation.RenderConfig, deployment.Id, false, restartOpts.RuntimeConfig); err != nil {
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
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
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	serviceDir := s.workspace.ServiceDir(app.Code, svc.InstanceKey)
	logPath := s.workspace.DeploymentLogPath(app.Code, svc.InstanceKey, deployment.Id)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
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
	if deployment.ServiceId == nil || *deployment.ServiceId == "" {
		return model.Service{}, fmt.Errorf("deployment %s missing service_id", deployment.Id)
	}
	svc, err := s.executionStore.Service(ctx, *deployment.ServiceId)
	if err != nil {
		return model.Service{}, err
	}
	if svc.ApplicationId != applicationId {
		return model.Service{}, fmt.Errorf("deployment %s service_id does not belong to application", deployment.Id)
	}
	return svc, nil
}

func (s Service) renderAndDeployWithOptions(
	ctx context.Context,
	app model.Application,
	version model.Version,
	components []model.VersionComponent,
	exposes []model.VersionExpose,
	svc model.Service,
	gateway *model.GatewayConfig,
	deploymentId string,
	forceRecreate bool,
	runtimeConfig map[string]string,
) error {
	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, app.Code, svc.InstanceKey)
	if err != nil {
		return err
	}
	resolvedRuntimeConfig, extraKeys, err := runtimeconfig.Resolve(runtimeConfig, version.EnvJSON, components)
	if err != nil {
		return err
	}
	result, err := s.RenderComposeDetailed(ctx, RenderInput{
		App: app, Version: version, Components: components, Exposes: exposes,
		Service: svc, Gateway: gateway, RuntimeConfig: resolvedRuntimeConfig, PhysicalSvcDir: physicalDir,
	})
	if err != nil {
		return err
	}

	serviceDir := s.workspace.ServiceDir(app.Code, svc.InstanceKey)
	logPath := s.workspace.DeploymentLogPath(app.Code, svc.InstanceKey, deploymentId)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()
	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	if len(extraKeys) > 0 {
		if _, err := fmt.Fprintf(logWriter, "Warning: ignored unused runtime config keys: %s\n", strings.Join(extraKeys, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(logWriter, "Rendering version %s (%s) with %d component(s) into instance %s\n",
		version.Label, version.Id, len(components), svc.InstanceKey); err != nil {
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
	if err := s.workspace.WriteConfig(app.Code, svc.InstanceKey, "docker-compose.yml", result.Compose); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
	command := deployComposeCommand(projectName, app.ImagePullPolicy, forceRecreate)
	if err := s.runner.Run(ctx, serviceDir, logWriter, command.Name, command.Args...); err != nil {
		return err
	}
	return nil
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

func parseDeployOptions(raw *string) (deploymentdto.DeployOptionsJSON, error) {
	if raw == nil || *raw == "" {
		return deploymentdto.DeployOptionsJSON{}, errors.New("options_json is required")
	}
	var opts deploymentdto.DeployOptionsJSON
	if err := json.Unmarshal([]byte(*raw), &opts); err != nil {
		return deploymentdto.DeployOptionsJSON{}, err
	}
	return opts, nil
}

func writeWorkingDirectory(w io.Writer, dir string) error {
	_, err := fmt.Fprintf(w, "Working directory: %s\n", dir)
	return err
}

// ensureSingleRuntime rejects a second active service binding for the same standard app.
func (s Service) ensureSingleRuntime(ctx context.Context, app model.Application, instanceKey string) error {
	if app.Kind == status.ApplicationKindGateway {
		return nil
	}
	if s.store == nil {
		return nil
	}
	services, err := s.store.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return err
	}
	for _, svc := range services {
		if !isActiveServiceStatus(svc.Status) {
			continue
		}
		if svc.InstanceKey == instanceKey {
			continue // same binding — replace in place
		}
		return fmt.Errorf("application already has an active runtime (service %s status=%s); single runtime only", svc.Id, svc.Status)
	}
	return nil
}

func (s Service) prepareGatewayDeployment(ctx context.Context, app model.Application, exposes []model.VersionExpose) (gatewayport.DeploymentPreparation, error) {
	if s.gatewayCoordinator == nil {
		return gatewayport.DeploymentPreparation{}, fmt.Errorf("gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.PrepareDeployment(ctx, app, exposes)
}

func (s Service) deployGatewayInPlace(ctx context.Context, gateway *model.GatewayConfig, parentDeploymentId string) error {
	if gateway == nil || s.store == nil {
		return nil
	}
	gwApp, err := s.store.Application(ctx, gateway.ApplicationId)
	if err != nil {
		return err
	}
	services, err := s.store.ListServicesByApplication(ctx, gwApp.Id)
	if err != nil {
		return err
	}
	var active *model.Service
	for i := range services {
		if isActiveServiceStatus(services[i].Status) || services[i].Status == status.ServiceStatusStopped || services[i].Status == status.ServiceStatusFaulted {
			// Prefer running/deploying; else last service for env
			if isActiveServiceStatus(services[i].Status) {
				active = &services[i]
				break
			}
			if active == nil {
				active = &services[i]
			}
		}
	}
	if active == nil {
		// Gateway not deployed yet: compile only is enough; first gateway deploy will pick ports.
		return nil
	}
	version, err := s.store.Version(ctx, active.VersionId)
	// After compile, unpublished managed version may be newer; use unpublished if present.
	versions, listErr := s.store.ListVersions(ctx, gwApp.Id)
	if listErr == nil {
		for _, v := range versions {
			if v.Status == status.VersionStatusUnpublished {
				version = v
				break
			}
		}
	}
	if err != nil && version.Id == "" {
		return err
	}
	components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return err
	}
	exposes, err := s.store.VersionExposesByVersion(ctx, version.Id)
	if err != nil {
		return err
	}
	// Mark deploying and roll gateway with force recreate.
	active.VersionId = version.Id
	active.Status = status.ServiceStatusDeploying
	if err := s.store.UpsertService(ctx, *active); err != nil {
		return err
	}
	logID := parentDeploymentId + "-gw"
	if err := s.renderAndDeployWithOptions(ctx, gwApp, version, components, exposes, *active, gateway, logID, true, cloneRuntimeConfig(active.RuntimeConfig)); err != nil {
		_ = s.store.UpdateServiceAfterDeploy(ctx, active.Id, status.ServiceStatusFaulted, version.Id)
		return fmt.Errorf("gateway reconcile deploy failed (business deploy aborted): %w", err)
	}
	return s.store.UpdateServiceAfterDeploy(ctx, active.Id, status.ServiceStatusRunning, version.Id)
}
