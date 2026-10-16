package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func sshCommand(deps Dependencies) *cobra.Command {
	var project string
	command := &cobra.Command{Use: "ssh", Short: "Open a remote shell (Ctrl+] detaches locally)", Args: cobra.NoArgs}
	command.Flags().StringVar(&project, "project", "", "Exact project code; omit to select interactively")
	command.RunE = func(command *cobra.Command, _ []string) error {
		input, inOK := command.InOrStdin().(*os.File)
		output, outOK := command.OutOrStdout().(*os.File)
		if !inOK || !outOK || !term.IsTerminal(int(input.Fd())) || !term.IsTerminal(int(output.Fd())) {
			return &ExitError{Code: 2, Err: errors.New("env ssh requires stdin and stdout TTYs")}
		}
		services, closeServices, err := deps.Open(command.Context())
		if err != nil {
			return failure(command, err, false)
		}
		defer closeServices()
		if project == "" {
			project, err = selectProject(command.Context(), services, input, output)
			if err != nil {
				return failure(command, err, false)
			}
		}
		return runTerminal(command.Context(), services, project, input, output)
	}
	return command
}

func selectProject(ctx context.Context, services Services, input io.Reader, output io.Writer) (string, error) {
	projects, err := services.Access.List(ctx, services.ActorId)
	if err != nil {
		return "", err
	}
	if len(projects) == 0 {
		return "", errors.New("no member-visible SSH environments")
	}
	for index, project := range projects {
		if _, err := fmt.Fprintf(output, "%d. %s (%s) %s@%s:%d\n", index+1, project.Code, project.Name, project.Username, project.Host, project.Port); err != nil {
			return "", err
		}
	}
	if _, err := io.WriteString(output, "Project: "); err != nil {
		return "", err
	}
	type selection struct {
		line string
		err  error
	}
	read := make(chan selection, 1)
	go func() {
		line, err := bufio.NewReader(input).ReadString('\n')
		read <- selection{line, err}
	}()
	var line string
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case result := <-read:
		if result.err != nil {
			return "", result.err
		}
		line = result.line
	}
	selected, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || selected < 1 || selected > len(projects) {
		return "", &ExitError{Code: 2, Err: errors.New("invalid project selection")}
	}
	return projects[selected-1].Code, nil
}

func runTerminal(ctx context.Context, services Services, project string, input, output *os.File) error {
	grant, release, err := services.Access.AcquireTerminal(ctx, services.ActorId, project)
	if err != nil {
		return &ExitError{Code: 255, Err: err}
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, environmentsvc.TerminalSessionLimit)
	defer cancel()
	columns, rows, err := term.GetSize(int(output.Fd()))
	if err != nil {
		return err
	}
	connectTimer := time.AfterFunc(environmentsvc.TerminalConnectLimit, cancel)
	defer connectTimer.Stop()
	session, err := services.TerminalRunner.StartTerminal(ctx, grant.Target, columns, rows)
	if err != nil {
		return &ExitError{Code: 255, Err: errors.New("SSH connection or authentication failed")}
	}
	defer func() { _ = session.Close() }()
	fingerprint := ""
	if metadata, ok := session.(environmentport.TerminalHostKey); ok {
		fingerprint = metadata.HostKeyFingerprint()
	}
	grant, err = services.Terminal.ConfirmConnection(ctx, grant, fingerprint)
	if err != nil {
		return &ExitError{Code: 255, Err: err}
	}
	connectTimer.Stop()
	restoreOutput, err := enableTerminalOutput(output)
	if err != nil {
		return err
	}
	defer restoreOutput()
	raw, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(int(input.Fd()), raw) }()
	var activity atomic.Int64
	activity.Store(time.Now().UnixNano())
	events := make(chan error, 3)
	report := func(err error) {
		select {
		case events <- err:
		case <-ctx.Done():
		}
	}
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		_, err := io.Copy(activityWriter{output, &activity}, session.Output())
		if err != nil && ctx.Err() == nil {
			report(err)
		}
	}()
	// This reads the process-owned TTY. The CLI process ends with this shell;
	// the remote streams and terminal mode are closed/restored before it exits.
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, readErr := input.Read(buffer)
			if ctx.Err() != nil {
				return
			}
			data := buffer[:n]
			detach := bytes.IndexByte(data, 0x1d)
			if detach >= 0 {
				data = data[:detach]
			}
			if len(data) != 0 {
				checkCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				err := services.Terminal.ValidateSession(checkCtx, grant)
				stop()
				if err == nil {
					_, err = session.Input().Write(data)
				}
				if err != nil {
					report(err)
					return
				}
				activity.Store(time.Now().UnixNano())
			}
			if detach >= 0 || readErr != nil {
				report(nil)
				return
			}
		}
	}()
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		reason := services.Terminal.MonitorSession(ctx, grant, func() time.Time { return time.Unix(0, activity.Load()) })
		if reason != "" {
			report(errors.New(reason))
		}
	}()
	defer func() { cancel(); _ = session.Close(); <-monitorDone; <-outputDone }()
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cols, lines, err := term.GetSize(int(output.Fd()))
				if err == nil && cols > 0 && lines > 0 && (cols != columns || lines != rows) {
					if err := session.Resize(cols, lines); err != nil {
						report(err)
						return
					}
					columns, rows = cols, lines
				}
			}
		}
	}()
	completed := make(chan error, 1)
	go func() {
		code, err := session.Wait()
		<-outputDone
		if err != nil {
			completed <- &ExitError{Code: 255, Err: err}
		} else if code != 0 {
			completed <- &ExitError{Code: code}
		} else {
			completed <- nil
		}
	}()
	select {
	case err := <-completed:
		return err
	case err := <-events:
		if err != nil {
			return &ExitError{Code: 255, Err: err}
		}
		return nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return &ExitError{Code: 255, Err: errors.New("terminal session limit reached")}
		}
		return &ExitError{Code: 130}
	}
}

type activityWriter struct {
	writer   io.Writer
	activity *atomic.Int64
}

func (w activityWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if n > 0 {
		w.activity.Store(time.Now().UnixNano())
	}
	return n, err
}
