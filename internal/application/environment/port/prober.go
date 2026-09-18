package port

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// Prober verifies Docker Compose prerequisites for an Environment target.
// Probe is SSH-only; ProbeLocal checks the control-plane Docker host.
type Prober interface {
	Probe(context.Context, model.Environment, environmentdto.DeploymentSSHPrivateKey) (string, error)
	ProbeLocal(context.Context, model.Environment) error
}
