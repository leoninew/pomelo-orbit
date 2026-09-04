package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string, forceRecreate bool) error {
	begun, err := s.executionStore.BeginDeployment(ctx, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, deployment.Id)
	defer cancel()
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if forceRecreate {
		opts.ForceRecreate = true
	}
	if opts.InstanceKey == "" {
		err := fmt.Errorf("deployment %s missing instance_key", deployment.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, opts, target); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}

	version, err := s.executionStore.Version(ctx, *deployment.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc.VersionId = version.Id
	components, err := s.executionStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, svc, components, overlays, env, nil)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	if err := s.ensureSingleRuntime(ctx, app, opts.InstanceKey); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan.Gateway = cloneGatewayConfig(opts.GatewayConfig)
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	if requiresGatewayConfig(plan) && plan.Gateway == nil {
		err := fmt.Errorf("deployment %s is missing Gateway configuration snapshot", deployment.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, target, plan, deployment.Id, opts.ForceRecreate); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	if err := s.publishGatewayRoutes(ctx, plan); err != nil {
		_ = s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	return s.completeDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, applicationId string, deploymentId string) error {
	begun, err := s.executionStore.BeginDeployment(ctx, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, deployment.Id)
	defer cancel()
	restartOpts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, restartOpts, target); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	version, err := s.executionStore.Version(ctx, *deployment.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc.VersionId = version.Id
	components, err := s.executionStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, svc, components, overlays, env, nil)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	setPlanJoinTraefikNetwork(&plan, restartOpts.JoinTraefikNetwork)
	plan.Gateway = cloneGatewayConfig(restartOpts.GatewayConfig)
	setPlanJoinTraefikNetwork(&plan, restartOpts.JoinTraefikNetwork)
	if requiresGatewayConfig(plan) && plan.Gateway == nil {
		err := fmt.Errorf("deployment %s is missing Gateway configuration snapshot", deployment.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, target, plan, deployment.Id, false); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	if err := s.publishGatewayRoutes(ctx, plan); err != nil {
		_ = s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	return s.completeDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) publishGatewayRoutes(ctx context.Context, plan model.EffectiveServicePlan) error {
	if !isGatewayCarrier(plan) {
		return nil
	}
	if s.gatewayRoutePublisher == nil {
		return fmt.Errorf("gateway route publisher is not configured")
	}
	if plan.Application.ProjectId == nil || strings.TrimSpace(*plan.Application.ProjectId) == "" {
		return fmt.Errorf("gateway deployment is missing project scope")
	}
	if err := s.gatewayRoutePublisher.PublishSnapshot(ctx, *plan.Application.ProjectId); err != nil {
		return fmt.Errorf("publish gateway route snapshot: %w", err)
	}
	return nil
}

func (s Service) ExecuteApplicationStop(ctx context.Context, applicationId string, deploymentId string, removeVolumes bool) error {
	begun, err := s.executionStore.BeginDeployment(ctx, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, deployment.Id)
	defer cancel()
	options, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, options, target); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	serviceDir, err := s.remoteRuntime.ServiceDir(target, svc.Code)
	if err != nil {
		return err
	}
	logWriter, err := s.logStore.Writer(svc.Code, deployment.Id)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if _, err := fmt.Fprintln(logWriter, "Preparing service shutdown"); err != nil {
		return err
	}
	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	if removeVolumes {
		if _, err := fmt.Fprintln(logWriter, "Stopping services and removing volumes"); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintln(logWriter, "Stopping services"); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
	command := stopComposeCommand(projectName, removeVolumes)
	if err := s.remoteRuntime.Run(executionCtx, target, svc.Code, logWriter, command.Name, command.Args...); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceStatus(ctx, svc.Id, status.ServiceStatusStopped); err != nil {
		return err
	}
	return s.completeDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}
func (s Service) deploymentCanceled(ctx context.Context, deploymentID string) bool {
	deployment, err := s.executionStore.Deployment(ctx, deploymentID)
	return err == nil && deployment.Status == status.WorkStatusCanceled
}

// reconcileCanceledService records the actual Compose runtime after a canceled
// command. The Deployment remains canceled regardless of observation failures.
func (s Service) reconcileCanceledService(ctx context.Context, target environmentport.SSHTarget, app model.Application, svc model.Service) {
	serviceStatus := status.ServiceStatusFaulted
	if s.remoteRuntime != nil {
		exists, err := s.remoteRuntime.ServiceDirExists(ctx, target, svc.Code)
		if err == nil && !exists {
			serviceStatus = status.ServiceStatusStopped
		} else if err == nil {
			command := containerPsCommand(composeProjectName(app.Code, svc.InstanceKey))
			output, runErr := s.remoteRuntime.Query(ctx, target, svc.Code, command.Name, command.Args...)
			if runErr == nil {
				containers, parseErr := parseComposePsOutput(output)
				if parseErr == nil {
					serviceStatus = observedServiceStatus(containers)
				}
			}
		}
	}
	_ = s.executionStore.UpdateServiceStatus(ctx, svc.Id, serviceStatus)
}
func observedServiceStatus(containers []deploymentdto.RuntimeContainer) string {
	if len(containers) == 0 {
		return status.ServiceStatusStopped
	}
	for _, container := range containers {
		if !strings.EqualFold(strings.TrimSpace(container.State), "running") {
			return status.ServiceStatusFaulted
		}
	}
	return status.ServiceStatusRunning
}

func (s Service) completeDeployment(ctx context.Context, deploymentID, statusValue, message string) error {
	_, err := s.executionStore.CompleteDeployment(ctx, deploymentID, statusValue, message)
	return err
}

func (s Service) deploymentExecutionContext(ctx context.Context, deploymentID string) (context.Context, context.CancelFunc) {
	monitoredCtx, cancelMonitored := context.WithCancel(ctx)
	done := make(chan struct{})
	interval := s.pollInterval
	if interval <= 0 {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-monitoredCtx.Done():
				return
			case <-ticker.C:
				deployment, err := s.executionStore.Deployment(ctx, deploymentID)
				if err == nil && deployment.Status == status.WorkStatusCanceled {
					cancelMonitored()
					return
				}
			}
		}
	}()
	return monitoredCtx, func() {
		close(done)
		cancelMonitored()
	}
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

func (s Service) renderAndDeployWithOptions(ctx context.Context, target environmentport.SSHTarget, plan model.EffectiveServicePlan, deploymentId string, forceRecreate bool) error {
	app, version, svc := plan.Application, plan.Version, plan.Service
	if s.remoteRuntime == nil {
		return fmt.Errorf("remote deployment runtime is not configured")
	}
	serviceDir, err := s.remoteRuntime.ServiceDir(target, svc.Code)
	if err != nil {
		return err
	}
	logWriter, err := s.logStore.Writer(svc.Code, deploymentId)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()
	if _, err := fmt.Fprintln(logWriter, "Preparing deployment"); err != nil {
		return err
	}
	if err := writeWorkingDirectory(logWriter, serviceDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(logWriter, "Rendering version %s (%s) with %d component(s) into service %s\n",
		version.Label, version.Id, len(plan.Components), svc.Code); err != nil {
		return err
	}
	result, err := s.RenderComposeDetailed(ctx, RenderInput{Plan: plan, LogicalSvcDir: serviceDir})
	if err != nil {
		return err
	}
	workspace, err := remoteWorkspaceFromRender(svc.Code, deploymentId, result)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(logWriter, "Staging deployment configuration on project environment"); err != nil {
		return err
	}
	if err := s.remoteRuntime.StageWorkspace(ctx, target, workspace); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
	command := deployComposeCommand(projectName, deploymentPullPolicy(plan), forceRecreate)
	if _, err := fmt.Fprintln(logWriter, "Starting services"); err != nil {
		return err
	}
	return s.remoteRuntime.Run(ctx, target, svc.Code, logWriter, command.Name, command.Args...)
}

func remoteWorkspaceFromRender(serviceCode string, deploymentID string, result RenderResult) (deploymentport.RemoteWorkspace, error) {
	workspace := deploymentport.RemoteWorkspace{ServiceCode: serviceCode, DeploymentID: deploymentID, Compose: result.Compose}
	for _, item := range result.ResolvedMounts {
		if !item.ShouldMaterialize {
			continue
		}
		if strings.TrimSpace(item.LogicalSource) == "" {
			return deploymentport.RemoteWorkspace{}, fmt.Errorf("logical mount source is required for remote materialization")
		}
		if item.IsFile {
			workspace.Files = append(workspace.Files, deploymentport.RemoteFile{
				Path: item.LogicalSource, Content: []byte(item.Content), Mode: uint32(item.FileMode.Perm()), IgnoreIfExists: item.IgnoreIfExists,
			})
			continue
		}
		workspace.Directories = append(workspace.Directories, item.LogicalSource)
	}
	return workspace, nil
}
func verifyDeploymentPlanHash(deployment model.Deployment, plan model.EffectiveServicePlan) error {
	if deployment.EffectivePlanHash == nil || *deployment.EffectivePlanHash == "" {
		return fmt.Errorf("deployment %s is missing effective plan hash", deployment.Id)
	}
	current, err := EffectiveServicePlanHash(plan)
	if err != nil {
		return err
	}
	if current != *deployment.EffectivePlanHash {
		return fmt.Errorf("service configuration changed after deployment was queued; deploy again")
	}
	return nil
}

func countLogicalMounts(items []ResolvedMount) int {
	n := 0
	for _, item := range items {
		if item.ShouldMaterialize {
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

// ensureSingleRuntime rejects a second active service binding for the same Application.
func (s Service) ensureSingleRuntime(ctx context.Context, app model.Application, instanceKey string) error {
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
