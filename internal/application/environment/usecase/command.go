package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const commandOutputLimit = 1024 * 1024

var errOutputLimit = errors.New("output_limit")
var errTargetChanged = errors.New("access_or_target_changed")

type CommandService struct {
	access *AccessService
	runner environmentport.CommandRunner
	mu     sync.Mutex
	active map[string]int
	total  int
}

func NewCommandService(access *AccessService, runner environmentport.CommandRunner) *CommandService {
	return &CommandService{access: access, runner: runner, active: make(map[string]int)}
}

func validateCommand(input environmentdto.CommandInput) (time.Duration, error) {
	timeout := uint32(60)
	if input.TimeoutSeconds != nil {
		timeout = *input.TimeoutSeconds
	}
	if timeout < 1 || timeout > 300 {
		return 0, apperror.New(apperror.KindValidation, "timeout_seconds must be 1–300")
	}
	size := 0
	for _, arg := range input.Argv {
		size += len(arg)
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return 0, apperror.New(apperror.KindValidation, "argv must contain UTF-8 without NUL")
		}
	}
	if len(input.Argv) == 0 || len(input.Argv) > 128 || input.Argv[0] == "" || size > 32*1024 {
		return 0, apperror.New(apperror.KindValidation, "argv requires a program, at most 128 arguments and 32 KiB")
	}
	if len(input.Stdin) > 64*1024 {
		return 0, apperror.New(apperror.KindValidation, "stdin exceeds 64 KiB")
	}
	return time.Duration(timeout) * time.Second, nil
}

func validateCommandWorkdir(value, platform string) error {
	if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
		return apperror.New(apperror.KindValidation, "Invalid workdir")
	}
	home := value == "~" || strings.HasPrefix(value, "~/")
	absolute := strings.HasPrefix(value, "/")
	if platform == model.EnvironmentPlatformWindows {
		absolute = windowsWorkspacePattern.MatchString(strings.ReplaceAll(value, "/", "\\")) || strings.HasPrefix(value, `\\`)
	}
	if !home && !absolute {
		return apperror.New(apperror.KindValidation, "workdir must be absolute or ~/...")
	}
	return nil
}

func (s *CommandService) acquire(actor string) (func(), error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.total >= 16 || s.active[actor] >= 2 {
		return nil, apperror.New(apperror.KindUnavailable, "Command session capacity reached")
	}
	s.total++
	s.active[actor]++
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.total--
		s.active[actor]--
		if s.active[actor] == 0 {
			delete(s.active, actor)
		}
	}, nil
}

