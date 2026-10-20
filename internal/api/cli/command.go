package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/spf13/cobra"
)

type EnvironmentAccess interface {
	List(context.Context, string) ([]environmentdto.ProjectSummary, error)
	AcquireTerminal(context.Context, string, string) (environmentsvc.TerminalGrant, func(), error)
}

type CommandExecutor interface {
	Execute(context.Context, string, environmentdto.CommandInput) (environmentdto.CommandResult, error)
}

type ForwardApplication interface {
	StartBackground(context.Context, string, environmentdto.ForwardInput) (environmentdto.ForwardStatus, error)
	Run(context.Context, string, environmentdto.ForwardInput, func(environmentdto.ForwardSpec) error) error
}

type Services struct {
	ActorId        string
	Access         EnvironmentAccess
	Command        CommandExecutor
	Forward        ForwardApplication
	Terminal       *environmentsvc.TerminalService
	TerminalRunner environmentport.TerminalRunner
}

type Dependencies struct {
	Open       func(context.Context) (Services, func(), error)
	Processes  func(context.Context) (environmentport.ForwardProcesses, error)
	RunForward func(context.Context, int, string) error
}

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}
func (e *ExitError) Unwrap() error { return e.Err }

func NewCommand(deps Dependencies) *cobra.Command {
	command := &cobra.Command{Use: "cli", Short: "Local operations using Orbit configuration and business services", SilenceUsage: true, SilenceErrors: true}
	env := &cobra.Command{Use: "env", Short: "Manage stored remote SSH environments"}
	command.AddCommand(env)
	env.AddCommand(listCommand(deps), execCommand(deps), sshCommand(deps), forwardCommand(deps))
	return command
}

func failure(command *cobra.Command, err error, asJSON bool) error {
	code := 255
	var exit *ExitError
	if errors.As(err, &exit) {
		code = exit.Code
	} else if apperror.IsKind(err, apperror.KindValidation) {
		code = 2
	} else if errors.Is(err, context.Canceled) {
		code = 130
	}
	if asJSON {
		if writeErr := json.NewEncoder(command.OutOrStdout()).Encode(map[string]string{"state": "platform_error", "error": err.Error()}); writeErr != nil {
			return &ExitError{Code: 255, Err: writeErr}
		}
		return &ExitError{Code: code}
	}
	return &ExitError{Code: code, Err: err}
}

func listCommand(deps Dependencies) *cobra.Command {
	var asJSON bool
	command := &cobra.Command{Use: "list", Short: "List member-visible SSH environments", Args: cobra.NoArgs}
	command.Flags().BoolVar(&asJSON, "json", false, "Emit JSON")
	command.RunE = func(command *cobra.Command, _ []string) error {
		services, closeServices, err := deps.Open(command.Context())
		if err != nil {
			return failure(command, err, asJSON)
		}
		defer closeServices()
		items, err := services.Access.List(command.Context(), services.ActorId)
		if err != nil {
			return failure(command, err, asJSON)
		}
		if asJSON {
			return json.NewEncoder(command.OutOrStdout()).Encode(map[string]any{"projects": items})
		}
		for _, item := range items {
			if _, err := fmt.Fprintf(command.OutOrStdout(), "%s\t%s\t%s@%s:%d\n", item.Code, item.Name, item.Username, item.Host, item.Port); err != nil {
				return err
			}
		}
		return nil
	}
	return command
}

