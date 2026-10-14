package dockerrunner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

type LogRunner interface {
	QueryRunner
	StreamAtEnvironmentRoot(context.Context, environmentport.Target, io.Writer, string, ...string) error
}

type ContainerLogs struct{ runner LogRunner }

func NewContainerLogs(runner LogRunner) ContainerLogs { return ContainerLogs{runner: runner} }

func (l ContainerLogs) ContainerIds(ctx context.Context, target environmentport.Target, project, component string) ([]string, error) {
	args := []string{"ps", "--all", "--filter", "label=com.docker.compose.project=" + project, "--format", "{{json .}}"}
	if component != "" {
		args = append(args, "--filter", "label=com.docker.compose.service="+component)
	}
	output, err := l.runner.QueryAtEnvironmentRoot(ctx, target, "docker", args...)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindUnavailable, "Failed to inspect log containers", err)
	}
	ids := make([]string, 0)
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var container struct{ Id string }
		if err := decoder.Decode(&container); err != nil {
			if errors.Is(err, io.EOF) {
				return ids, nil
			}
			return nil, apperror.Wrap(apperror.KindInternal, "Invalid container observation", err)
		}
		ids = append(ids, container.Id)
	}
}

func (l ContainerLogs) Follow(ctx context.Context, target environmentport.Target, request deploymentport.ContainerLogRequest, output io.Writer) error {
	// Existing containers can be read without a Compose file or service workspace.
	args := []string{"compose", "-p", request.Project, "logs", "--follow", "--timestamps", "--no-color"}
	if request.Since != "" {
		args = append(args, "--since", request.Since)
	} else {
		args = append(args, "--tail", strconv.Itoa(request.Tail))
	}
	if request.Component != "" {
		args = append(args, request.Component)
	}
	return l.runner.StreamAtEnvironmentRoot(ctx, target, output, "docker", args...)
}
