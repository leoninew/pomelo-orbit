package localrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type terminalCapture struct {
	mu   sync.Mutex
	text strings.Builder
	done chan struct{}
	stop chan struct{}
}

func captureTerminal(session environmentport.TerminalSession) *terminalCapture {
	capture := &terminalCapture{done: make(chan struct{}), stop: make(chan struct{})}
	go func() {
		defer close(capture.done)
		buffer := make([]byte, 4096)
		for {
			n, err := session.Output().Read(buffer)
			if n > 0 {
				capture.mu.Lock()
				if capture.text.Len() < 4*1024*1024 {
					_, _ = capture.text.Write(buffer[:n])
				}
				capture.mu.Unlock()
				if strings.Contains(string(buffer[:n]), "\x1b[6n") {
					_, _ = session.Input().Write([]byte("\x1b[1;1R"))
				}
			}
			if err != nil {
				return
			}
			select {
			case <-capture.stop:
				return
			default:
			}
		}
	}()
	return capture
}

func (c *terminalCapture) output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.text.String()
}

func awaitLocalTerminal(t *testing.T, condition func() bool, failure func() string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(failure())
}

func startTestLocalTerminal(t *testing.T, directory string) (environmentport.TerminalSession, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	t.Cleanup(cancel)
	runner := NewRuntime(func(context.Context, string) (string, error) {
		t.Error("terminal invoked Docker path resolution")
		return "", fmt.Errorf("Docker unavailable")
	})
	session, err := runner.StartTerminal(ctx, environmentport.Target{Environment: model.Environment{
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: directory}}, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session, cancel
}

func writeTerminalCommand(t *testing.T, session environmentport.TerminalSession, command string) {
	t.Helper()
	if _, err := session.Input().Write([]byte(command + "\r")); err != nil {
		t.Fatal(err)
	}
}

func TestLocalTerminalWorkspaceResizeExitAndTailWithoutDocker(t *testing.T) {
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	directory := filepath.Join(t.TempDir(), "new-workspace")
	session, _ := startTestLocalTerminal(t, directory)
	capture := captureTerminal(session)
	if err := session.Resize(110, 35); err != nil {
		t.Fatal(err)
	}
	command := `printf 'ORBIT-%s:%s\n' CWD "$PWD"; printf 'ORBIT-%s:%s\n' SIZE "$(stty size)"; printf 'ORBIT-%s:%s\n' HOST "$(hostname)"; printf 'ORBIT-%s\n' TAIL; exit 7`
	if runtime.GOOS == "windows" {
		command = `Write-Output ('ORBIT'+'-CWD:'+(Get-Location).Path); Write-Output ('ORBIT'+'-SIZE:'+ [Console]::WindowWidth+':'+[Console]::WindowHeight); Write-Output ('ORBIT'+'-HOST:'+ [Environment]::MachineName); Write-Output ('ORBIT'+'-TAIL'); exit 7`
	}
	writeTerminalCommand(t, session, command)
	exit := make(chan int, 1)
	go func() { code, _ := session.Wait(); exit <- code }()
	select {
	case code := <-exit:
		if code != 7 {
			t.Fatalf("exit code=%d output=%q", code, capture.output())
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("shell did not exit: %q", capture.output())
	}
	<-capture.done
	output := capture.output()
	if !strings.Contains(output, "ORBIT-CWD:"+directory) || !strings.Contains(output, "ORBIT-TAIL") {
		t.Fatalf("workspace or tail output missing: %q", output)
	}
	size := "ORBIT-SIZE:35 110"
	if runtime.GOOS == "windows" {
		size = "ORBIT-SIZE:110:35"
	}
	if !strings.Contains(output, size) {
		t.Fatalf("actual terminal size missing: %q", output)
	}
	host, err := os.Hostname()
	if err != nil || !strings.Contains(strings.ToLower(output), strings.ToLower("ORBIT-HOST:"+host)) {
		t.Fatalf("terminal did not run where Orbit runs: %q err=%v", output, err)
	}
}

func TestLocalTerminalExpandsHomeWorkspace(t *testing.T) {
	directory := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", directory)
	} else {
		t.Setenv("HOME", directory)
	}
	session, _ := startTestLocalTerminal(t, "~/workspace")
	if info, err := os.Stat(filepath.Join(directory, "workspace")); err != nil || !info.IsDir() {
		t.Fatalf("home workspace missing: %v", err)
	}
	_ = session.Close()
}

func TestLocalTerminalDisconnectKillsForegroundChild(t *testing.T) {
	for _, mode := range []string{"close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			directory := t.TempDir()
			session, cancel := startTestLocalTerminal(t, directory)
			capture := captureTerminal(session)
			pidPath := filepath.Join(directory, "child.pid")
			writeTerminalCommand(t, session, terminalChildCommand(pidPath))
			var pid int
			awaitLocalTerminal(t, func() bool {
				data, err := os.ReadFile(pidPath)
				if err != nil {
					return false
				}
				pid, err = strconv.Atoi(strings.TrimSpace(string(data)))
				return err == nil && pid > 0
			}, func() string { return "foreground child did not start: " + capture.output() })
			closed := make(chan struct{})
			go func() {
				if mode == "cancel" {
					cancel()
					_, _ = session.Wait()
				} else {
					_ = session.Close()
					_ = session.Close()
				}
				close(closed)
			}()
			select {
			case <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("local terminal did not close")
			}
			awaitLocalTerminal(t, func() bool { return !terminalTestProcessAlive(pid) }, func() string { return fmt.Sprintf("foreground child %d survived disconnect", pid) })
		})
	}
}

