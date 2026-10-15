package port

import (
	"context"
	"io"
)

type Command struct {
	Argv    []string
	Workdir string
	Stdin   []byte
}

// Sent includes an uncertain exec request acknowledgement: callers must not retry.
type CommandOutcome struct {
	Sent     bool
	ExitCode *int32
}

type CommandSession interface {
	HostKeyFingerprint() string
	Run(context.Context, Command, io.Writer, io.Writer) (CommandOutcome, error)
	Close() error
}

type CommandRunner interface {
	Open(context.Context, Target) (CommandSession, error)
}
