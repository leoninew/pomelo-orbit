package cisvc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gitee.com/leoninew/pomelo-orbit/internal/runtimepath"
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
	return newCIWorkspaceWithResolver(dataRoot, runtimepath.ResolvePhysicalDataRoot)
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
