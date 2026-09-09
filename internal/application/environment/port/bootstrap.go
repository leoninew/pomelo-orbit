package port

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// BootstrapAuth is an ephemeral SSH authentication method used only to
// initialize a Linux deployment host. It must never be persisted or logged.
type BootstrapAuth struct {
	Username             string
	Password             string
	PrivateKey           string
	PrivateKeyPassphrase string
}

// Bootstrapper configures a reachable Linux SSH host so the generated
// deployment key can be used for subsequent non-interactive deployments.
type Bootstrapper interface {
	Bootstrap(context.Context, model.Environment, string, BootstrapAuth) error
}