func (s *CommandService) Execute(ctx context.Context, actorId string, input environmentdto.CommandInput) (environmentdto.CommandResult, error) {
	started := time.Now()
	timeout, err := validateCommand(input)
	if err != nil {
		return environmentdto.CommandResult{}, err
	}
	ctx, deadlineCancel := context.WithTimeout(ctx, timeout)
	defer deadlineCancel()
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	p, target, err := s.access.target(ctx, actorId, input.ProjectCode)
	if err != nil {
		return environmentdto.CommandResult{}, err
	}
	if input.Workdir == "" {
		input.Workdir = target.Environment.WorkspaceRoot
	}
	if err := validateCommandWorkdir(input.Workdir, target.Environment.SSH.Platform); err != nil {
		return environmentdto.CommandResult{}, err
	}
	release, err := s.acquire(actorId)
	if err != nil {
		return environmentdto.CommandResult{}, err
	}
	defer release()
	connectCtx, connectCancel := context.WithTimeout(ctx, TerminalConnectLimit)
	session, err := s.runner.Open(connectCtx, target)
	connectCancel()
	if err != nil {
		return environmentdto.CommandResult{}, apperror.New(apperror.KindUnavailable, "SSH connection or authentication failed")
	}
	// This owner starts Close once, with one shared five-second cleanup budget.
	var closeOnce sync.Once
	closeDone := make(chan struct{})
	var cleanupDeadline time.Time
	closeSession := func() {
		closeOnce.Do(func() {
			cleanupDeadline = time.Now().Add(5 * time.Second)
			go func() { _ = session.Close(); close(closeDone) }()
		})
	}
	defer func() {
		closeSession()
		timer := time.NewTimer(max(time.Until(cleanupDeadline), 0))
		defer timer.Stop()
		select {
		case <-closeDone:
		case <-timer.C:
		}
	}()
	grant := TerminalGrant{ActorId: actorId, ProjectId: p.Id, Target: target}
	grant, err = s.access.environments.confirmConnection(ctx, grant, session.HostKeyFingerprint())
	if err != nil {
		return environmentdto.CommandResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return environmentdto.CommandResult{}, apperror.New(apperror.KindUnavailable, "Command setup timed out or was cancelled")
	}
	output := &commandOutput{cancel: cancel}
	type completion struct {
		outcome environmentport.CommandOutcome
		err     error
	}
	done := make(chan completion, 1)
	go func() {
		outcome, err := session.Run(ctx, environmentport.Command{Argv: input.Argv, Workdir: input.Workdir, Stdin: input.Stdin}, commandStream{output, false}, commandStream{output, true})
		done <- completion{outcome, err}
	}()
	monitorCtx, stopMonitor := context.WithCancel(ctx)
	defer stopMonitor()
	go s.monitorCommand(monitorCtx, grant, cancel)
	var completed completion
	select {
	case completed = <-done:
	case <-ctx.Done():
		closeSession()
		timer := time.NewTimer(max(time.Until(cleanupDeadline), 0))
		defer timer.Stop()
		select {
		case completed = <-done:
		case <-timer.C:
			completed.outcome.Sent = true
		}
	}
	if !completed.outcome.Sent {
		return environmentdto.CommandResult{}, apperror.New(apperror.KindUnavailable, "SSH command could not be started")
	}
	result := environmentdto.CommandResult{ProjectId: p.Id, ProjectCode: p.Code, EnvironmentId: target.Environment.Id, TargetRevision: target.Environment.TargetRevision, State: "transport_lost", DurationMs: time.Since(started).Milliseconds()}
	result.Stdout, result.Stderr, result.OutputTruncated = output.snapshot()
	switch {
	case errors.Is(context.Cause(ctx), errOutputLimit):
		result.State = "output_limit"
	case errors.Is(context.Cause(ctx), errTargetChanged):
		result.State = "access_or_target_changed"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.State = "timed_out"
	case ctx.Err() != nil:
	case completed.err == nil && completed.outcome.ExitCode != nil:
		result.State = "exited"
		result.ExitCode = completed.outcome.ExitCode
	}
	if logger := s.access.environments.logger; logger != nil {
		logger.Info("environment command completed", "actor_id", actorId, "project_id", p.Id, "environment_id", result.EnvironmentId, "target_revision", result.TargetRevision, "credential_revision", target.Environment.SSH.CredentialRevision, "duration_ms", result.DurationMs, "state", result.State, "exit_code", result.ExitCode)
	}
	return result, nil
}

func (s *CommandService) monitorCommand(ctx context.Context, grant TerminalGrant, cancel context.CancelCauseFunc) {
	ticker := time.NewTicker(terminalCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			checkCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			err := s.access.environments.validateConnection(checkCtx, grant)
			stop()
			if err != nil && ctx.Err() == nil {
				cancel(errTargetChanged)
				return
			}
		}
	}
}

type commandOutput struct {
	mu             sync.Mutex
	stdout, stderr []byte
	truncated      bool
	cancel         context.CancelCauseFunc
}
type commandStream struct {
	output *commandOutput
	stderr bool
}

func (w commandStream) Write(data []byte) (int, error) {
	o := w.output
	o.mu.Lock()
	defer o.mu.Unlock()
	remaining := commandOutputLimit - len(o.stdout) - len(o.stderr)
	n := min(len(data), remaining)
	if w.stderr {
		o.stderr = append(o.stderr, data[:n]...)
	} else {
		o.stdout = append(o.stdout, data[:n]...)
	}
	if n < len(data) {
		o.truncated = true
		o.cancel(errOutputLimit)
		return n, errOutputLimit
	}
	return n, nil
}
func (o *commandOutput) snapshot() ([]byte, []byte, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]byte(nil), o.stdout...), append([]byte(nil), o.stderr...), o.truncated
}
