package environmentsvc

import (
	"context"
	"errors"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

type ForwardService struct {
	access    *AccessService
	runner    environmentport.ForwardRunner
	processes environmentport.ForwardProcesses
}

func NewForwardService(access *AccessService, runner environmentport.ForwardRunner, processes environmentport.ForwardProcesses) *ForwardService {
	return &ForwardService{access: access, runner: runner, processes: processes}
}

func (s *ForwardService) resolve(ctx context.Context, actorId string, input environmentdto.ForwardInput) (environmentdto.ForwardSpec, TerminalGrant, error) {
	if input.LocalPort < 1 || input.LocalPort > 65535 || input.RemotePort < 1 || input.RemotePort > 65535 {
		return environmentdto.ForwardSpec{}, TerminalGrant{}, apperror.New(apperror.KindValidation, "forward ports must be 1–65535")
	}
	project, target, err := s.access.target(ctx, actorId, input.ProjectCode)
	if err != nil {
		return environmentdto.ForwardSpec{}, TerminalGrant{}, err
	}
	spec := environmentdto.ForwardSpec{ForwardInput: input, ActorId: actorId, ProjectId: project.Id,
		EnvironmentId: target.Environment.Id, TargetRevision: target.Environment.TargetRevision,
		CredentialRevision: target.Environment.SSH.CredentialRevision}
	return spec, TerminalGrant{ActorId: actorId, ProjectId: project.Id, Target: target}, nil
}

func (s *ForwardService) StartBackground(ctx context.Context, actorId string, input environmentdto.ForwardInput) (environmentdto.ForwardStatus, error) {
	spec, _, err := s.resolve(ctx, actorId, input)
	if err != nil {
		return environmentdto.ForwardStatus{}, err
	}
	return s.processes.Start(ctx, spec)
}

// Run owns one forward until cancellation, lost SSH, or revoked connection
// access. Forwarding has no terminal ticket, terminal quota, or idle deadline.
func (s *ForwardService) Run(ctx context.Context, actorId string, input environmentdto.ForwardInput, ready func(environmentdto.ForwardSpec) error) error {
	spec, grant, err := s.resolve(ctx, actorId, input)
	if err != nil {
		return err
	}
	connectCtx, stopConnect := context.WithTimeout(ctx, TerminalConnectLimit)
	session, err := s.runner.OpenForward(connectCtx, grant.Target, input.LocalPort, input.RemotePort)
	stopConnect()
	if err != nil {
		return apperror.New(apperror.KindUnavailable, "SSH connection or authentication failed")
	}
	defer func() { _ = session.Close() }()
	grant, err = s.access.environments.confirmConnection(ctx, grant, session.HostKeyFingerprint())
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := session.Listen(); err != nil {
		return err
	}
	if err := ready(spec); err != nil {
		return err
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		s.access.environments.monitorConnection(ctx, grant, cancel)
	}()
	wait := make(chan error, 1)
	go func() { wait <- session.Wait() }()
	select {
	case err = <-wait:
		cancel(nil)
	case <-ctx.Done():
		_ = session.Close()
		<-wait
		err = context.Cause(ctx)
	}
	<-monitorDone
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
