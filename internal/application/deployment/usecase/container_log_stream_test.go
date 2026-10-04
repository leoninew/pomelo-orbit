package deploymentsvc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	logdto "github.com/leoninew/pomelo-orbit/internal/application/logstream/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type containerStreamStore struct {
	*runtimeQueryStore
	repository.DeploymentStore
	mu         sync.Mutex
	deployment model.Deployment
	member     bool
}

func (s *containerStreamStore) Deployment(context.Context, string, string) (model.Deployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deployment, nil
}
func (s *containerStreamStore) IsProjectMember(context.Context, string, string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.member, nil
}
func (s *containerStreamStore) CreateDeployment(ctx context.Context, projectId string, deployment model.Deployment) error {
	return s.runtimeQueryStore.CreateDeployment(ctx, projectId, deployment)
}
func (s *containerStreamStore) HasActiveDeployment(ctx context.Context, projectId, appId string) (bool, error) {
	return s.runtimeQueryStore.HasActiveDeployment(ctx, projectId, appId)
}

type containerStreamRuntime struct {
	deploymentport.Runtime
	mu       sync.Mutex
	id       string
	commands []string
	closed   int
}

type containerStreamApplicationStore struct {
	repository.ApplicationStore
	store *runtimeQueryStore
}

func (s containerStreamApplicationStore) VersionComponentsByVersion(ctx context.Context, projectId, versionId string) ([]model.VersionComponent, error) {
	return s.store.VersionComponentsByVersion(ctx, projectId, versionId)
}

type containerStreamTargetResolver struct {
	target environmentport.Target
}

func (r *containerStreamTargetResolver) ResolveProjectTarget(context.Context, string) (environmentport.Target, error) {
	return r.target, nil
}

func (r *containerStreamRuntime) QueryAtEnvironmentRoot(context.Context, environmentport.Target, string, ...string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.id == "" {
		return "", nil
	}
	return fmt.Sprintf(`{"ID":%q}`, r.id), nil
}
func (r *containerStreamRuntime) StreamAtEnvironmentRoot(ctx context.Context, _ environmentport.Target, output io.Writer, _ string, args ...string) error {
	r.mu.Lock()
	id := r.id
	r.commands = append(r.commands, strings.Join(args, " "))
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.closed++; r.mu.Unlock() }()
	if _, err := fmt.Fprintf(output, "web | 2026-09-30T00:00:00.123456789Z %s\n", id); err != nil {
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func newContainerStreamService() (Service, *containerStreamStore, *containerStreamRuntime) {
	svc, queryStore := newRuntimeQueryService()
	queryStore.service.ProjectId = "project-1"
	appId, serviceId := "app-1", "service-1"
	options := "{}"
	queryStore.service.RuntimeDirectory, queryStore.service.RuntimeTargetRevision = "/custom/demo", 1
	store := &containerStreamStore{runtimeQueryStore: queryStore, member: true, deployment: model.Deployment{WorkingDirectory: &queryStore.service.RuntimeDirectory, Id: "deployment-1", ApplicationId: &appId, ServiceId: &serviceId, Status: status.WorkStatusRunning, StartedAt: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)}}
	target := testSSHTarget("project-1")
	store.deployment.OptionsJSON = &options
	applyDeploymentTargetSnapshot(&store.deployment, target, nil)
	runtime := &containerStreamRuntime{}
	svc.project, svc.commandStore, svc.executionStore, svc.deployment = store, store, store, store
	svc.targetResolver = staticTargetResolver{target: target}
	svc.runtime = runtime
	return svc, store, runtime
}

