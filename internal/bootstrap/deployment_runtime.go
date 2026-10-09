package bootstrap

import (
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	targetrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/target"
)

func newDeploymentRuntime(resolvePath localrunner.PhysicalPathResolver) (*localrunner.Runtime, targetrunner.Runtime) {
	localRuntime := localrunner.NewRuntime(resolvePath)
	return localRuntime, targetrunner.New(localRuntime, sshrunner.NewRuntime())
}
