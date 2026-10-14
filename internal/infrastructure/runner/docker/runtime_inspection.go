package dockerrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

type RuntimeQueryRunner interface {
	Query(context.Context, environmentport.Target, deploymentport.ServiceLocation, string, ...string) (string, error)
}

type RuntimeInspector struct{ runner RuntimeQueryRunner }

func NewRuntimeInspector(runner RuntimeQueryRunner) RuntimeInspector {
	return RuntimeInspector{runner: runner}
}

func (r RuntimeInspector) query(ctx context.Context, scope deploymentport.RuntimeScope, args ...string) (string, error) {
	output, err := r.runner.Query(ctx, scope.Target, scope.Location, "docker", args...)
	if err != nil {
		message := output
		if message == "" {
			message = err.Error()
		}
		return output, apperror.New(apperror.KindInternal, message)
	}
	return output, nil
}

func (r RuntimeInspector) Containers(ctx context.Context, scope deploymentport.RuntimeScope) (deploymentport.ContainerObservation, error) {
	output, err := r.query(ctx, scope, "compose", "-p", scope.Project, "-f", "docker-compose.yml", "ps", "--format", "json")
	if err != nil {
		return deploymentport.ContainerObservation{}, err
	}
	containers, err := parseComposePsOutput(output)
	if err != nil {
		return deploymentport.ContainerObservation{}, apperror.Wrap(apperror.KindInternal, "Failed to parse compose status", err)
	}
	var raw any
	if strings.TrimSpace(output) != "" && json.Unmarshal([]byte(output), &raw) != nil {
		raw = output
	}
	return deploymentport.ContainerObservation{Containers: containers, Raw: raw}, nil
}

func (r RuntimeInspector) Config(ctx context.Context, scope deploymentport.RuntimeScope) (string, error) {
	return r.query(ctx, scope, "compose", "-f", "docker-compose.yml", "config")
}

func (r RuntimeInspector) Logs(ctx context.Context, scope deploymentport.RuntimeScope, tail int, since string, services []string) (string, error) {
	args := []string{"compose", "-p", scope.Project, "-f", "docker-compose.yml", "logs"}
	if since != "" {
		args = append(args, "--since", since)
	} else {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	args = append(args, services...)
	return r.query(ctx, scope, args...)
}

func (r RuntimeInspector) InspectContainer(ctx context.Context, scope deploymentport.RuntimeScope, id string) (deploymentport.ContainerInspection, error) {
	output, err := r.query(ctx, scope, "inspect", id)
	if err != nil {
		return deploymentport.ContainerInspection{}, err
	}
	var raw any
	if json.Unmarshal([]byte(output), &raw) != nil {
		return deploymentport.ContainerInspection{}, apperror.New(apperror.KindInternal, "invalid container inspect output")
	}
	result := deploymentport.ContainerInspection{Raw: raw}
	// Raw is retained for the diagnostic API; business callers consume these facts.
	items, _ := raw.([]any)
	if len(items) == 0 {
		return result, nil
	}
	item, _ := items[0].(map[string]any)
	state, _ := item["State"].(map[string]any)
	result.State, _ = state["Status"].(string)
	health, _ := state["Health"].(map[string]any)
	result.Health, _ = health["Status"].(string)
	if restarts, ok := item["RestartCount"].(float64); ok {
		result.RestartCount = int(restarts)
	}
	settings, _ := item["NetworkSettings"].(map[string]any)
	networks, _ := settings["Networks"].(map[string]any)
	for name := range networks {
		result.Networks = append(result.Networks, name)
	}
	return result, nil
}

func (r RuntimeInspector) InspectNetwork(ctx context.Context, scope deploymentport.RuntimeScope, name string) (any, error) {
	output, err := r.query(ctx, scope, "network", "inspect", name)
	if err != nil {
		return nil, err
	}
	var raw any
	if json.Unmarshal([]byte(output), &raw) != nil {
		return nil, apperror.New(apperror.KindInternal, "invalid network inspect output")
	}
	return raw, nil
}

func (r RuntimeInspector) HTTPProbe(ctx context.Context, scope deploymentport.RuntimeScope, component string, port int, path string) (string, error) {
	return r.query(ctx, scope, "compose", "-p", scope.Project, "-f", "docker-compose.yml", "exec", "-T", component, "curl", "-fsS", "--max-time", "10", fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
}

func (r RuntimeInspector) Version(ctx context.Context, scope deploymentport.RuntimeScope) (string, error) {
	output, err := r.query(ctx, scope, "version", "--format", "json")
	return strings.TrimSpace(output), err
}
