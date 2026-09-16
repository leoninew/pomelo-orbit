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

func (s Service) ExecuteApplicationDeploy(ctx context.Context, projectId string, applicationId string, deploymentId string, forceRecreate bool) error {
	begun, err := s.executionStore.BeginDeployment(ctx, projectId, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, projectId, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, projectId, deployment.Id)
	defer cancel()
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, projectId, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if forceRecreate {
		opts.ForceRecreate = true
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, opts, target); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}

	version, err := s.executionStore.Version(ctx, projectId, *deployment.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc.VersionId = version.Id
	components, err := s.executionStore.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, projectId, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, projectId, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, svc, components, overlays, env, nil)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	plan.Gateway = cloneGatewayConfig(opts.GatewayConfig)
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	if requiresGatewayConfig(plan) && plan.Gateway == nil {
		err := fmt.Errorf("deployment %s is missing Gateway configuration snapshot", deployment.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, target, plan, deployment.Id, opts.ForceRecreate); err != nil {
		if s.deploymentCanceled(ctx, projectId, deployment.Id) {
			s.reconcileCanceledService(ctx, projectId, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, projectId, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, projectId, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	if err := s.publishGatewayRoutes(ctx, projectId, plan); err != nil {
		_ = s.executionStore.UpdateServiceStatus(ctx, projectId, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	return s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, projectId string, applicationId string, deploymentId string) error {
	begun, err := s.executionStore.BeginDeployment(ctx, projectId, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, projectId, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, projectId, deployment.Id)
	defer cancel()
	restartOpts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, restartOpts, target); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, projectId, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		err := fmt.Errorf("deployment %s missing version_id", deployment.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	version, err := s.executionStore.Version(ctx, projectId, *deployment.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc.VersionId = version.Id
	components, err := s.executionStore.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, projectId, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, projectId, svc.Id)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, svc, components, overlays, env, nil)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	setPlanJoinTraefikNetwork(&plan, restartOpts.JoinTraefikNetwork)
	plan.Gateway = cloneGatewayConfig(restartOpts.GatewayConfig)
	setPlanJoinTraefikNetwork(&plan, restartOpts.JoinTraefikNetwork)
	if requiresGatewayConfig(plan) && plan.Gateway == nil {
		err := fmt.Errorf("deployment %s is missing Gateway configuration snapshot", deployment.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, target, plan, deployment.Id, false); err != nil {
		if s.deploymentCanceled(ctx, projectId, deployment.Id) {
			s.reconcileCanceledService(ctx, projectId, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, projectId, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, projectId, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	if err := s.publishGatewayRoutes(ctx, projectId, plan); err != nil {
		_ = s.executionStore.UpdateServiceStatus(ctx, projectId, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	return s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) publishGatewayRoutes(ctx context.Context, projectId string, plan model.EffectiveServicePlan) error {
	if !isGatewayCarrier(plan) {
		return nil
	}
	if s.gatewayRoutePublisher == nil {
		return fmt.Errorf("gateway route publisher is not configured")
	}
	if err := s.gatewayRoutePublisher.PublishSnapshot(ctx, projectId); err != nil {
		return fmt.Errorf("publish gateway route snapshot: %w", err)
	}
	return nil
}

func (s Service) ExecuteApplicationStop(ctx context.Context, projectId string, applicationId string, deploymentId string, removeVolumes bool) error {
	begun, err := s.executionStore.BeginDeployment(ctx, projectId, deploymentId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	app, deployment, err := s.loadDeploymentExecution(ctx, projectId, applicationId, deploymentId)
	if err != nil {
		return err
	}
	executionCtx, cancel := s.deploymentExecutionContext(ctx, projectId, deployment.Id)
	defer cancel()
	options, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		err = fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, options, target); err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, projectId, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	serviceDir, err := s.runtime.ServiceDir(target, svc.Code)
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
	projectName := composeProjectName(svc.Code)
	command := stopComposeCommand(projectName, removeVolumes)
	if err := s.runtime.Run(executionCtx, target, svc.Code, logWriter, command.Name, command.Args...); err != nil {
		if s.deploymentCanceled(ctx, projectId, deployment.Id) {
			s.reconcileCanceledService(ctx, projectId, target, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceStatus(ctx, projectId, svc.Id, status.ServiceStatusFaulted)
		_ = s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceStatus(ctx, projectId, svc.Id, status.ServiceStatusStopped); err != nil {
		return err
	}
	return s.completeDeployment(ctx, projectId, deployment.Id, status.WorkStatusRanToCompletion, "")
}
func (s Service) deploymentCanceled(ctx context.Context, projectId string, deploymentId string) bool {
	deployment, err := s.executionStore.Deployment(ctx, projectId, deploymentId)
	return err == nil && deployment.Status == status.WorkStatusCanceled
}

// reconcileCanceledService records the actual Compose runtime after a canceled
// command. The Deployment remains canceled regardless of observation failures.
func (s Service) reconcileCanceledService(ctx context.Context, projectId string, target environmentport.Target, app model.Application, svc model.Service) {
	serviceStatus := status.ServiceStatusFaulted
	if s.runtime != nil {
		exists, err := s.runtime.ServiceDirExists(ctx, target, svc.Code)
		if err == nil && !exists {
			serviceStatus = status.ServiceStatusStopped
		} else if err == nil {
			command := containerPsCommand(composeProjectName(svc.Code))
			output, runErr := s.runtime.Query(ctx, target, svc.Code, command.Name, command.Args...)
			if runErr == nil {
				containers, parseErr := parseComposePsOutput(output)
				if parseErr == nil {
					serviceStatus = observedServiceStatus(containers)
				}
			}
		}
	}
	_ = s.executionStore.UpdateServiceStatus(ctx, projectId, svc.Id, serviceStatus)
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

func (s Service) completeDeployment(ctx context.Context, projectId, deploymentId, statusValue, message string) error {
	_, err := s.executionStore.CompleteDeployment(ctx, projectId, deploymentId, statusValue, message)
	return err
}

func (s Service) deploymentExecutionContext(ctx context.Context, projectId string, deploymentId string) (context.Context, context.CancelFunc) {
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
				deployment, err := s.executionStore.Deployment(ctx, projectId, deploymentId)
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

func (s Service) loadDeploymentExecution(ctx context.Context, projectId string, applicationId string, deploymentId string) (model.Application, model.Deployment, error) {
	app, err := s.executionStore.Application(ctx, projectId, applicationId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	deployment, err := s.executionStore.Deployment(ctx, projectId, deploymentId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	return app, deployment, nil
}

func (s Service) resolveServiceFromDeployment(ctx context.Context, projectId, applicationId string, deployment model.Deployment) (model.Service, error) {
	if deployment.ServiceId == nil || *deployment.ServiceId == "" {
		return model.Service{}, fmt.Errorf("deployment %s missing service_id", deployment.Id)
	}
	svc, err := s.executionStore.Service(ctx, projectId, *deployment.ServiceId)
	if err != nil {
		return model.Service{}, err
	}
	if svc.ApplicationId != applicationId {
		return model.Service{}, fmt.Errorf("deployment %s service_id does not belong to application", deployment.Id)
	}
	return svc, nil
}

func (s Service) renderAndDeployWithOptions(ctx context.Context, target environmentport.Target, plan model.EffectiveServicePlan, deploymentId string, forceRecreate bool) error {
	version, svc := plan.Version, plan.Service
	if s.runtime == nil {
		return fmt.Errorf("deployment runtime is not configured")
	}
	serviceDir, err := s.runtime.ServiceDir(target, svc.Code)
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
	composeMountSourceDir, err := s.runtime.ComposeMountSourceDir(ctx, target, svc.Code)
	if err != nil {
		return err
	}
	result, err := s.RenderComposeDetailed(ctx, RenderInput{Plan: plan, LogicalSvcDir: serviceDir, ComposeMountSourceDir: composeMountSourceDir})
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
	if err := s.runtime.StageWorkspace(ctx, target, workspace); err != nil {
		return err
	}
	projectName := composeProjectName(svc.Code)
	command := deployComposeCommand(projectName, deploymentPullPolicy(plan), forceRecreate)
	if _, err := fmt.Fprintln(logWriter, "Starting services"); err != nil {
		return err
	}
	return s.runtime.Run(ctx, target, svc.Code, logWriter, command.Name, command.Args...)
}

func remoteWorkspaceFromRender(serviceCode string, deploymentId string, result RenderResult) (deploymentport.Workspace, error) {
	workspace := deploymentport.Workspace{ServiceCode: serviceCode, DeploymentId: deploymentId, Compose: result.Compose}
	for _, item := range result.ResolvedMounts {
		if !item.ShouldMaterialize {
			continue
		}
		if strings.TrimSpace(item.LogicalSource) == "" {
			return deploymentport.Workspace{}, fmt.Errorf("logical mount source is required for workspace materialization")
		}
		if item.IsFile {
			workspace.Files = append(workspace.Files, deploymentport.WorkspaceFile{
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
