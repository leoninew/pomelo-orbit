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

	version, err := s.executionStore.Version(ctx, *deployment.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
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
	if err := s.ensureSingleRuntime(ctx, app, opts.InstanceKey); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	gateway, err := s.gatewayForDeployment(ctx, app, plan)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan.Gateway = gateway
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, plan, deployment.Id, opts.ForceRecreate); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
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
	_ = restartOpts
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	version, err := s.executionStore.Version(ctx, svc.VersionId)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
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
	gateway, err := s.gatewayForDeployment(ctx, app, plan)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	plan.Gateway = gateway
	if err := verifyDeploymentPlanHash(deployment, plan); err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.renderAndDeployWithOptions(executionCtx, plan, deployment.Id, false); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, app, svc)
			return nil
		}
		_ = s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusFaulted, version.Id)
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.UpdateServiceAfterDeploy(ctx, svc.Id, status.ServiceStatusRunning, version.Id); err != nil {
		return err
	}
	return s.completeDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
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
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		_ = s.completeDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	serviceDir := s.workspace.ServiceDir(svc.Code)
	logPath := s.workspace.DeploymentLogPath(svc.Code, deployment.Id)
	logWriter, err := s.executionLogStore.Writer(logPath)
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
	if err := s.runner.Run(executionCtx, serviceDir, logWriter, command.Name, command.Args...); err != nil {
		if s.deploymentCanceled(ctx, deployment.Id) {
			s.reconcileCanceledService(ctx, app, svc)
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
func (s Service) reconcileCanceledService(ctx context.Context, app model.Application, svc model.Service) {
	serviceStatus := status.ServiceStatusFaulted
	if s.workspace != nil && s.queryRunner != nil {
		exists, err := s.workspace.ServiceDirExists(svc.Code)
		if err == nil && !exists {
			serviceStatus = status.ServiceStatusStopped
		} else if err == nil {
			command := containerPsCommand(composeProjectName(app.Code, svc.InstanceKey))
			output, runErr := s.queryRunner.Run(ctx, s.workspace.ServiceDir(svc.Code), command.Name, command.Args...)
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

func (s Service) renderAndDeployWithOptions(ctx context.Context, plan model.EffectiveServicePlan, deploymentId string, forceRecreate bool) error {
	app, version, svc := plan.Application, plan.Version, plan.Service
	serviceDir := s.workspace.ServiceDir(svc.Code)
	logPath := s.workspace.DeploymentLogPath(svc.Code, deploymentId)
	logWriter, err := s.executionLogStore.Writer(logPath)
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

	physicalDir, err := s.workspace.PhysicalServiceDir(ctx, svc.Code)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(logWriter, "Rendering version %s (%s) with %d component(s) into service %s\n",
		version.Label, version.Id, len(plan.Components), svc.Code); err != nil {
		return err
	}
	result, err := s.RenderComposeDetailed(ctx, RenderInput{Plan: plan, PhysicalSvcDir: physicalDir})
	if err != nil {
		return err
	}
	if logicalMounts := countLogicalMounts(result.ResolvedMounts); logicalMounts > 0 {
		if _, err := fmt.Fprintf(logWriter, "Materializing %d logical mount source(s)\n", logicalMounts); err != nil {
			return err
		}
		if err := MaterializeLogicalMountSources(result.ResolvedMounts); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(logWriter, "Writing deployment configuration"); err != nil {
		return err
	}
	if err := s.workspace.WriteConfig(svc.Code, "docker-compose.yml", result.Compose); err != nil {
		return err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
	command := deployComposeCommand(projectName, deploymentPullPolicy(plan), forceRecreate)
	if _, err := fmt.Fprintln(logWriter, "Starting services"); err != nil {
		return err
	}
	if err := s.runner.Run(ctx, serviceDir, logWriter, command.Name, command.Args...); err != nil {
		return err
	}
	return nil
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
		if item.SourceType == mountSourceDirectory || item.SourceType == mountSourceFile || item.SourceType == mountSourceControlledFile {
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
