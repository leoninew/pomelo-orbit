package bootstrap

import (
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	targetrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/target"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// Compose the existing repositories into the worker's execution port.
type deploymentExecutionStore struct {
	repository.ApplicationStore
	repository.ServiceStore
	repository.DeploymentStore
}

func newDeploymentRuntime(resolvePath localrunner.PhysicalPathResolver) (*localrunner.Runtime, targetrunner.Runtime) {
	localRuntime := localrunner.NewRuntime(resolvePath)
	return localRuntime, targetrunner.New(localRuntime, sshrunner.NewRuntime())
}
