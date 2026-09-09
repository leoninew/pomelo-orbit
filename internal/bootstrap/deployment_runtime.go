package bootstrap

import (
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	targetrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/target"
)

func newDeploymentRuntime(resolvePath localrunner.PhysicalPathResolver) (*localrunner.Runtime, deploymentport.Runtime) {
	localRuntime := localrunner.NewRuntime(resolvePath)
	return localRuntime, targetrunner.New(localRuntime, sshrunner.NewRuntime())
}
