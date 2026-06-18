package cisvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
)

type physicalDataRootResolver func(ctx context.Context, logicalDataRoot string) (string, error)

type CIWorkspace struct {
	logicalDataRoot string
	resolver        physicalDataRootResolver

	physicalOnce     sync.Once
	physicalDataRoot string
	physicalErr      error
}

func NewCIWorkspace(dataRoot string) *CIWorkspace {
	return newCIWorkspaceWithResolver(dataRoot, defaultPhysicalDataRoot)
}

func newCIWorkspaceWithResolver(dataRoot string, resolver physicalDataRootResolver) *CIWorkspace {
	return &CIWorkspace{logicalDataRoot: filepath.Clean(dataRoot), resolver: resolver}
}

func (w *CIWorkspace) CreateRunDirectories(projectCode string, runId string) error {
	paths := []string{
		w.WorkspacePath(projectCode),
		w.ArtifactsPath(runId),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create workspace path %s: %w", path, err)
		}
	}
	return nil
}

func (w *CIWorkspace) WorkspacePath(projectCode string) string {
	return filepath.Join(w.logicalDataRoot, "ci", projectCode, "workspace")
}

func (w *CIWorkspace) ArtifactsPath(runId string) string {
	return filepath.Join(w.logicalDataRoot, "ci", "runs", runId, "artifacts")
}

func (w *CIWorkspace) StageLogPath(runId string, stageRunId string) string {
	return filepath.Join(w.logicalDataRoot, "ci", "runs", runId, "stages", stageRunId+".log")
}

func (w *CIWorkspace) DockerStageMounts(ctx context.Context, projectCode string, runId string) ([]VolumeMount, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return nil, err
	}
	return []VolumeMount{
		{HostPath: filepath.Join(physicalDataRoot, "ci", projectCode, "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		{HostPath: filepath.Join(physicalDataRoot, "ci", "runs", runId, "artifacts"), ContainerPath: "/artifacts", Mode: "rw"},
	}, nil
}

func (w *CIWorkspace) PhysicalDataRoot(ctx context.Context) (string, error) {
	w.physicalOnce.Do(func() {
		w.physicalDataRoot, w.physicalErr = w.resolver(ctx, w.logicalDataRoot)
	})
	return w.physicalDataRoot, w.physicalErr
}

func defaultPhysicalDataRoot(ctx context.Context, logicalDataRoot string) (string, error) {
	absoluteDataRoot, err := filepath.Abs(logicalDataRoot)
	if err != nil {
		return "", fmt.Errorf("resolve CI data root %s: %w", logicalDataRoot, err)
	}
	containerId, ok := currentContainerId()
	if !ok {
		return absoluteDataRoot, nil
	}
	physicalDataRoot, err := currentContainerMountSource(ctx, containerId, absoluteDataRoot)
	if err != nil {
		return "", err
	}
	return physicalDataRoot, nil
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

func currentContainerMountSource(ctx context.Context, containerId string, containerDataRoot string) (string, error) {
	output, err := exec.CommandContext(ctx, "docker", "inspect", containerId, "--format", "{{json .Mounts}}").Output()
	if err != nil {
		return "", fmt.Errorf("resolve CI physical data root: inspect current container %s: %w", containerId, err)
	}
	var mounts []dockerInspectMount
	if err := json.Unmarshal(output, &mounts); err != nil {
		return "", fmt.Errorf("resolve CI physical data root: parse current container mounts: %w", err)
	}
	wanted := cleanContainerPath(containerDataRoot)
	for _, mount := range mounts {
		if cleanContainerPath(mount.Destination) == wanted {
			if strings.TrimSpace(mount.Source) == "" {
				return "", fmt.Errorf("resolve CI physical data root: container data root %s has empty host source", containerDataRoot)
			}
			return mount.Source, nil
		}
	}
	return "", fmt.Errorf("resolve CI physical data root: container data root %s is not mounted from the host; mount the data directory explicitly", containerDataRoot)
}

func cleanContainerPath(value string) string {
	return pathpkg.Clean(filepath.ToSlash(value))
}
