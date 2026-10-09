package sshrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"golang.org/x/crypto/ssh"
)

var _ environmentport.TerminalRunner = (*Runtime)(nil)

type terminalSession struct {
	closeClient func()
	session     *ssh.Session
	stdin       io.WriteCloser
	output      *io.PipeReader
	outputWrite *io.PipeWriter
	once        sync.Once
	fingerprint string
}

func (r *Runtime) StartTerminal(ctx context.Context, target environmentport.Target, columns, rows int) (environmentport.TerminalSession, error) {
	if columns < 1 || rows < 1 {
		return nil, fmt.Errorf("terminal dimensions are invalid")
	}
	var fingerprint string
	client, err := r.dialSSHWithHostKey(ctx, target, &fingerprint)
	if err != nil {
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	closeClient := func() {
		stop()
		_ = client.Close()
	}
	session, err := client.NewSession()
	if err != nil {
		closeClient()
		return nil, fmt.Errorf("create terminal SSH session: %w", err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		closeClient()
		return nil, fmt.Errorf("open terminal stdin: %w", err)
	}
	output, outputWrite := io.Pipe()
	session.Stdout = outputWrite
	session.Stderr = outputWrite
	if err := session.RequestPty("xterm-256color", rows, columns, ssh.TerminalModes{ssh.ECHO: 1}); err != nil {
		_ = session.Close()
		closeClient()
		_ = output.Close()
		_ = outputWrite.Close()
		return nil, fmt.Errorf("request remote terminal PTY: %w", err)
	}
	if err := session.Shell(); err != nil {
		_ = session.Close()
		closeClient()
		_ = output.Close()
		_ = outputWrite.Close()
		return nil, fmt.Errorf("start remote terminal shell: %w", err)
	}
	return &terminalSession{closeClient: closeClient, session: session, stdin: stdin,
		output: output, outputWrite: outputWrite, fingerprint: fingerprint}, nil
}

func (s *terminalSession) HostKeyFingerprint() string { return s.fingerprint }

func (s *terminalSession) Input() io.WriteCloser { return s.stdin }
func (s *terminalSession) Output() io.Reader     { return s.output }

func (s *terminalSession) Resize(columns, rows int) error {
	return s.session.WindowChange(rows, columns)
}

func (s *terminalSession) Wait() (int, error) {
	err := s.session.Wait()
	_ = s.outputWrite.Close()
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitStatus(), nil
	}
	return 0, err
}

func (s *terminalSession) Close() error {
	var err error
	s.once.Do(func() {
		_ = s.outputWrite.Close()
		_ = s.output.Close()
		s.closeClient()
		err = s.session.Close()
		_ = s.stdin.Close()
	})
	return err
}
