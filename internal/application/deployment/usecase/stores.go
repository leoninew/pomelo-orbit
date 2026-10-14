package deploymentsvc

import "github.com/leoninew/pomelo-orbit/internal/repository"

// stores composes existing repositories into the command persistence port.
type stores struct {
	repository.ProjectReader
	repository.ApplicationStore
	repository.ServiceStore
	repository.DeploymentStore
}
