package port

import (
	"context"
	"io"
)

type DeploySettings struct {
	PullPolicy    string
	ForceRecreate bool
}

// LifecycleCommands describes the same operations that ServiceLifecycle executes.
// Descriptions are persisted for diagnostics, never parsed back for execution.
type LifecycleCommands interface {
	DeployDescription(string, DeploySettings) string
	StopDescription(string, bool) string
}

// ServiceLifecycle borrows the operation context and output; the caller owns
// the runtime session and closes the output after execution.
type ServiceLifecycle interface {
	Deploy(context.Context, RuntimeScope, DeploySettings, io.Writer) error
	Stop(context.Context, RuntimeScope, bool, io.Writer) error
}
