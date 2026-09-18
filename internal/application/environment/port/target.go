package port

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// Target is the fully resolved, revision-pinned deployment target for one
// Project. PrivateKey is present only for SSH and is never persisted or logged.
type Target struct {
	Environment model.Environment
	PrivateKey  *environmentdto.DeploymentSSHPrivateKey
}

type TargetResolver interface {
	ResolveProjectTarget(ctx context.Context, projectId string) (Target, error)
}
