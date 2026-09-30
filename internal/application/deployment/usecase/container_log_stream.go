package deploymentsvc

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	logstream "github.com/leoninew/pomelo-orbit/internal/application/logstream/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type containerCursor struct {
	Scope  string `json:"s"`
	Source string `json:"c"`
	Time   string `json:"t"`
}

func (s Service) OpenApplicationLogStream(ctx context.Context, userId, projectId, appId, serviceId, component, cursor string) (logstream.Subscription, error) {
	if _, err := s.loadApplicationForUser(ctx, userId, projectId, appId); err != nil {
		return logstream.Subscription{}, err
	}
	service, err := s.resolveServiceTarget(ctx, projectId, appId, deploymentdto.ServiceTargetInput{ServiceId: serviceId})
	if err != nil {
		return logstream.Subscription{}, err
	}
	if err := s.validateLogComponent(ctx, projectId, service, component); err != nil {
		return logstream.Subscription{}, err
	}
	return s.openContainerLogStream(ctx, userId, projectId, service, component, cursor, time.Time{}, nil)
}

func (s Service) OpenDeploymentContainerLogStream(ctx context.Context, userId, projectId, deploymentId, cursor string) (logstream.Subscription, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
	if err != nil {
		return logstream.Subscription{}, err
	}
	if deployment.OperationType == "stop" {
		return logstream.Subscription{}, apperror.New(apperror.KindValidation, "Container logs are unavailable for stop operations")
	}
	if deployment.ApplicationId == nil {
		return logstream.Subscription{}, apperror.New(apperror.KindNotFound, "Deployment application not found")
	}
	service, err := s.resolveServiceFromDeployment(ctx, projectId, *deployment.ApplicationId, deployment)
	if err != nil {
		return logstream.Subscription{}, apperror.New(apperror.KindNotFound, "Deployment service not found")
	}
	since := deployment.StartedAt
	return s.openContainerLogStream(ctx, userId, projectId, service, "", cursor, since, &deployment)
}

func (s Service) validateLogComponent(ctx context.Context, projectId string, service model.Service, component string) error {
	if component == "" {
		return nil
	}
	components, err := s.application.VersionComponentsByVersion(ctx, projectId, service.VersionId)
	if err != nil {
		return err
	}
	for _, item := range components {
		if item.Name == component {
			return nil
		}
	}
	return apperror.New(apperror.KindNotFound, "Log component not found")
}

func sameLogTarget(a, b model.Environment) bool {
	if a.Id != b.Id || a.TargetType != b.TargetType || a.TargetRevision != b.TargetRevision || a.WorkspaceRoot != b.WorkspaceRoot {
		return false
	}
	if a.IsLocal() {
		return true
	}
	return a.IsSSH() && b.IsSSH() && a.SSH.CredentialId == b.SSH.CredentialId && a.SSH.CredentialRevision == b.SSH.CredentialRevision
}

func (s Service) logContainerSource(ctx context.Context, target environmentport.Target, service model.Service, component, scope string) (string, error) {
	args := []string{"ps", "--all", "--filter", "label=com.docker.compose.project=" + service.Code, "--format", "{{json .}}"}
	if component != "" {
		args = append(args, "--filter", "label=com.docker.compose.service="+component)
	}
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	output, err := s.runtime.QueryAtEnvironmentRoot(queryCtx, target, "docker", args...)
	if err != nil {
		return "", apperror.Wrap(apperror.KindUnavailable, "Failed to inspect log containers", err)
	}
	containers, err := parseComposePsOutput(output)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Invalid container observation", err)
	}
	ids := make([]string, 0, len(containers))
	for _, container := range containers {
		ids = append(ids, container.Id)
	}
	if len(ids) == 0 {
		return "", nil
	}
	sort.Strings(ids)
	return logstream.SourceId(scope + ":" + strings.Join(ids, ",")), nil
}