func execCommand(deps Dependencies) *cobra.Command {
	var project, workdir string
	var timeout uint32
	var readStdin, asJSON bool
	command := &cobra.Command{Use: "exec --project CODE [options] -- PROGRAM [ARG...]", Short: "Execute literal argv on a remote SSH environment", Args: cobra.MinimumNArgs(1)}
	command.Flags().StringVar(&project, "project", "", "Exact project code")
	command.Flags().StringVar(&workdir, "workdir", "", "Absolute remote directory or ~/path; default: saved workspace")
	command.Flags().Uint32Var(&timeout, "timeout", 60, "Execution timeout in seconds (1–300)")
	command.Flags().BoolVar(&readStdin, "stdin", false, "Read up to 64 KiB of stdin; otherwise send EOF")
	command.Flags().BoolVar(&asJSON, "json", false, "Emit execution metadata and output text as JSON")
	_ = command.MarkFlagRequired("project")
	command.RunE = func(command *cobra.Command, argv []string) error {
		input := environmentdto.CommandInput{ProjectCode: project, Workdir: workdir, Argv: argv, TimeoutSeconds: &timeout}
		if readStdin {
			var err error
			input.Stdin, err = readCommandStdin(command.Context(), command.InOrStdin())
			if err != nil {
				return failure(command, err, asJSON)
			}
			if len(input.Stdin) > 64*1024 {
				return failure(command, &ExitError{Code: 2, Err: errors.New("stdin exceeds 64 KiB")}, asJSON)
			}
		}
		services, closeServices, err := deps.Open(command.Context())
		if err != nil {
			return failure(command, err, asJSON)
		}
		defer closeServices()
		result, err := services.Command.Execute(command.Context(), services.ActorId, input)
		if command.Context().Err() != nil {
			return failure(command, command.Context().Err(), asJSON)
		}
		if err != nil {
			return failure(command, err, asJSON)
		}
		if asJSON {
			output := struct {
				environmentdto.CommandResult
				Stdout                 string `json:"stdout"`
				Stderr                 string `json:"stderr"`
				OutputDecodingReplaced bool   `json:"output_decoding_replaced"`
			}{result, strings.ToValidUTF8(string(result.Stdout), "\uFFFD"), strings.ToValidUTF8(string(result.Stderr), "\uFFFD"), !utf8.Valid(result.Stdout) || !utf8.Valid(result.Stderr)}
			if err := json.NewEncoder(command.OutOrStdout()).Encode(output); err != nil {
				return err
			}
		} else {
			if _, err := command.OutOrStdout().Write(result.Stdout); err != nil {
				return err
			}
			if _, err := command.ErrOrStderr().Write(result.Stderr); err != nil {
				return err
			}
			if result.State != "exited" {
				_, _ = fmt.Fprintf(command.ErrOrStderr(), "\nOrbit: %s; command result incomplete\n", result.State)
			}
		}
		if code := commandExitCode(result); code != 0 {
			return &ExitError{Code: code}
		}
		return nil
	}
	return command
}

func readCommandStdin(ctx context.Context, input io.Reader) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	// The CLI owns stdin for its process lifetime. A blocked console/pipe read
	// must not prevent signal cancellation from returning and exiting.
	go func() {
		data, err := io.ReadAll(io.LimitReader(input, 64*1024+1))
		done <- result{data, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case value := <-done:
		return value.data, value.err
	}
}

func commandExitCode(result environmentdto.CommandResult) int {
	switch result.State {
	case "exited":
		if result.ExitCode != nil && *result.ExitCode >= 0 && *result.ExitCode <= 255 {
			return int(*result.ExitCode)
		}
	case "timed_out":
		return 124
	case "output_limit", "access_or_target_changed":
		return 125
	}
	return 255
}

