package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type commandExecutorFake struct {
	input  environmentdto.CommandInput
	actor  string
	result environmentdto.CommandResult
}

func (f *commandExecutorFake) Execute(_ context.Context, actor string, input environmentdto.CommandInput) (environmentdto.CommandResult, error) {
	f.input, f.actor = input, actor
	return f.result, nil
}

func TestExecPreservesLiteralArgvBinaryStreamsAndRemoteExit(t *testing.T) {
	code := int32(17)
	executor := &commandExecutorFake{result: environmentdto.CommandResult{State: "exited", ExitCode: &code, Stdout: []byte{0, 255}, Stderr: []byte("remote stderr")}}
	closed := false
	command := NewCommand(Dependencies{Open: func(context.Context) (Services, func(), error) {
		return Services{ActorId: "actor", Command: executor}, func() { closed = true }, nil
	}})
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetIn(bytes.NewReader([]byte{0, 254}))
	argv := []string{"program", "--json", "a b", "$HOME", ""}
	command.SetArgs(append([]string{"env", "exec", "--project", "prod", "--stdin", "--workdir", "/work", "--"}, argv...))
	err := command.Execute()
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 17 || exit.Err != nil {
		t.Fatalf("wrong exit: %v", err)
	}
	if !reflect.DeepEqual(executor.input.Argv, argv) || !bytes.Equal(executor.input.Stdin, []byte{0, 254}) || executor.input.Workdir != "/work" || executor.actor != "actor" || !closed {
		t.Fatalf("request=%+v actor=%s closed=%v", executor.input, executor.actor, closed)
	}
	if !bytes.Equal(stdout.Bytes(), executor.result.Stdout) || stderr.String() != "remote stderr" {
		t.Fatalf("stdout=%q stderr=%q", stdout.Bytes(), stderr.String())
	}
}

func TestExecJSONIncludesTextMetadataAndTimeoutExit(t *testing.T) {
	executor := &commandExecutorFake{result: environmentdto.CommandResult{State: "timed_out", ProjectCode: "prod", Stdout: []byte{'a', 255}, Stderr: []byte("err"), DurationMs: 123}}
	command := NewCommand(Dependencies{Open: func(context.Context) (Services, func(), error) {
		return Services{ActorId: "actor", Command: executor}, func() {}, nil
	}})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"env", "exec", "--project", "prod", "--json", "--", "program"})
	err := command.Execute()
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != 124 {
		t.Fatalf("wrong timeout exit: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["stdout"] != "a\uFFFD" || result["stderr"] != "err" || result["output_decoding_replaced"] != true || result["project_code"] != "prod" || result["duration_ms"] != float64(123) {
		t.Fatalf("invalid JSON result: %+v", result)
	}
}

func TestExecCancellationWhileWaitingForStdin(t *testing.T) {
	input, writer := io.Pipe()
	defer func() { _ = input.Close(); _ = writer.Close() }()
	command := NewCommand(Dependencies{Open: func(context.Context) (Services, func(), error) {
		t.Error("opened business services before stdin completed")
		return Services{}, nil, errors.New("unexpected open")
	}})
	command.SetIn(input)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"env", "exec", "--project", "prod", "--stdin", "--", "program"})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- command.ExecuteContext(ctx) }()
	cancel()
	select {
	case err := <-done:
		var exit *ExitError
		if !errors.As(err, &exit) || exit.Code != 130 {
			t.Fatalf("wrong cancellation exit: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stdin blocked cancellation")
	}
}

type forwardApplicationFake struct {
	input environmentdto.ForwardInput
	actor string
}

func (f *forwardApplicationFake) StartBackground(_ context.Context, actor string, input environmentdto.ForwardInput) (environmentdto.ForwardStatus, error) {
	f.input, f.actor = input, actor
	return environmentdto.ForwardStatus{ForwardSpec: environmentdto.ForwardSpec{ForwardInput: input}, State: "running"}, nil
}

func (*forwardApplicationFake) Run(context.Context, string, environmentdto.ForwardInput, func(environmentdto.ForwardSpec) error) error {
	return errors.New("unexpected worker entry")
}

func TestForwardStartPassesPortMappingToApplication(t *testing.T) {
	for _, flag := range []string{"-p", "--port"} {
		t.Run(flag, func(t *testing.T) {
			forward := &forwardApplicationFake{}
			command := NewCommand(Dependencies{Open: func(context.Context) (Services, func(), error) {
				return Services{ActorId: "actor", Forward: forward}, func() {}, nil
			}})
			command.SetOut(io.Discard)
			command.SetArgs([]string{"env", "forward", "start", "--project", "prod", flag, "15432:5432"})
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if forward.actor != "actor" || forward.input != (environmentdto.ForwardInput{ProjectCode: "prod", LocalPort: 15432, RemotePort: 5432}) {
				t.Fatalf("actor=%s input=%+v", forward.actor, forward.input)
			}
		})
	}
}

