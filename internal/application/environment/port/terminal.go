package port

import (
	"context"
	"io"
)

type TerminalSession interface {
	Input() io.WriteCloser
	Output() io.Reader
	Resize(columns, rows int) error
	Wait() (int, error)
	Close() error
}

type TerminalRunner interface {
	StartTerminal(ctx context.Context, target Target, columns, rows int) (TerminalSession, error)
}

// SSH terminals report the host key from their authenticated connection.
type TerminalHostKey interface {
	HostKeyFingerprint() string
}
