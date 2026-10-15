package environmentsvc

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type cliProjects struct {
	*terminalProjectReader
	items []model.Project
}

func (p cliProjects) ProjectByCode(_ context.Context, code string) (model.Project, error) {
	for _, item := range p.items {
		if item.Code == code {
			return item, nil
		}
	}
	return model.Project{}, repository.ErrNotFound
}
func (p cliProjects) ListProjectsByMember(context.Context, string) ([]model.Project, error) {
	return p.items, nil
}

type commandRunnerFake struct {
	open        func(context.Context, environmentport.Target) error
	run         func(context.Context, environmentport.Command, io.Writer, io.Writer) (environmentport.CommandOutcome, error)
	fingerprint string
	closed      atomic.Bool
	calls       atomic.Int32
}

func (r *commandRunnerFake) Open(ctx context.Context, target environmentport.Target) (environmentport.CommandSession, error) {
	r.calls.Add(1)
	if r.open != nil {
		if err := r.open(ctx, target); err != nil {
			return nil, err
		}
	}
	return r, nil
}
func (r *commandRunnerFake) HostKeyFingerprint() string { return r.fingerprint }
func (r *commandRunnerFake) Close() error               { r.closed.Store(true); return nil }
func (r *commandRunnerFake) Run(ctx context.Context, c environmentport.Command, out, errOut io.Writer) (environmentport.CommandOutcome, error) {
	return r.run(ctx, c, out, errOut)
}
func newCommandTest(t *testing.T) (*CommandService, *probeEnvironmentStore, *terminalProjectReader, *commandRunnerFake) {
	t.Helper()
	environment := readyTargetEnvironment()
	environment.Code = "project"
	environment.SSH.HostKeyFingerprint = ""
	environment.LastProbeStatus = nil
	environment.GatewayApplicationId = nil
	store := &probeEnvironmentStore{environment: environment}
	projects := &terminalProjectReader{member: true}
	service := New(store, projects, credentialsForEnvironment(t, environment, "managed-key"), testCredentialSecret, nil, nil)
	access := NewAccessService(service, cliProjects{projects, []model.Project{{Id: environment.ProjectId, Code: "production"}}}, NewTerminalService(service))
	runner := &commandRunnerFake{fingerprint: "SHA256:abcdefghijk"}
	return NewCommandService(access, runner), store, projects, runner
}

func TestCommandPinsSameConnectionBeforeLiteralExecutionAndPreservesExit(t *testing.T) {
	service, store, _, runner := newCommandTest(t)
	input := environmentdto.CommandInput{ProjectCode: "production", Argv: []string{"program", "$HOME", "~/file", "a b", "--help"}, Stdin: []byte{0, 255}}
	runner.run = func(_ context.Context, c environmentport.Command, out, errOut io.Writer) (environmentport.CommandOutcome, error) {
		if !store.updated || store.environment.SSH.HostKeyFingerprint != runner.fingerprint {
			t.Error("command sent before fingerprint confirmation")
		}
		if !reflect.DeepEqual(c.Argv, input.Argv) || !reflect.DeepEqual(c.Stdin, input.Stdin) || c.Workdir != store.environment.WorkspaceRoot {
			t.Errorf("command changed: %#v", c)
		}
		_, _ = out.Write([]byte{0, 255})
		_, _ = io.WriteString(errOut, "failure")
		code := int32(7)
		return environmentport.CommandOutcome{Sent: true, ExitCode: &code}, nil
	}
	result, err := service.Execute(context.Background(), "actor", input)
	if err != nil || result.State != "exited" || result.ExitCode == nil || *result.ExitCode != 7 || !reflect.DeepEqual(result.Stdout, []byte{0, 255}) || string(result.Stderr) != "failure" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !runner.closed.Load() || service.total != 0 {
		t.Fatal("command resources were not released")
	}
}