func TestDeploymentContainerLogsWaitThenFollowRebuildAfterDeploymentCompletes(t *testing.T) {
	svc, store, runtime := newContainerStreamService()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	stream, err := svc.OpenDeploymentContainerLogStream(ctx, "user-1", "project-1", "deployment-1", "")
	if err != nil {
		t.Fatal(err)
	}
	var chunks []string
	var gap bool
	err = stream.Run(ctx, func(event logdto.Event) error {
		switch event.Type {
		case "waiting":
			store.mu.Lock()
			store.deployment.Status = status.WorkStatusRanToCompletion
			store.mu.Unlock()
			runtime.mu.Lock()
			runtime.id = "container-1"
			runtime.mu.Unlock()
		case "gap":
			gap = true
		case "chunk":
			chunks = append(chunks, string(event.Data))
			if len(chunks) == 1 {
				runtime.mu.Lock()
				runtime.id = "container-2"
				runtime.mu.Unlock()
			} else {
				cancel()
			}
		case "complete":
			t.Fatal("deployment completion must not complete container logs")
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("follow result = %v", err)
	}
	if len(chunks) != 2 || !strings.Contains(chunks[0], "container-1") || !strings.Contains(chunks[1], "container-2") || !gap {
		t.Fatalf("chunks=%v, gap=%v", chunks, gap)
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.closed != 2 {
		t.Fatalf("follow commands closed = %d", runtime.closed)
	}
	if !strings.Contains(runtime.commands[0], "--follow --timestamps --no-color --since 2026-09-30T00:00:00Z") || !strings.Contains(runtime.commands[1], "--since 2026-09-29T23:59:58.123456789Z") {
		t.Fatalf("commands = %v", runtime.commands)
	}
}

func TestContainerLogSubscriptionStopsWhenMembershipIsRevoked(t *testing.T) {
	svc, store, runtime := newContainerStreamService()
	runtime.id = "container-1"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream, err := svc.OpenApplicationLogStream(ctx, "user-1", "project-1", "app-1", "service-1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	err = stream.Run(ctx, func(event logdto.Event) error {
		if event.Type == "chunk" {
			store.mu.Lock()
			store.member = false
			store.mu.Unlock()
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "Permission denied") {
		t.Fatalf("membership revocation = %v", err)
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.closed != 1 {
		t.Fatal("revoked subscription left follower running")
	}
}

func TestLocalApplicationContainerLogs(t *testing.T) {
	svc, store, runtime := newContainerStreamService()
	target := testSSHTarget("project-1")
	target.Environment.TargetType = model.EnvironmentTargetTypeLocal
	target.Environment.SSH = nil
	target.Environment.WorkspaceRoot = t.TempDir()
	svc.targetResolver = staticTargetResolver{target: target}
	store.version = model.Version{Id: "version-1"}
	store.components = []model.VersionComponent{{VersionId: "version-1", Name: "web"}}
	svc.application = containerStreamApplicationStore{store: store.runtimeQueryStore}
	runtime.id = "container-1"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	subscription, err := svc.OpenApplicationLogStream(ctx, "user-1", "project-1", "app-1", "service-1", "web", "")
	if err != nil {
		t.Fatal(err)
	}
	var chunk logdto.Event
	err = subscription.Run(ctx, func(event logdto.Event) error {
		if event.Type == "chunk" {
			chunk = event
			cancel()
		}
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("follow result = %v", err)
	}
	if !strings.Contains(string(chunk.Data), "container-1") || chunk.Cursor == "" {
		t.Fatalf("chunk = %+v", chunk)
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.closed != 1 || len(runtime.commands) != 1 {
		t.Fatalf("closed=%d, commands=%v", runtime.closed, runtime.commands)
	}
	if !strings.Contains(runtime.commands[0], "--follow --timestamps --no-color --tail 200 web") {
		t.Fatalf("command = %s", runtime.commands[0])
	}
}

func TestStoppedServiceContainerLogs(t *testing.T) {
	for _, targetType := range []string{model.EnvironmentTargetTypeLocal, model.EnvironmentTargetTypeSSH} {
		t.Run(targetType, func(t *testing.T) {
			svc, store, runtime := newContainerStreamService()
			store.service.Status = status.ServiceStatusStopped
			store.service.RuntimeDirectory, store.service.RuntimeTargetRevision = "", 0
			store.version = model.Version{Id: "version-1"}
			store.components = []model.VersionComponent{{VersionId: "version-1", Name: "web"}}
			svc.application = containerStreamApplicationStore{store: store.runtimeQueryStore}
			target := testSSHTarget("project-1")
			if targetType == model.EnvironmentTargetTypeLocal {
				target.Environment.TargetType = targetType
				target.Environment.SSH = nil
			}
			svc.targetResolver = staticTargetResolver{target: target}
			runtime.id = "manually-started-container"
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			subscription, err := svc.OpenApplicationLogStream(ctx, "user-1", "project-1", "app-1", "service-1", "web", "")
			if err != nil {
				t.Fatal(err)
			}
			var chunk logdto.Event
			err = subscription.Run(ctx, func(event logdto.Event) error {
				if event.Type == "chunk" {
					chunk = event
					cancel()
				}
				return nil
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("follow result = %v", err)
			}
			if !strings.Contains(string(chunk.Data), runtime.id) || chunk.Cursor == "" {
				t.Fatalf("stopped service log chunk = %+v", chunk)
			}
			if len(runtime.commands) != 1 || runtime.commands[0] != "compose -p demo-default logs --follow --timestamps --no-color --tail 200 web" || runtime.closed != 1 {
				t.Fatalf("commands = %v, closed = %d", runtime.commands, runtime.closed)
			}
		})
	}
}

func TestContainerLogSubscriptionRejectsChangedTarget(t *testing.T) {
	for _, targetType := range []string{model.EnvironmentTargetTypeLocal, model.EnvironmentTargetTypeSSH} {
		t.Run(targetType, func(t *testing.T) {
			svc, _, runtime := newContainerStreamService()
			target := testSSHTarget("project-1")
			if targetType == model.EnvironmentTargetTypeLocal {
				target.Environment.TargetType = targetType
				target.Environment.SSH = nil
			}
			resolver := &containerStreamTargetResolver{target: target}
			svc.targetResolver = resolver
			runtime.id = "container-1"
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			subscription, err := svc.OpenApplicationLogStream(ctx, "user-1", "project-1", "app-1", "service-1", "", "")
			if err != nil {
				t.Fatal(err)
			}
			if targetType == model.EnvironmentTargetTypeLocal {
				resolver.target.Environment.TargetRevision++
			} else {
				ssh := *target.Environment.SSH
				ssh.CredentialRevision++
				resolver.target.Environment.SSH = &ssh
			}
			err = subscription.Run(ctx, func(logdto.Event) error { return nil })
			if err == nil || !strings.Contains(err.Error(), "Container log target changed") {
				t.Fatalf("target change result = %v", err)
			}
			if len(runtime.commands) != 0 {
				t.Fatalf("changed target started a follower: %v", runtime.commands)
			}
		})
	}
}
