//go:build !windows

package localrunner

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/aymanbagabas/go-pty"
	"golang.org/x/sys/unix"
)

type unixTerminalProcess struct {
	command *pty.Cmd
	pty     pty.UnixPty
	once    sync.Once
}

func startTerminalProcess(terminal pty.Pty, directory string) (terminalProcess, error) {
	command := terminal.Command("/bin/sh", "-i")
	command.Dir = directory
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "TERM=") {
			command.Env = append(command.Env, value)
		}
	}
	command.Env = append(command.Env, "TERM=xterm-256color")
	if err := command.Start(); err != nil {
		return nil, err
	}
	terminalUnix := terminal.(pty.UnixPty)
	_ = terminalUnix.Slave().Close()
	return &unixTerminalProcess{command: command, pty: terminalUnix}, nil
}

func (p *unixTerminalProcess) wait() (int, error) {
	err := p.command.Wait()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		err = nil
	}
	if p.command.ProcessState == nil {
		return 1, err
	}
	return p.command.ProcessState.ExitCode(), err
}

func (p *unixTerminalProcess) stop() {
	p.once.Do(func() {
		// Interactive job control can put the foreground command in another group.
		_ = p.pty.Control(func(fd uintptr) {
			foreground, err := unix.IoctlGetInt(int(fd), unix.TIOCGPGRP)
			if err == nil && foreground > 0 && foreground != p.command.Process.Pid {
				_ = unix.Kill(-foreground, unix.SIGKILL)
			}
		})
		_ = unix.Kill(-p.command.Process.Pid, unix.SIGKILL)
	})
}

func terminalOutputReader(terminal pty.Pty) (io.ReadCloser, error) {
	return terminal.(pty.UnixPty).Master(), nil
}

func finishTerminalOutput(*localTerminal) {}

func terminalOutputEnded(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, unix.EIO) || errors.Is(err, os.ErrClosed)
}