func TestForwardStartRequiresPairedPortsBeforeOpeningServices(t *testing.T) {
	command := NewCommand(Dependencies{Open: func(context.Context) (Services, func(), error) {
		t.Error("opened business services for an invalid port mapping")
		return Services{}, nil, errors.New("unexpected open")
	}})
	command.SetErr(io.Discard)
	command.SetArgs([]string{"env", "forward", "start", "--project", "prod", "-p", "15432"})
	var exit *ExitError
	if err := command.Execute(); !errors.As(err, &exit) || exit.Code != 2 {
		t.Fatalf("wrong mapping error: %v", err)
	}
}

type localProcessesFake struct {
	stopped   int
	inspected []int
	statuses  []environmentdto.ForwardStatus
	stopErr   error
}

func (*localProcessesFake) Start(context.Context, environmentdto.ForwardSpec) (environmentdto.ForwardStatus, error) {
	return environmentdto.ForwardStatus{}, errors.New("unexpected start")
}
func (f *localProcessesFake) Status(_ context.Context, port int) ([]environmentdto.ForwardStatus, error) {
	f.inspected = append(f.inspected, port)
	return f.statuses, nil
}
func (f *localProcessesFake) Stop(_ context.Context, port int) (bool, error) {
	f.stopped = port
	return len(f.statuses) != 0, f.stopErr
}

func TestForwardStatusAndStopDoNotOpenBusinessServices(t *testing.T) {
	local := &localProcessesFake{statuses: []environmentdto.ForwardStatus{{ForwardSpec: environmentdto.ForwardSpec{ForwardInput: environmentdto.ForwardInput{ProjectCode: "prod", LocalPort: 15432, RemotePort: 5432}}, State: "running", Pid: 123}}}
	deps := Dependencies{
		Open: func(context.Context) (Services, func(), error) {
			t.Fatal("opened business database/identity for local control")
			return Services{}, nil, nil
		},
		Processes: func(context.Context) (environmentport.ForwardProcesses, error) { return local, nil },
	}
	for _, args := range [][]string{
		{"env", "forward", "status", "--json"},
		{"env", "forward", "status", "-p", "15432"},
		{"env", "forward", "status", "--port", "15432"},
		{"env", "forward", "stop", "-p", "15432"},
		{"env", "forward", "stop", "--port", "15432"},
	} {
		command := NewCommand(deps)
		var output bytes.Buffer
		command.SetOut(&output)
		command.SetErr(io.Discard)
		command.SetArgs(args)
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		if args[2] == "stop" && output.String() != "Stopped managed forward on local port 15432.\n" {
			t.Fatalf("missing stop confirmation: %q", output.String())
		}
	}
	if local.stopped != 15432 || !reflect.DeepEqual(local.inspected, []int{0, 15432, 15432}) {
		t.Fatalf("wrong local port: stopped=%d inspected=%v", local.stopped, local.inspected)
	}
}

func TestForwardMissingRecordFeedback(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"status"}, "No managed forwards found"},
		{[]string{"status", "-p", "15432"}, "No managed forward found for local port 15432"},
		{[]string{"stop", "-p", "15432"}, "No managed forward found for local port 15432"},
		{[]string{"status", "--json"}, "{\"forwards\":[]}\n"},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			local := &localProcessesFake{statuses: []environmentdto.ForwardStatus{}}
			command := NewCommand(Dependencies{Processes: func(context.Context) (environmentport.ForwardProcesses, error) { return local, nil }})
			var output bytes.Buffer
			command.SetOut(&output)
			command.SetArgs(append([]string{"env", "forward"}, test.args...))
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(output.String(), test.want) {
				t.Fatalf("output=%q stopped=%d", output.String(), local.stopped)
			}
		})
	}
}

func TestForwardStopFailureDoesNotReportSuccess(t *testing.T) {
	local := &localProcessesFake{statuses: []environmentdto.ForwardStatus{{State: "unresponsive"}}, stopErr: errors.New("cannot verify/stop managed forward")}
	command := NewCommand(Dependencies{Processes: func(context.Context) (environmentport.ForwardProcesses, error) { return local, nil }})
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"env", "forward", "stop", "-p", "15432"})
	if err := command.Execute(); !errors.Is(err, local.stopErr) {
		t.Fatalf("wrong stop error: %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("reported success after failed stop: %q", output.String())
	}
}