func TestCommandRejectsUnavailableAccessBeforeSending(t *testing.T) {
	for _, scenario := range []string{"nonmember", "unknown code", "local", "fingerprint conflict", "credential changed"} {
		t.Run(scenario, func(t *testing.T) {
			service, store, projects, runner := newCommandTest(t)
			code := "production"
			switch scenario {
			case "nonmember":
				projects.member = false
			case "unknown code":
				code = "project-1"
			case "local":
				store.environment.TargetType = "local"
				store.environment.SSH = nil
			case "fingerprint conflict":
				store.stale = true
			case "credential changed":
				runner.open = func(context.Context, environmentport.Target) error { store.environment.TargetRevision++; return nil }
			}
			runner.run = func(context.Context, environmentport.Command, io.Writer, io.Writer) (environmentport.CommandOutcome, error) {
				t.Error("sent unauthorized command")
				return environmentport.CommandOutcome{}, nil
			}
			_, err := service.Execute(context.Background(), "actor", environmentdto.CommandInput{ProjectCode: code, Argv: []string{"true"}})
			if err == nil || service.total != 0 {
				t.Fatalf("err=%v quota=%d", err, service.total)
			}
		})
	}
}

func TestCommandOutputBudgetAndUnknownTransport(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown transport", true: "combined output budget"}[overflow], func(t *testing.T) {
			service, _, _, runner := newCommandTest(t)
			runner.run = func(_ context.Context, _ environmentport.Command, out, errOut io.Writer) (environmentport.CommandOutcome, error) {
				if overflow {
					_, _ = io.WriteString(out, strings.Repeat("a", commandOutputLimit-2))
					_, _ = io.WriteString(errOut, "error")
				}
				return environmentport.CommandOutcome{Sent: true}, errors.New("secret transport cause")
			}
			result, err := service.Execute(context.Background(), "actor", environmentdto.CommandInput{ProjectCode: "production", Argv: []string{"true"}})
			if err != nil || result.ExitCode != nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if overflow {
				if result.State != "output_limit" || !result.OutputTruncated || len(result.Stdout)+len(result.Stderr) != commandOutputLimit {
					t.Fatal("output was not bounded")
				}
			} else if result.State != "transport_lost" {
				t.Fatalf("state=%s", result.State)
			}
		})
	}
}

func TestCommandTimeoutAndAccessRevocationCloseConnection(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "revocation"}[revoke], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, _, projects, runner := newCommandTest(t)
				running := make(chan struct{})
				runner.run = func(ctx context.Context, _ environmentport.Command, _, _ io.Writer) (environmentport.CommandOutcome, error) {
					close(running)
					<-ctx.Done()
					return environmentport.CommandOutcome{Sent: true}, ctx.Err()
				}
				timeout := uint32(1)
				if revoke {
					timeout = 60
				}
				resultCh := make(chan environmentdto.CommandResult, 1)
				go func() {
					result, err := service.Execute(context.Background(), "actor", environmentdto.CommandInput{ProjectCode: "production", Argv: []string{"wait"}, TimeoutSeconds: &timeout})
					if err != nil {
						t.Error(err)
					}
					resultCh <- result
				}()
				<-running
				synctest.Wait()
				if revoke {
					projects.member = false
				}
				time.Sleep(3 * time.Second)
				result := <-resultCh
				expected := "timed_out"
				if revoke {
					expected = "access_or_target_changed"
				}
				if result.State != expected || result.ExitCode != nil || !runner.closed.Load() || service.total != 0 {
					t.Fatalf("result=%+v closed=%v quota=%d", result, runner.closed.Load(), service.total)
				}
			})
		})
	}
}

func TestCommandValidationAndQuota(t *testing.T) {
	zero := uint32(0)
	for _, input := range []environmentdto.CommandInput{{}, {Argv: []string{"x\x00"}}, {Argv: []string{"x"}, Stdin: make([]byte, 64*1024+1)}, {Argv: []string{"x"}, TimeoutSeconds: &zero}} {
		if _, err := validateCommand(input); !apperror.IsKind(err, apperror.KindValidation) {
			t.Fatalf("accepted invalid command: %v", err)
		}
	}
	if err := validateCommandWorkdir("relative", "linux"); err == nil {
		t.Fatal("accepted relative workdir")
	}
	service, _, _, _ := newCommandTest(t)
	first, err := service.acquire("actor")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.acquire("actor")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.acquire("actor"); !apperror.IsKind(err, apperror.KindUnavailable) {
		t.Fatal("quota not enforced")
	}
	first()
	second()
	if service.total != 0 {
		t.Fatal("quota not released")
	}
}
