package dockerrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type QueryRunner interface {
	QueryAtEnvironmentRoot(context.Context, environmentport.Target, string, ...string) (string, error)
}

type Inventory struct{ runner QueryRunner }

func NewInventory(runner QueryRunner) Inventory { return Inventory{runner: runner} }

func (i Inventory) Containers(ctx context.Context, target environmentport.Target) (map[string][]deploymentport.ContainerState, error) {
	output, err := i.runner.QueryAtEnvironmentRoot(ctx, target, "docker", "ps", "--all", "--filter", "label=com.docker.compose.project", "--format", `{{json .}}`)
	if err != nil {
		if diagnostic := strings.TrimSpace(output); diagnostic != "" {
			const maxDiagnosticBytes = 8192
			if len(diagnostic) > maxDiagnosticBytes {
				diagnostic = "...\n" + diagnostic[len(diagnostic)-maxDiagnosticBytes:]
			}
			err = fmt.Errorf("%w\n%s", err, diagnostic)
		}
		return nil, err
	}
	result := make(map[string][]deploymentport.ContainerState)
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var container struct{ Labels, State, Status string }
		if err := decoder.Decode(&container); err != nil {
			if errors.Is(err, io.EOF) {
				return result, nil
			}
			return nil, fmt.Errorf("decode environment container states: %w", err)
		}
		project, oneoff := "", ""
		for _, label := range strings.Split(container.Labels, ",") {
			key, value, _ := strings.Cut(label, "=")
			switch key {
			case "com.docker.compose.project":
				project = value
			case "com.docker.compose.oneoff":
				oneoff = value
			}
		}
		if project == "" || strings.EqualFold(oneoff, "true") {
			continue
		}
		result[project] = append(result[project], deploymentport.ContainerState{State: container.State, Unhealthy: strings.Contains(container.Status, "(unhealthy)")})
	}
}