func forwardCommand(deps Dependencies) *cobra.Command {
	command := &cobra.Command{Use: "forward", Short: "Manage background local TCP port forwards"}
	var input environmentdto.ForwardInput
	var ports string
	start := &cobra.Command{Use: "start", Short: "Start a local loopback port forward in the background", Args: cobra.NoArgs}
	start.Flags().StringVar(&input.ProjectCode, "project", "", "Exact project code")
	start.Flags().StringVarP(&ports, "port", "p", "", "LOCAL:REMOTE TCP ports, e.g. 15432:5432 (both endpoints use 127.0.0.1)")
	for _, flag := range []string{"project", "port"} {
		_ = start.MarkFlagRequired(flag)
	}
	start.RunE = func(command *cobra.Command, _ []string) error {
		local, remote, paired := strings.Cut(ports, ":")
		localPort, localErr := strconv.Atoi(local)
		remotePort, remoteErr := strconv.Atoi(remote)
		if !paired || localErr != nil || remoteErr != nil {
			return failure(command, &ExitError{Code: 2, Err: errors.New("port must use LOCAL:REMOTE, e.g. 15432:5432")}, false)
		}
		input.LocalPort, input.RemotePort = localPort, remotePort
		services, closeServices, err := deps.Open(command.Context())
		if err != nil {
			return failure(command, err, false)
		}
		defer closeServices()
		status, err := services.Forward.StartBackground(command.Context(), services.ActorId, input)
		if err != nil {
			return failure(command, err, false)
		}
		return writeForward(command.OutOrStdout(), status)
	}
	var statusPort, stopPort, workerPort int
	var asJSON bool
	status := &cobra.Command{Use: "status", Short: "Inspect this OS user's managed forwards", Args: cobra.NoArgs}
	status.Flags().IntVarP(&statusPort, "port", "p", 0, "Inspect one local port; omit for all")
	status.Flags().BoolVar(&asJSON, "json", false, "Emit JSON")
	status.RunE = func(command *cobra.Command, _ []string) error {
		if command.Flags().Changed("port") && (statusPort < 1 || statusPort > 65535) {
			return failure(command, &ExitError{Code: 2, Err: errors.New("local port must be 1–65535")}, asJSON)
		}
		processes, err := deps.Processes(command.Context())
		if err != nil {
			return failure(command, err, asJSON)
		}
		items, err := processes.Status(command.Context(), statusPort)
		if err != nil {
			return failure(command, err, asJSON)
		}
		if asJSON {
			return json.NewEncoder(command.OutOrStdout()).Encode(map[string]any{"forwards": items})
		}
		if len(items) == 0 {
			return writeMissingForward(command.OutOrStdout(), statusPort)
		}
		for _, item := range items {
			if err := writeForward(command.OutOrStdout(), item); err != nil {
				return err
			}
		}
		return nil
	}
	stop := &cobra.Command{Use: "stop", Short: "Stop one managed forward", Args: cobra.NoArgs}
	stop.Flags().IntVarP(&stopPort, "port", "p", 0, "Local port to stop")
	_ = stop.MarkFlagRequired("port")
	stop.RunE = func(command *cobra.Command, _ []string) error {
		if stopPort < 1 || stopPort > 65535 {
			return failure(command, &ExitError{Code: 2, Err: errors.New("local port must be 1–65535")}, false)
		}
		processes, err := deps.Processes(command.Context())
		if err != nil {
			return failure(command, err, false)
		}
		found, err := processes.Stop(command.Context(), stopPort)
		if err != nil {
			return failure(command, err, false)
		}
		if !found {
			return writeMissingForward(command.OutOrStdout(), stopPort)
		}
		_, err = fmt.Fprintf(command.OutOrStdout(), "Stopped managed forward on local port %d.\n", stopPort)
		return err
	}
	var runId string
	worker := &cobra.Command{Use: "__run", Hidden: true, Args: cobra.NoArgs}
	worker.Flags().IntVar(&workerPort, "local-port", 0, "Internal local port")
	worker.Flags().StringVar(&runId, "run-id", "", "Internal run identity")
	_ = worker.MarkFlagRequired("local-port")
	_ = worker.MarkFlagRequired("run-id")
	worker.RunE = func(command *cobra.Command, _ []string) error {
		return deps.RunForward(command.Context(), workerPort, runId)
	}
	command.AddCommand(start, status, stop, worker)
	return command
}

func writeMissingForward(output io.Writer, port int) error {
	if port != 0 {
		_, err := fmt.Fprintf(output, "No managed forward found for local port %d in the current working directory for this OS user.\n", port)
		return err
	}
	_, err := fmt.Fprintln(output, "No managed forwards found in the current working directory for this OS user.")
	return err
}

func writeForward(output io.Writer, state environmentdto.ForwardStatus) error {
	_, err := fmt.Fprintf(output, "%s\t127.0.0.1:%d -> %s:127.0.0.1:%d\t%s\tpid=%d\t%s\n", state.ProjectCode, state.LocalPort, state.ProjectCode, state.RemotePort, state.State, state.Pid, state.Reason)
	return err
}
