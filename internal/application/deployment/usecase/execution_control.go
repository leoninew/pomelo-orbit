package deploymentsvc

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var (
	errDeploymentSuperseded = errors.New("deployment superseded by a newer Service operation")
	errDeploymentCanceled   = errors.New("deployment canceled")
	errDeploymentSkipped    = errors.New("deployment no longer executable")
)

type deploymentBody func(context.Context, model.Application, model.Deployment) error

type deploymentStageKey struct{}

func setDeploymentStage(ctx context.Context, name string) {
	if stage, ok := ctx.Value(deploymentStageKey{}).(*atomic.Value); ok {
		stage.Store(name)
	}
}

type deploymentSupersededError struct{ currentDeploymentId string }

func (e *deploymentSupersededError) Error() string {
	return fmt.Sprintf("%s (current_deployment_id=%s)", errDeploymentSuperseded, e.currentDeploymentId)
}

func (*deploymentSupersededError) Unwrap() error { return errDeploymentSuperseded }

func (s Service) executeDeployment(ctx context.Context, projectId, applicationId, deploymentId, operation string, body deploymentBody) error {
	started := time.Now()
	if s.executionTimeout <= 0 || s.cancelTimeout <= 0 {
		return errors.New("deployment execution and cancellation timeouts must be configured")
	}
	timeoutErr := fmt.Errorf("deployment execution exceeded %s: %w", s.executionTimeout, context.DeadlineExceeded)
	timeoutCtx, stopTimeout := context.WithTimeoutCause(ctx, s.executionTimeout, timeoutErr)
	defer stopTimeout()
	executionCtx, cancel := context.WithCancelCause(timeoutCtx)
	defer cancel(nil)
	stage := &atomic.Value{}
	stage.Store("load_execution")
	executionCtx = context.WithValue(executionCtx, deploymentStageKey{}, stage)
	identity := make(chan string, 1)
	done := make(chan error, 1)
	// The body owns runtime resources; only the controller below commits results.
	go func() {
		app, deployment, err := s.loadDeploymentExecution(executionCtx, projectId, applicationId, deploymentId)
		if err != nil {
			done <- err
			return
		}
		if deployment.ServiceId != nil {
			identity <- *deployment.ServiceId
			go s.monitorDeployment(executionCtx, cancel, projectId, *deployment.ServiceId, deploymentId)
		}
		setDeploymentStage(executionCtx, "begin_execution")
		begun, err := s.executionStore.BeginDeployment(executionCtx, projectId, deploymentId)
		if err != nil {
			done <- err
			return
		}
		if !begun {
			done <- errDeploymentSkipped
			return
		}
		if deployment.ServiceId == nil {
			done <- fmt.Errorf("deployment %s missing service_id", deploymentId)
			return
		}
		setDeploymentStage(executionCtx, "cancel_previous_operations")
		s.cancelSupersededDeployments(executionCtx, projectId, *deployment.ServiceId, deploymentId)
		if err := s.ensureDeploymentCurrent(executionCtx, projectId, deploymentId); err != nil {
			done <- err
			return
		}
		setDeploymentStage(executionCtx, "prepare_operation")
		done <- body(executionCtx, app, deployment)
	}()

	var executionErr error
	select {
	case executionErr = <-done:
		if executionCtx.Err() != nil {
			executionErr = context.Cause(executionCtx)
		}
	case <-executionCtx.Done():
		executionErr = context.Cause(executionCtx)
		select {
		case <-done:
		default:
			serviceId := ""
			select {
			case serviceId = <-identity:
			default:
			}
			err := fmt.Errorf("deployment stage %s: %w", stage.Load(), executionErr)
			s.warnExecution(projectId, serviceId, deploymentId, "runtime_exit_unconfirmed", err, time.Since(started))
		}
	}
	if executionErr != nil {
		executionErr = fmt.Errorf("deployment stage %s: %w", stage.Load(), executionErr)
	}
	cancel(executionErr)
	return s.finishDeploymentBounded(ctx, projectId, deploymentId, operation, executionErr)
}

