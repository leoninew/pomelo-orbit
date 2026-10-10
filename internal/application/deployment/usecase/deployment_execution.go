package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, projectId, applicationId, deploymentId string, forceRecreate bool) error {
	return s.executeDeployment(ctx, projectId, applicationId, deploymentId, "deploy", func(ctx context.Context, app model.Application, deployment model.Deployment) error {
		return s.executeUp(ctx, projectId, app, deployment, forceRecreate)
	})
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, projectId, applicationId, deploymentId string) error {
	return s.executeDeployment(ctx, projectId, applicationId, deploymentId, "restart", func(ctx context.Context, app model.Application, deployment model.Deployment) error {
		return s.executeUp(ctx, projectId, app, deployment, false)
	})
}

func (s Service) ExecuteApplicationStop(ctx context.Context, projectId, applicationId, deploymentId string, removeVolumes bool) error {
	return s.executeDeployment(ctx, projectId, applicationId, deploymentId, "stop", func(ctx context.Context, app model.Application, deployment model.Deployment) error {
		return s.executeStop(ctx, projectId, app, deployment, removeVolumes)
	})
}

func (s Service) executeUp(ctx context.Context, projectId string, app model.Application, deployment model.Deployment, forceRecreate bool) error {
	setDeploymentStage(ctx, "resolve_plan")
	if deployment.VersionId == nil || *deployment.VersionId == "" {
		return fmt.Errorf("deployment %s missing version_id", deployment.Id)
	}
	svc, err := s.resolveServiceFromDeployment(ctx, projectId, app.Id, deployment)
	if err != nil {
		return err
	}
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		return fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
	}
	opts.ForceRecreate = opts.ForceRecreate || forceRecreate
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, opts, target); err != nil {
		return err
	}
	version, err := s.executionStore.Version(ctx, projectId, *deployment.VersionId)
	if err != nil {
		return err
	}
	svc.VersionId = version.Id
	components, err := s.executionStore.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return err
	}
	overlays, err := s.executionStore.ServiceComponentsByService(ctx, projectId, svc.Id)
	if err != nil {
		return err
	}
	env, err := s.executionStore.ServiceEnvByService(ctx, projectId, svc.Id)
	if err != nil {
		return err
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, svc, components, overlays, env, nil)
	if err != nil {
		return err
	}
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	plan.Gateway = cloneGatewayConfig(opts.GatewayConfig)
	setPlanJoinTraefikNetwork(&plan, opts.JoinTraefikNetwork)
	if requiresGatewayConfig(plan) && plan.Gateway == nil {
		return fmt.Errorf("deployment %s is missing Gateway configuration snapshot", deployment.Id)
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		return err
	}
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		return err
	}
	if err := s.renderAndDeployWithOptions(ctx, target, plan, deployment.Id, opts.ForceRecreate); err != nil {
		return err
	}
	if err := s.ensureDeploymentCurrent(ctx, projectId, deployment.Id); err != nil {
		return err
	}
	setDeploymentStage(ctx, "gateway_readiness")
	return s.waitGatewayReady(ctx, projectId, plan)
}

func (s Service) waitGatewayReady(ctx context.Context, projectId string, plan model.EffectiveServicePlan) error {
	if !isGatewayCarrier(plan) {
		return nil
	}
	if s.gatewayReadiness == nil {
		return fmt.Errorf("gateway readiness checker is not configured")
	}
	timeout := time.Duration(plan.Gateway.RestReadyTimeoutSeconds) * time.Second
	return s.gatewayReadiness.WaitUntilReady(ctx, projectId, *plan.Gateway, timeout)
}

func (s Service) executeStop(ctx context.Context, projectId string, app model.Application, deployment model.Deployment, removeVolumes bool) error {
	setDeploymentStage(ctx, "resolve_stop_workspace")
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		return fmt.Errorf("deployment %s has invalid options: %w", deployment.Id, err)
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return err
	}
	if err := verifyDeploymentTargetSnapshot(deployment, opts, target); err != nil {
		return err
	}
	svc, err := s.resolveServiceFromDeployment(ctx, projectId, app.Id, deployment)
	if err != nil {
		return err
	}
	serviceDir, err := s.runtime.ResolveDirectory(ctx, target, plannedServiceLocation(svc))
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
	removeVolumes = removeVolumes || opts.RemoveVolumes
	if removeVolumes {
		if _, err := fmt.Fprintln(logWriter, "Stopping services and removing volumes"); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintln(logWriter, "Stopping services"); err != nil {
		return err
	}
	if err := s.ensureDeploymentCurrent(ctx, projectId, deployment.Id); err != nil {
		return err
	}
	command := stopComposeCommand(composeProjectName(svc.Code), removeVolumes)
	setDeploymentStage(ctx, "compose_stop")
	return s.runtime.Run(ctx, target, deploymentport.ServiceLocation{Code: svc.Code, Directory: serviceDir}, logWriter, command.Name, command.Args...)
}

