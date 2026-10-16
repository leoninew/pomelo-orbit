package environmentsvc

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type forwardRunnerFake struct {
	open   func(environmentport.Target, int, int)
	listen func() error
	done   chan struct{}
	once   sync.Once
	closed atomic.Bool
}

func (r *forwardRunnerFake) OpenForward(_ context.Context, target environmentport.Target, local, remote int) (environmentport.ForwardSession, error) {
	if r.open != nil {
		r.open(target, local, remote)
	}
	return r, nil
}
func (*forwardRunnerFake) HostKeyFingerprint() string { return "SHA256:abcdefghijk" }
func (r *forwardRunnerFake) Listen() error {
	if r.listen != nil {
		return r.listen()
	}
	return nil
}
func (r *forwardRunnerFake) Wait() error { <-r.done; return nil }
func (r *forwardRunnerFake) Close() error {
	r.closed.Store(true)
	r.once.Do(func() { close(r.done) })
	return nil
}

func newForwardTest(t *testing.T) (*ForwardService, *probeEnvironmentStore, *terminalProjectReader, *forwardRunnerFake) {
	t.Helper()
	environment := readyTargetEnvironment()
	environment.Code = "project"
	environment.SSH.HostKeyFingerprint = ""
	environment.LastProbeStatus = nil
	store := &probeEnvironmentStore{environment: environment}
	projects := &terminalProjectReader{member: true}
	environments := New(store, projects, credentialsForEnvironment(t, environment, "managed-key"), testCredentialSecret, nil, nil)
	access := NewAccessService(environments, cliProjects{projects, []model.Project{{Id: environment.ProjectId, Code: "production"}}}, NewTerminalService(environments))
	runner := &forwardRunnerFake{done: make(chan struct{})}
	return NewForwardService(access, runner, nil), store, projects, runner
}

func TestForwardConfirmsSameConnectionBeforeListeningAndCleansUp(t *testing.T) {
	service, store, _, runner := newForwardTest(t)
	runner.open = func(target environmentport.Target, local, remote int) {
		if target.PrivateKey == nil || target.PrivateKey.PrivateKey != "managed-key" || local != 15432 || remote != 5432 {
			t.Error("forward did not use the saved credential and requested ports")
		}
	}
	runner.listen = func() error {
		if !store.updated || store.environment.SSH.HostKeyFingerprint != runner.HostKeyFingerprint() {
			t.Error("local listener exposed before host-key confirmation")
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := false
	err := service.Run(ctx, "actor", environmentdto.ForwardInput{ProjectCode: "production", LocalPort: 15432, RemotePort: 5432}, func(spec environmentdto.ForwardSpec) error {
		ready = true
		if spec.ActorId != "actor" || spec.EnvironmentId != store.environment.Id || spec.CredentialRevision != store.environment.SSH.CredentialRevision {
			t.Errorf("wrong forward identity: %+v", spec)
		}
		cancel()
		return nil
	})
	if err != nil || !ready || !runner.closed.Load() {
		t.Fatalf("ready=%v closed=%v err=%v", ready, runner.closed.Load(), err)
	}
}

func TestForwardClosesOnListenFailureAndDoesNotPublishReady(t *testing.T) {
	service, _, _, runner := newForwardTest(t)
	runner.listen = func() error { return errors.New("port occupied") }
	err := service.Run(context.Background(), "actor", environmentdto.ForwardInput{ProjectCode: "production", LocalPort: 15432, RemotePort: 5432}, func(environmentdto.ForwardSpec) error {
		t.Fatal("published a failed listener")
		return nil
	})
	if err == nil || !runner.closed.Load() {
		t.Fatalf("err=%v closed=%v", err, runner.closed.Load())
	}
}

func TestForwardRechecksAccessWithoutTerminalIdleLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, _, projects, runner := newForwardTest(t)
		ready, done := make(chan struct{}), make(chan error, 1)
		go func() {
			done <- service.Run(t.Context(), "actor", environmentdto.ForwardInput{ProjectCode: "production", LocalPort: 15432, RemotePort: 5432}, func(environmentdto.ForwardSpec) error { close(ready); return nil })
		}()
		select {
		case <-ready:
		case err := <-done:
			t.Fatalf("forward failed before ready: %v", err)
		}
		time.Sleep(16 * time.Minute)
		synctest.Wait()
		select {
		case err := <-done:
			t.Fatalf("idle forward terminated: %v", err)
		default:
		}
		projects.member = false
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if err := <-done; !errors.Is(err, errTargetChanged) || !runner.closed.Load() {
			t.Fatalf("revoked forward not closed: %v", err)
		}
	})
}

func TestForwardRejectsLocalAndRevokedAccessBeforeConnecting(t *testing.T) {
	for _, scenario := range []string{"local", "revoked"} {
		t.Run(scenario, func(t *testing.T) {
			service, store, projects, runner := newForwardTest(t)
			if scenario == "local" {
				store.environment.TargetType, store.environment.SSH = "local", nil
			} else {
				projects.member = false
			}
			runner.open = func(environmentport.Target, int, int) { t.Fatal("connected an unauthorized target") }
			if err := service.Run(t.Context(), "actor", environmentdto.ForwardInput{ProjectCode: "production", LocalPort: 15432, RemotePort: 5432}, func(environmentdto.ForwardSpec) error { return nil }); err == nil {
				t.Fatal("accepted unauthorized forward")
			}
		})
	}
}