func (s Service) monitorDeployment(ctx context.Context, cancel context.CancelCauseFunc, projectId, serviceId, deploymentId string) {
	interval := s.pollInterval
	if interval <= 0 || interval > time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	warned := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			started := time.Now()
			checkCtx, stop := context.WithTimeout(ctx, s.cancelTimeout)
			err := s.ensureDeploymentCurrent(checkCtx, projectId, deploymentId)
			stop()
			if err == nil {
				warned = false
				continue
			}
			if isDeploymentCancellation(err) || errors.Is(err, repository.ErrNotFound) {
				cancel(err)
				return
			}
			if ctx.Err() == nil && !warned {
				s.warnExecution(projectId, serviceId, deploymentId, "cancellation_check", err, time.Since(started))
				warned = true
			}
		}
	}
}

func (s Service) ensureDeploymentCurrent(ctx context.Context, projectId, deploymentId string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	deployment, err := s.executionStore.Deployment(ctx, projectId, deploymentId)
	if err != nil {
		return err
	}
	if deployment.ServiceId == nil {
		return fmt.Errorf("deployment %s missing service_id", deploymentId)
	}
	svc, err := s.executionStore.Service(ctx, projectId, *deployment.ServiceId)
	if err != nil {
		return err
	}
	if svc.CurrentDeploymentId == nil || *svc.CurrentDeploymentId != deploymentId {
		currentId := ""
		if svc.CurrentDeploymentId != nil {
			currentId = *svc.CurrentDeploymentId
		}
		return &deploymentSupersededError{currentDeploymentId: currentId}
	}
	if deployment.Status == status.WorkStatusCanceled {
		return errDeploymentCanceled
	}
	if deployment.Status != status.WorkStatusRunning && deployment.Status != status.WorkStatusWaitingToRun {
		return errDeploymentSkipped
	}
	return ctx.Err()
}

func (s Service) cancelSupersededDeployments(ctx context.Context, projectId, serviceId, deploymentId string) {
	started := time.Now()
	cancelCtx, stop := context.WithTimeout(ctx, s.cancelTimeout)
	defer stop()
	type cancellationResult struct {
		ids []string
		err error
	}
	done := make(chan cancellationResult, 1)
	go func() {
		ids, err := s.executionStore.CancelSupersededDeployments(cancelCtx, projectId, serviceId, deploymentId)
		done <- cancellationResult{ids: ids, err: err}
	}()
	var ids []string
	var err error
	select {
	case result := <-done:
		ids, err = result.ids, result.err
	case <-cancelCtx.Done():
		err = cancelCtx.Err()
	}
	if err != nil {
		s.warnExecution(projectId, serviceId, deploymentId, "cancel_previous_operations", err, time.Since(started))
	}
	if len(ids) > 0 && s.logger != nil {
		s.logger.Info("previous Service operations marked canceled", "project_id", projectId, "service_id", serviceId, "deployment_id", deploymentId, "superseded_deployment_ids", ids, "duration_ms", time.Since(started).Milliseconds())
	}
}

func isDeploymentCancellation(err error) bool {
	return errors.Is(err, errDeploymentSuperseded) || errors.Is(err, errDeploymentCanceled) || errors.Is(err, errDeploymentSkipped)
}

func (s Service) finishDeploymentBounded(ctx context.Context, projectId, deploymentId, operation string, executionErr error) error {
	cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), s.cancelTimeout)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- s.finishDeployment(cleanupCtx, projectId, deploymentId, operation, executionErr) }()
	select {
	case err := <-done:
		return err
	case <-cleanupCtx.Done():
		s.warnExecution(projectId, "", deploymentId, "execution_cleanup_timeout", cleanupCtx.Err())
		if isDeploymentCancellation(executionErr) || errors.Is(executionErr, repository.ErrNotFound) {
			return nil
		}
		return errors.Join(executionErr, cleanupCtx.Err())
	}
}

