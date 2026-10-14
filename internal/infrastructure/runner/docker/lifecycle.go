package dockerrunner

import (
	"context"
	"io"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type LifecycleRunner interface {
	Run(context.Context, environmentport.Target, deploymentport.ServiceLocation, io.Writer, string, ...string) error
}

type Lifecycle struct{ runner LifecycleRunner }

func NewLifecycle(runner LifecycleRunner) Lifecycle { return Lifecycle{runner: runner} }

type Commands struct{}

func deployArgs(project string, settings deploymentport.DeploySettings) []string {
	args := []string{"compose", "-p", project, "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", settings.PullPolicy}
	if settings.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	return args
}

func stopArgs(project string, removeVolumes bool) []string {
	args := []string{"compose", "-p", project, "-f", "docker-compose.yml", "down"}
	if removeVolumes {
		args = append(args, "-v")
	}
	return args
}

func (Commands) DeployDescription(project string, settings deploymentport.DeploySettings) string {
	return "docker " + strings.Join(deployArgs(project, settings), " ")
}

func (Commands) StopDescription(project string, removeVolumes bool) string {
	return "docker " + strings.Join(stopArgs(project, removeVolumes), " ")
}

func (l Lifecycle) Deploy(ctx context.Context, scope deploymentport.RuntimeScope, settings deploymentport.DeploySettings, output io.Writer) error {
	return l.runner.Run(ctx, scope.Target, scope.Location, output, "docker", deployArgs(scope.Project, settings)...)
}

func (l Lifecycle) Stop(ctx context.Context, scope deploymentport.RuntimeScope, removeVolumes bool, output io.Writer) error {
	return l.runner.Run(ctx, scope.Target, scope.Location, output, "docker", stopArgs(scope.Project, removeVolumes)...)
}
