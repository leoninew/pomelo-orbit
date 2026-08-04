package runtimepath

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"
)

// ResolvePhysicalDataRoot resolves the backend logical data root to the host path
// Docker should use as a bind-mount source.
func ResolvePhysicalDataRoot(ctx context.Context, logicalDataRoot string) (string, error) {
	return ResolvePhysicalPath(ctx, logicalDataRoot)
}

// ResolvePhysicalPath resolves a path visible to the backend process to its host path
// when the backend itself runs inside Docker.
func ResolvePhysicalPath(ctx context.Context, logicalPath string) (string, error) {
	absolutePath, err := filepath.Abs(logicalPath)
	if err != nil {
		return "", fmt.Errorf("resolve physical path %s: %w", logicalPath, err)
	}
	containerId, ok := currentContainerId()
	if !ok {
		return absolutePath, nil
	}
	physicalPath, err := currentContainerPathSource(ctx, containerId, absolutePath)
	if err != nil {
		return "", err
	}
	return physicalPath, nil
}

func currentContainerId() (string, bool) {
	if containerId, ok := containerIdFromCgroup(); ok {
		return containerId, true
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		hostname, err := os.Hostname()
		if err == nil && strings.TrimSpace(hostname) != "" {
			return strings.TrimSpace(hostname), true
		}
	}
	return "", false
}

func containerIdFromCgroup() (string, bool) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, token := range strings.Split(line, "/") {
			containerId := normalizeContainerId(token)
			if containerId != "" {
				return containerId, true
			}
		}
	}
	return "", false
}

func normalizeContainerId(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "docker-")
	value = strings.TrimSuffix(value, ".scope")
	if len(value) < 12 {
		return ""
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ""
		}
	}
	return value
}

type dockerInspectMount struct {
	Type        string `json:"Type"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

func currentContainerPathSource(ctx context.Context, containerId string, containerPath string) (string, error) {
	output, err := exec.CommandContext(ctx, "docker", "inspect", containerId, "--format", "{{json .Mounts}}").Output()
	if err != nil {
		return "", fmt.Errorf("resolve physical data root: inspect current container %s: %w", containerId, err)
	}
	var mounts []dockerInspectMount
	if err := json.Unmarshal(output, &mounts); err != nil {
		return "", fmt.Errorf("resolve physical data root: parse current container mounts: %w", err)
	}
	wanted := cleanContainerPath(containerPath)
	matchedDestination := ""
	matchedSource := ""
	for _, mount := range mounts {
		destination := cleanContainerPath(mount.Destination)
		if _, ok := containerPathRelative(destination, wanted); !ok {
			continue
		}
		if len(destination) > len(matchedDestination) {
			matchedDestination = destination
			matchedSource = mount.Source
		}
	}
	if matchedDestination == "" || strings.TrimSpace(matchedSource) == "" {
		return "", fmt.Errorf("resolve physical path: container path %s is not mounted from the host; mount it explicitly", containerPath)
	}
	relativePath, _ := containerPathRelative(matchedDestination, wanted)
	if relativePath == "." {
		return matchedSource, nil
	}
	return filepath.Join(matchedSource, filepath.FromSlash(relativePath)), nil
}

func containerPathRelative(basePath string, targetPath string) (string, bool) {
	if targetPath == basePath {
		return ".", true
	}
	prefix := strings.TrimRight(basePath, "/") + "/"
	if !strings.HasPrefix(targetPath, prefix) {
		return "", false
	}
	return strings.TrimPrefix(targetPath, prefix), true
}

func cleanContainerPath(value string) string {
	return pathpkg.Clean(strings.ReplaceAll(filepath.ToSlash(value), "\\", "/"))
}
