package forwardprocess

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
)

func TestMain(tests *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "cli" {
		// The real background-process adapter launches this executable. Replace
		// only its application callback, keeping real detach, records, locks,
		// authenticated local control, and OS process identity checks.
		manager, err := New("data/cli/forward")
		if err == nil && len(os.Args) == 9 {
			port, parseErr := strconv.Atoi(os.Args[6])
			err = parseErr
			if err == nil {
				err = manager.Run(context.Background(), port, os.Args[8], func(ctx context.Context, input environmentdto.ForwardInput, ready func(environmentdto.ForwardSpec) error) error {
					listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", input.LocalPort))
					if err != nil {
						return err
					}
					defer func() { _ = listener.Close() }()
					state, err := manager.read(port)
					if err != nil {
						return err
					}
					if err := ready(state.ForwardSpec); err != nil {
						return err
					}
					<-ctx.Done()
					return nil
				})
			}
		}
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(tests.Run())
}

func processTestManager(t *testing.T) *Manager {
	t.Helper()
	t.Chdir(t.TempDir())
	manager, err := New("data/cli/forward")
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func reservePort(t *testing.T) (net.Listener, int) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener, listener.Addr().(*net.TCPAddr).Port
}

func TestManagedForwardProcessLifecycleAndConcurrentStart(t *testing.T) {
	manager := processTestManager(t)
	reservation, port := reservePort(t)
	_ = reservation.Close()
	spec := environmentdto.ForwardSpec{ForwardInput: environmentdto.ForwardInput{ProjectCode: "prod", LocalPort: port, RemotePort: 5432}, ActorId: "actor", EnvironmentId: "environment", TargetRevision: 4, CredentialRevision: 2}
	t.Cleanup(func() {
		if _, err := manager.Stop(context.Background(), port); err != nil {
			t.Error(err)
		}
	})
	var results [2]environmentdto.ForwardStatus
	var failures [2]error
	var workers sync.WaitGroup
	for index := range results {
		workers.Go(func() { results[index], failures[index] = manager.Start(t.Context(), spec) })
	}
	workers.Wait()
	for index, result := range results {
		if failures[index] != nil || result.State != "running" || result.Pid == os.Getpid() || result.Pid != results[0].Pid {
			t.Fatalf("result=%+v failure=%v", result, failures[index])
		}
	}
	statuses, err := manager.Status(t.Context(), 0)
	if err != nil || len(statuses) != 1 || statuses[0].Pid != results[0].Pid || statuses[0].State != "running" {
		t.Fatalf("status=%+v err=%v", statuses, err)
	}
	state, err := manager.read(port)
	if err != nil {
		t.Fatal(err)
	}
	state.ControlKey = strings.Repeat("0", 64)
	if _, err := manager.control(t.Context(), state, "stop"); err == nil {
		t.Fatal("public run ID authorized a stop without the private control key")
	}
	if statuses, err := manager.Status(t.Context(), port); err != nil || len(statuses) != 1 || statuses[0].State != "running" {
		t.Fatalf("unauthenticated control affected the forward: %+v %v", statuses, err)
	}
	conflict := spec
	conflict.RemotePort++
	if _, err := manager.Start(t.Context(), conflict); err == nil {
		t.Fatal("replaced a conflicting active mapping")
	}
	if found, err := manager.Stop(t.Context(), port); err != nil || !found {
		t.Fatalf("stop found=%v err=%v", found, err)
	}
	if found, err := manager.Stop(t.Context(), port); err != nil || found {
		t.Fatalf("repeated stop found=%v err=%v", found, err)
	}
	if statuses, err := manager.Status(t.Context(), port); err != nil || len(statuses) != 0 {
		t.Fatalf("stopped state not removed: %+v %v", statuses, err)
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal("stopped child still owns the listener:", err)
	}
	_ = listener.Close()
}

func TestStartFailureReapsChildAndPreservesForeignListener(t *testing.T) {
	manager := processTestManager(t)
	listener, port := reservePort(t)
	spec := environmentdto.ForwardSpec{ForwardInput: environmentdto.ForwardInput{ProjectCode: "prod", LocalPort: port, RemotePort: 5432}}
	if _, err := manager.Start(t.Context(), spec); err == nil {
		t.Fatal("started on an occupied port")
	}
	if _, err := os.Stat(manager.path(port)); !os.IsNotExist(err) {
		t.Fatalf("failed startup left a record: %v", err)
	}
	if found, err := manager.Stop(t.Context(), port); err != nil || found {
		t.Fatalf("stop without record found=%v err=%v", found, err)
	}
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal("foreign listener was stopped:", err)
	}
	_ = connection.Close()
}

func TestStatusAndStopDoNotTrustReusedPid(t *testing.T) {
	manager := processTestManager(t)
	listener, port := reservePort(t)
	state := record{ForwardStatus: environmentdto.ForwardStatus{ForwardSpec: environmentdto.ForwardSpec{ForwardInput: environmentdto.ForwardInput{LocalPort: port}}, Pid: os.Getpid(), State: "running"},
		RunId: strings.Repeat("a", 64), ControlKey: strings.Repeat("b", 64), Created: "previous process", Executable: manager.executable, Control: listener.Addr().String()}
	if err := manager.write(state); err != nil {
		t.Fatal(err)
	}
	statuses, err := manager.Status(t.Context(), port)
	if err != nil || len(statuses) != 1 || statuses[0].State != "stale" {
		t.Fatalf("stale PID accepted: %+v %v", statuses, err)
	}
	if found, err := manager.Stop(t.Context(), port); err != nil || !found {
		t.Fatalf("stale record cleanup found=%v err=%v", found, err)
	}
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal("unrelated process/listener affected:", err)
	}
	_ = connection.Close()
}