func (s Service) finishDeployment(ctx context.Context, projectId, deploymentId, operation string, executionErr error) error {
	deployment, err := s.executionStore.Deployment(ctx, projectId, deploymentId)
	if errors.Is(err, repository.ErrNotFound) {
		s.warnExecution(projectId, "", deploymentId, "execution_record_missing", err)
		return nil
	}
	if err != nil {
		if isDeploymentCancellation(executionErr) {
			s.warnExecution(projectId, "", deploymentId, "cancel_record_read", err)
			return nil
		}
		return errors.Join(executionErr, err)
	}
	if deployment.ServiceId == nil {
		return errors.Join(executionErr, errors.New("deployment missing service_id"))
	}
	serviceId := *deployment.ServiceId
	svc, err := s.executionStore.Service(ctx, projectId, serviceId)
	if err != nil {
		if isDeploymentCancellation(executionErr) || errors.Is(err, repository.ErrNotFound) {
			s.warnExecution(projectId, serviceId, deploymentId, "cancel_service_read", err)
			return nil
		}
		return errors.Join(executionErr, err)
	}
	if svc.CurrentDeploymentId == nil || *svc.CurrentDeploymentId != deploymentId {
		if _, err := s.executionStore.CancelObsoleteDeployment(ctx, projectId, deploymentId); err != nil {
			s.warnExecution(projectId, serviceId, deploymentId, "cancel_obsolete_operation", err)
		}
		return nil
	}
	if deployment.Status == status.WorkStatusCanceled {
		target, err := s.resolveProjectTarget(ctx, projectId)
		if err != nil {
			s.warnExecution(projectId, serviceId, deploymentId, "cancel_target_resolution", err)
			return nil
		}
		opts, err := parseDeployOptions(deployment.OptionsJSON)
		if err != nil {
			s.warnExecution(projectId, serviceId, deploymentId, "cancel_options", err)
			return nil
		}
		if err := verifyDeploymentTargetSnapshot(deployment, opts, target); err != nil {
			s.warnExecution(projectId, serviceId, deploymentId, "cancel_target_changed", err)
			return nil
		}
		s.reconcileCanceledService(ctx, projectId, deploymentId, target, model.Application{}, svc)
		return nil
	}
	if isDeploymentCancellation(executionErr) || status.WorkStatusIsComplete(deployment.Status) {
		return nil
	}
	serviceState, deploymentState, message := status.ServiceStatusRunning, status.WorkStatusRanToCompletion, ""
	versionId := deployment.VersionId
	if operation == "stop" {
		serviceState, versionId = status.ServiceStatusStopped, nil
	}
	if executionErr != nil {
		serviceState, deploymentState, message = status.ServiceStatusFaulted, status.WorkStatusFaulted, executionErr.Error()
	}
	if s.transactionRunner == nil {
		return errors.Join(executionErr, errors.New("deployment transaction runner is not configured"))
	}
	err = s.transactionRunner.RunInTransaction(ctx, func(ctx context.Context) error {
		changed, err := s.executionStore.UpdateServiceDeploymentResult(ctx, projectId, serviceId, deploymentId, serviceState, versionId)
		if err != nil {
			return err
		}
		if !changed {
			return errDeploymentSkipped
		}
		changed, err = s.executionStore.CompleteDeployment(ctx, projectId, deploymentId, deploymentState, message)
		if err != nil {
			return err
		}
		if !changed {
			return errDeploymentSkipped
		}
		return nil
	})
	if errors.Is(err, errDeploymentSkipped) {
		return nil
	}
	return errors.Join(executionErr, err)
}

func (s Service) warnExecution(projectId, serviceId, deploymentId, stage string, err error, elapsed ...time.Duration) {
	if s.logger != nil {
		fields := []any{"project_id", projectId, "service_id", serviceId, "deployment_id", deploymentId, "stage", stage, "execution_timeout", s.executionTimeout, "cancel_timeout", s.cancelTimeout, "cause", err}
		if len(elapsed) > 0 {
			fields = append(fields, "duration_ms", elapsed[0].Milliseconds())
		}
		var superseded *deploymentSupersededError
		if errors.As(err, &superseded) {
			fields = append(fields, "current_deployment_id", superseded.currentDeploymentId)
		}
		s.logger.Warn("Service operation cancellation or cleanup incomplete", fields...)
	}
}