func (s Service) openContainerLogStream(ctx context.Context, userId, projectId string, service model.Service, component, cursor string, since time.Time, deployment *model.Deployment) (logstream.Subscription, error) {
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return logstream.Subscription{}, err
	}
	if deployment != nil {
		options, err := parseDeployOptions(deployment.OptionsJSON)
		if err != nil {
			return logstream.Subscription{}, err
		}
		if err := verifyDeploymentTargetSnapshot(*deployment, options, target); err != nil {
			return logstream.Subscription{}, apperror.New(apperror.KindConflict, "Deployment log target changed")
		}
	}
	deploymentId := ""
	if deployment != nil {
		deploymentId = deployment.Id
	}
	var credentialRevision int64
	if target.Environment.IsSSH() {
		credentialRevision = target.Environment.SSH.CredentialRevision
	}
	scope := logstream.SourceId(fmt.Sprintf("container:%s:%s:%s:%s:%s:%d:%d:%s", projectId, service.ApplicationId, service.Id, component, target.Environment.Id, target.Environment.TargetRevision, credentialRevision, deploymentId))
	resume := containerCursor{Scope: scope}
	if cursor != "" {
		data, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(data) > 1024 || json.Unmarshal(data, &resume) != nil {
			return logstream.Subscription{}, apperror.New(apperror.KindValidation, "Invalid container log cursor")
		}
		if resume.Scope != scope {
			return logstream.Subscription{}, apperror.New(apperror.KindConflict, "Container log target changed")
		}
		if _, err := time.Parse(time.RFC3339Nano, resume.Time); err != nil {
			return logstream.Subscription{}, apperror.New(apperror.KindValidation, "Invalid container log timestamp")
		}
	}
	source, err := s.logContainerSource(ctx, target, service, component, scope)
	if err != nil {
		return logstream.Subscription{}, err
	}
	initialSource := source
	if initialSource == "" {
		initialSource = scope
	}
	return logstream.Subscription{SourceId: initialSource, Close: func() error { return nil }, Run: func(ctx context.Context, emit logdto.Emit) error {
		currentSource := resume.Source
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
				return err
			}
			if deployment != nil {
				if _, err := s.loadDeploymentForUser(ctx, userId, projectId, deployment.Id); err != nil {
					return err
				}
			}
			currentService, err := s.resolveServiceTarget(ctx, projectId, service.ApplicationId, deploymentdto.ServiceTargetInput{ServiceId: service.Id})
			if err != nil {
				return err
			}
			if currentService.Code != service.Code {
				return apperror.New(apperror.KindConflict, "Container log service changed")
			}
			if err := s.validateLogComponent(ctx, projectId, currentService, component); err != nil {
				return err
			}
			currentTarget, err := s.resolveProjectTarget(ctx, projectId)
			if err != nil {
				return err
			}
			if !sameLogTarget(target.Environment, currentTarget.Environment) {
				return apperror.New(apperror.KindConflict, "Container log target changed")
			}
			source, err = s.logContainerSource(ctx, target, service, component, scope)
			if err != nil {
				return err
			}
			if source == "" {
				if err := emit(logdto.Event{Type: "waiting", SourceId: initialSource}); err != nil {
					return err
				}
				if err := logstream.Wait(ctx, logstream.RecheckInterval); err != nil {
					return err
				}
				continue
			}
			if currentSource != "" && source != currentSource {
				if err := emit(logdto.Event{Type: "source_changed", SourceId: source}); err != nil {
					return err
				}
				if err := emit(logdto.Event{Type: "gap", SourceId: source, Message: "Container source changed; removed logs cannot be recovered"}); err != nil {
					return err
				}
			} else if currentSource == "" && source != initialSource {
				if err := emit(logdto.Event{Type: "source_changed", SourceId: source}); err != nil {
					return err
				}
			}
			currentSource = source
			resume.Source = source
			command := composeCommand{Name: "docker", Args: []string{"compose", "-p", service.Code, "-f", "docker-compose.yml", "logs", "--follow", "--timestamps", "--no-color"}}
			if resume.Time != "" {
				lastTime, _ := time.Parse(time.RFC3339Nano, resume.Time)
				command.Args = append(command.Args, "--since", lastTime.Add(-2*time.Second).UTC().Format(time.RFC3339Nano))
			} else if !since.IsZero() {
				command.Args = append(command.Args, "--since", since.UTC().Format(time.RFC3339Nano))
			} else {
				command.Args = append(command.Args, "--tail", "200")
			}
			if component != "" {
				command.Args = append(command.Args, component)
			}
			if err := emit(logdto.Event{Type: "ready", SourceId: source}); err != nil {
				return err
			}
			check := func(ctx context.Context) error {
				if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
					return err
				}
				if deployment != nil {
					if _, err := s.loadDeploymentForUser(ctx, userId, projectId, deployment.Id); err != nil {
						return err
					}
				}
				current, err := s.resolveServiceTarget(ctx, projectId, service.ApplicationId, deploymentdto.ServiceTargetInput{ServiceId: service.Id})
				if err != nil {
					return err
				}
				if current.Code != service.Code {
					return apperror.New(apperror.KindConflict, "Container log service changed")
				}
				return s.validateLogComponent(ctx, projectId, current, component)
			}
			changed, err := s.followContainerCommand(ctx, target, service, component, scope, source, command, &resume, emit, check)
			if err != nil {
				return err
			}
			if !changed {
				if err := logstream.Wait(ctx, logstream.RecheckInterval); err != nil {
					return err
				}
			}
		}
	}}, nil
}