func (s Service) reconcileCanceledService(ctx context.Context, projectId, deploymentId string, target environmentport.Target, app model.Application, svc model.Service) {
	current, err := s.executionStore.Service(ctx, projectId, svc.Id)
	if err != nil || current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != deploymentId {
		return
	}
	location, err := lifecycleServiceLocation(target, current)
	if err != nil {
		return
	}
	command := containerPsCommand(composeProjectName(svc.Code))
	output, err := s.runtime.Query(ctx, target, location, command.Name, command.Args...)
	if err != nil {
		s.warnExecution(projectId, svc.Id, deploymentId, "cancel_observation", err)
		return
	}
	containers, err := parseComposePsOutput(output)
	if err != nil {
		s.warnExecution(projectId, svc.Id, deploymentId, "cancel_observation", err)
		return
	}
	if _, err := s.executionStore.ReconcileServiceAfterCancellation(ctx, projectId, svc.Id, deploymentId, observedServiceStatus(containers)); err != nil {
		s.warnExecution(projectId, svc.Id, deploymentId, "cancel_observation_write", err)
	}
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
	if deployment.WorkingDirectory == nil || *deployment.WorkingDirectory == "" || deployment.EnvironmentTargetRevision == nil {
		return model.Service{}, fmt.Errorf("deployment %s missing working_directory snapshot", deployment.Id)
	}
	svc.DeploymentDirectory, svc.DirectoryTargetRevision = *deployment.WorkingDirectory, *deployment.EnvironmentTargetRevision
	return svc, nil
}

func (s Service) renderAndDeployWithOptions(ctx context.Context, target environmentport.Target, plan model.EffectiveServicePlan, deploymentId string, forceRecreate bool) error {
	setDeploymentStage(ctx, "resolve_workspace")
	version, svc := plan.Version, plan.Service
	if s.runtime == nil {
		return fmt.Errorf("deployment runtime is not configured")
	}
	if sessions, ok := s.runtime.(deploymentport.RuntimeSessions); ok {
		var closeSession func()
		var err error
		ctx, closeSession, err = sessions.OpenSession(ctx, target)
		if err != nil {
			return err
		}
		defer closeSession()
	}
	serviceDir, err := s.runtime.ResolveDirectory(ctx, target, plannedServiceLocation(svc))
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
	composeMountSourceDir, err := s.runtime.ComposeMountSourceDir(ctx, target, plannedServiceLocation(svc))
	if err != nil {
		return err
	}
	setDeploymentStage(ctx, "render_compose")
	result, err := s.RenderComposeDetailed(ctx, RenderInput{Plan: plan, LogicalSvcDir: serviceDir, ComposeMountSourceDir: composeMountSourceDir, ResolveMountSource: func(ctx context.Context, source string) (string, error) {
		return s.runtime.ComposeMountSourceDir(ctx, target, deploymentport.ServiceLocation{Code: svc.Code, Directory: filepath.ToSlash(source)})
	}})
	if err != nil {
		return err
	}
	workspace, err := remoteWorkspaceFromRender(deploymentport.ServiceLocation{Code: svc.Code, Directory: serviceDir}, deploymentId, result)
	if err != nil {
		return err
	}
	setDeploymentStage(ctx, "check_directory_ownership")
	if err := s.checkDirectoryOwnership(ctx, target, svc, serviceDir, result.ResolvedMounts); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(logWriter, "Staging deployment configuration on project environment"); err != nil {
		return err
	}
	if err := s.ensureDeploymentCurrent(ctx, svc.ProjectId, deploymentId); err != nil {
		return err
	}
	setDeploymentStage(ctx, "stage_workspace")
	if err := s.runtime.StageWorkspace(ctx, target, workspace); err != nil {
		return err
	}
	projectName := composeProjectName(svc.Code)
	// Recreate Gateway containers so replaced static config bind mounts are reloaded.
	command := deployComposeCommand(projectName, deploymentPullPolicy(plan), forceRecreate || isGatewayCarrier(plan))
	if _, err := fmt.Fprintln(logWriter, "Starting services"); err != nil {
		return err
	}
	if err := s.ensureDeploymentCurrent(ctx, svc.ProjectId, deploymentId); err != nil {
		return err
	}
	setDeploymentStage(ctx, "bind_runtime_directory")
	bound, err := s.executionStore.BindServiceRuntimeDirectory(ctx, svc.ProjectId, svc.Id, deploymentId, svc.DeploymentDirectory, target.Environment.TargetRevision)
	if err != nil {
		return err
	}
	if !bound {
		if err := s.ensureDeploymentCurrent(ctx, svc.ProjectId, deploymentId); err != nil {
			return err
		}
		return fmt.Errorf("environment changed before runtime directory binding")
	}
	if err := s.ensureDeploymentCurrent(ctx, svc.ProjectId, deploymentId); err != nil {
		return err
	}
	setDeploymentStage(ctx, "compose_up")
	return s.runtime.Run(ctx, target, deploymentport.ServiceLocation{Code: svc.Code, Directory: serviceDir}, logWriter, command.Name, command.Args...)
}

func remoteWorkspaceFromRender(location deploymentport.ServiceLocation, deploymentId string, result RenderResult) (deploymentport.Workspace, error) {
	workspace := deploymentport.Workspace{Location: location, DeploymentId: deploymentId, Compose: result.Compose}
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
