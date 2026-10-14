package dockerrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

func (i Inventory) RunningBindMounts(ctx context.Context, target environmentport.Target, serviceCode string) ([]deploymentport.RunningBindMount, error) {
	output, err := i.runner.QueryAtEnvironmentRoot(ctx, target, "docker", "ps", "--filter", "label=com.docker.compose.project="+serviceCode, "--filter", "status=running", "--format", "{{.ID}}")
	if err != nil {
		return nil, fmt.Errorf("list running containers for service %s: %w", serviceCode, err)
	}
	ids := strings.Fields(output)
	if len(ids) == 0 {
		return nil, nil
	}
	// Native JSON avoids PowerShell stripping quotes from template keys.
	output, err = i.runner.QueryAtEnvironmentRoot(ctx, target, "docker", append([]string{"inspect", "--type", "container"}, ids...)...)
	if err != nil {
		return nil, fmt.Errorf("inspect running mounts for service %s: %w", serviceCode, err)
	}
	var containers []struct {
		Config struct{ Labels map[string]string }
		State  struct{ Running bool }
		Mounts []struct{ Type, Source, Destination string }
	}
	if err := json.Unmarshal([]byte(output), &containers); err != nil {
		return nil, fmt.Errorf("parse running mounts for service %s: %w", serviceCode, err)
	}
	var result []deploymentport.RunningBindMount
	for _, container := range containers {
		if !container.State.Running {
			continue
		}
		var sharedTargets []string
		if value := container.Config.Labels["io.pomelo-orbit.shared-mounts"]; value != "" {
			if err := json.Unmarshal([]byte(value), &sharedTargets); err != nil {
				return nil, fmt.Errorf("parse shared mounts for service %s: %w", serviceCode, err)
			}
		}
		for _, mount := range container.Mounts {
			if mount.Type != "bind" {
				continue
			}
			shared := false
			for _, target := range sharedTargets {
				if mountTargetKey(target) == mountTargetKey(mount.Destination) {
					shared = true
					break
				}
			}
			result = append(result, deploymentport.RunningBindMount{Source: mount.Source, Shared: shared})
		}
	}
	return result, nil
}

func mountTargetKey(value string) string {
	if len(value) > 1 && value[1] == ':' {
		value = strings.ToLower(strings.ReplaceAll(value, "\\", "/"))
	}
	for _, prefix := range []string{"/run/desktop/mnt/host/", "/host_mnt/"} {
		if suffix, ok := strings.CutPrefix(value, prefix); ok && len(suffix) >= 2 && suffix[1] == '/' && (suffix[0] >= 'a' && suffix[0] <= 'z' || suffix[0] >= 'A' && suffix[0] <= 'Z') {
			value = strings.ToLower(suffix[:1] + ":" + suffix[1:])
			break
		}
	}
	return path.Clean(value)
}
