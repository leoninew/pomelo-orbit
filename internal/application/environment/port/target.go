package port

import (
	"context"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// SSHTarget is the fully resolved, revision-pinned deployment target for one
// Project. The private key is transient and must never be persisted or logged.
type SSHTarget struct {
	Environment model.Environment
	PrivateKey  credentialdto.DeploymentSSHPrivateKey
}

type TargetResolver interface {
	ResolveProjectTarget(ctx context.Context, projectID string) (SSHTarget, error)
}
