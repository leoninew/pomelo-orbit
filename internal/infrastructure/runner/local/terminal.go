package localrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/aymanbagabas/go-pty"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/common/workspacepath"
)

var _ environmentport.TerminalRunner = (*Runtime)(nil)
var _ environmentport.TerminalSession = (*localTerminal)(nil)

type terminalProcess interface {
	wait() (int, error)
	stop()
}

type localTerminal struct {
	pty       pty.Pty
	process   terminalProcess
	reader    io.ReadCloser
	output    *io.PipeReader
	writer    *io.PipeWriter
	pumpDone  chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	ptyMu     sync.Mutex
	ptyClosed bool
	exitCode  int
	waitErr   error
}

func (r *Runtime) StartTerminal(ctx context.Context, target environmentport.Target, columns, rows int) (environmentport.TerminalSession, error) {
	if !target.Environment.IsLocal() {
		return nil, errors.New("local terminal received a non-local environment")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !validTerminalSize(columns, rows) {
		return nil, errors.New("terminal dimensions are invalid")
	}
	directory, err := workspacepath.ExpandLocalHomePath(target.Environment.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(directory) {
		return nil, errors.New("local terminal workspace must be absolute")
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("create local terminal workspace: %w", err)
	}
	terminal, err := pty.New()
	if err != nil {
		return nil, fmt.Errorf("create local terminal PTY: %w", err)
	}
	if err := terminal.Resize(columns, rows); err != nil {
		_ = terminal.Close()
		return nil, err
	}
	reader, err := terminalOutputReader(terminal)
	if err != nil {
		_ = terminal.Close()
		return nil, err
	}
	output, writer := io.Pipe()
	s := &localTerminal{pty: terminal, reader: reader, output: output, writer: writer,
		pumpDone: make(chan struct{}), done: make(chan struct{})}
	go s.pumpOutput()
	process, err := startTerminalProcess(terminal, directory)
	if err != nil {
		_ = output.Close()
		s.closePty()
		_ = reader.Close()
		<-s.pumpDone
		return nil, fmt.Errorf("start local terminal shell: %w", err)
	}
	s.process = process
	go func() {
		s.exitCode, s.waitErr = process.wait()
		process.stop()
		finishTerminalOutput(s)
		<-s.pumpDone
		s.closePty()
		_ = reader.Close()
		close(s.done)
	}()
	go func() {
		select {
		case <-ctx.Done():
			_ = s.Close()
		case <-s.done:
		}
	}()
	return s, nil
}

func (s *localTerminal) pumpOutput() {
	defer close(s.pumpDone)
	defer func() { _ = s.writer.Close() }()
	buffer := make([]byte, 16*1024)
	discard := false
	for {
		n, err := s.reader.Read(buffer)
		if n > 0 && !discard {
			if _, writeErr := s.writer.Write(buffer[:n]); writeErr != nil {
				// Keep draining ConPTY when the WebSocket consumer has gone away.
				discard = true
			}
		}
		if err != nil {
			if !terminalOutputEnded(err) {
				_ = s.writer.CloseWithError(err)
			}
			return
		}
	}
}

type localTerminalInput struct{ session *localTerminal }

func (w localTerminalInput) Write(data []byte) (int, error) { return w.session.pty.Write(data) }
func (w localTerminalInput) Close() error                   { return w.session.Close() }
func (s *localTerminal) Input() io.WriteCloser              { return localTerminalInput{session: s} }
func (s *localTerminal) Output() io.Reader                  { return s.output }

func validTerminalSize(columns, rows int) bool {
	return columns >= 1 && columns <= 500 && rows >= 1 && rows <= 200
}

func (s *localTerminal) Resize(columns, rows int) error {
	if !validTerminalSize(columns, rows) {
		return errors.New("terminal dimensions are invalid")
	}
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	if s.ptyClosed {
		return os.ErrClosed
	}
	return s.pty.Resize(columns, rows)
}

func (s *localTerminal) Wait() (int, error) {
	<-s.done
	return s.exitCode, s.waitErr
}

func (s *localTerminal) closePty() {
	s.ptyMu.Lock()
	defer s.ptyMu.Unlock()
	if !s.ptyClosed {
		s.ptyClosed = true
		_ = s.pty.Close()
	}
}

func (s *localTerminal) Close() error {
	s.closeOnce.Do(func() {
		_ = s.output.Close()
		s.process.stop()
		s.closePty()
		_ = s.reader.Close()
	})
	<-s.done
	return nil
}