func (s Service) followContainerCommand(ctx context.Context, target environmentport.Target, service model.Service, component, scope, source string, command composeCommand, resume *containerCursor, emit logdto.Emit, check func(context.Context) error) (bool, error) {
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	reader, writer := io.Pipe()
	defer func() { _ = reader.Close() }()
	done := make(chan error, 1)
	go func() {
		err := s.runtime.Stream(commandCtx, target, service.Code, writer, command.Name, command.Args...)
		_ = writer.CloseWithError(err)
		done <- err
	}()
	lines := make(chan []byte, 1)
	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, logstream.ChunkSize), 1024*1024)
		for scanner.Scan() {
			line := append(append([]byte(nil), scanner.Bytes()...), '\n')
			select {
			case lines <- line:
			case <-commandCtx.Done():
				scanDone <- commandCtx.Err()
				return
			}
		}
		scanDone <- scanner.Err()
		close(lines)
	}()
	defer func() { cancel(); _ = reader.Close(); <-done }()
	ticker := time.NewTicker(logstream.RecheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case line, ok := <-lines:
			if !ok {
				err := <-scanDone
				if errors.Is(err, deploymentport.ErrLogNotReady) {
					return false, emit(logdto.Event{Type: "waiting", SourceId: source})
				}
				if err != nil && !errors.Is(err, context.Canceled) {
					return false, apperror.Wrap(apperror.KindUnavailable, "Container log read failed", err)
				}
				return false, nil
			}
			for _, field := range strings.Fields(string(line)) {
				if timestamp, err := time.Parse(time.RFC3339Nano, field); err == nil {
					last, _ := time.Parse(time.RFC3339Nano, resume.Time)
					if timestamp.After(last) {
						resume.Time = timestamp.UTC().Format(time.RFC3339Nano)
					}
					break
				}
			}
			var cursor string
			if resume.Time != "" {
				data, _ := json.Marshal(resume)
				cursor = base64.RawURLEncoding.EncodeToString(data)
			}
			if err := emit(logdto.Event{Type: "chunk", SourceId: source, Cursor: cursor, Data: line}); err != nil {
				return false, err
			}
		case <-ticker.C:
			if err := check(ctx); err != nil {
				return false, err
			}
			current, err := s.resolveProjectTarget(ctx, service.ProjectId)
			if err != nil {
				return false, err
			}
			if !sameLogTarget(target.Environment, current.Environment) {
				return false, apperror.New(apperror.KindConflict, "Container log target changed")
			}
			observed, err := s.logContainerSource(ctx, target, service, component, scope)
			if err != nil {
				return false, err
			}
			if observed != source {
				return true, nil
			}
		}
	}
}