func TestLocalTerminalClosesWithUnreadLargeOutput(t *testing.T) {
	session, _ := startTestLocalTerminal(t, t.TempDir())
	capture := captureTerminal(session)
	command := `while :; do printf 'ORBIT-%s%01000d\n' FLOOD 0; done`
	if runtime.GOOS == "windows" {
		command = `while ($true) { [Console]::WriteLine('ORBIT'+'-FLOOD'+('X' * 1000)) }`
	}
	writeTerminalCommand(t, session, command)
	awaitLocalTerminal(t, func() bool { return strings.Count(capture.output(), "ORBIT-FLOOD") >= 4 }, func() string {
		return "shell did not produce flood output: " + capture.output()
	})
	close(capture.stop)
	<-capture.done
	// The real foreground command keeps producing output after its consumer exits.
	time.Sleep(100 * time.Millisecond)
	closed := make(chan struct{})
	go func() { _ = session.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("closing terminal with an absent output consumer blocked")
	}
	_, _ = io.Copy(io.Discard, session.Output())
}

func TestLocalTerminalInterruptReturnsToShell(t *testing.T) {
	directory := t.TempDir()
	session, _ := startTestLocalTerminal(t, directory)
	capture := captureTerminal(session)
	promptCommand := `PS1='ORBIT-''PROMPT> '`
	if runtime.GOOS == "windows" {
		promptCommand = `function prompt { 'ORBIT'+'-PROMPT> ' }`
	}
	writeTerminalCommand(t, session, promptCommand)
	awaitLocalTerminal(t, func() bool { return strings.Contains(capture.output(), "ORBIT-PROMPT> ") }, func() string {
		return "shell did not set the test prompt: " + capture.output()
	})
	pidPath := filepath.Join(directory, "child.pid")
	writeTerminalCommand(t, session, terminalChildCommand(pidPath))
	awaitLocalTerminal(t, func() bool {
		_, err := os.Stat(pidPath)
		return err == nil
	}, func() string { return "foreground command did not start: " + capture.output() })
	prompts := strings.Count(capture.output(), "ORBIT-PROMPT> ")
	if _, err := session.Input().Write([]byte{3}); err != nil {
		t.Fatal(err)
	}
	awaitLocalTerminal(t, func() bool { return strings.Count(capture.output(), "ORBIT-PROMPT> ") > prompts }, func() string {
		return "Ctrl-C did not return to the shell prompt: " + capture.output()
	})
	command := `printf 'ORBIT-%s\n' INTERRUPTED`
	if runtime.GOOS == "windows" {
		command = `Write-Output ('ORBIT'+'-INTERRUPTED')`
	}
	writeTerminalCommand(t, session, command)
	awaitLocalTerminal(t, func() bool {
		return strings.Contains(capture.output(), "ORBIT-INTERRUPTED")
	}, func() string { return "shell did not accept input after Ctrl-C: " + capture.output() })
}

func TestLocalTerminalDooDSocket(t *testing.T) {
	if os.Getenv("ORBIT_TEST_DOO_D") != "1" {
		t.Skip("requires an isolated Linux container with Docker CLI and socket")
	}
	session, _ := startTestLocalTerminal(t, t.TempDir())
	capture := captureTerminal(session)
	writeTerminalCommand(t, session, `[ -S /var/run/docker.sock ] && docker version --format '{{.Server.Version}}' >/dev/null && printf 'ORBIT-%s\n' DOOD`)
	awaitLocalTerminal(t, func() bool { return strings.Contains(capture.output(), "ORBIT-DOOD") }, func() string {
		return "container terminal could not use inherited Docker socket: " + capture.output()
	})
}
