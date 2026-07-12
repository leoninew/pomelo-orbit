package ciworkspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ciport "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/port"
)

type PhysicalDataRootResolver func(ctx context.Context, logicalDataRoot string) (string, error)

type Workspace struct {
	logicalDataRoot string
	resolver        PhysicalDataRootResolver

	physicalOnce     sync.Once
	physicalDataRoot string
	physicalErr      error
}

func NewWithResolver(dataRoot string, resolver PhysicalDataRootResolver) *Workspace {
	return &Workspace{logicalDataRoot: filepath.Clean(dataRoot), resolver: resolver}
}

func (w *Workspace) CreateRunDirectories(projectCode string, runId string) error {
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

func (w *Workspace) WorkspacePath(projectCode string) string {
	return filepath.Join(w.logicalDataRoot, "ci", projectCode, "workspace")
}

func (w *Workspace) ArtifactsPath(runId string) string {
	return filepath.Join(w.logicalDataRoot, "ci", "runs", runId, "artifacts")
}

func (w *Workspace) ArtifactExists(runID string, artifactPath string) (bool, error) {
	_, err := os.Stat(filepath.Join(w.ArtifactsPath(runID), artifactPath))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (w *Workspace) StageLogPath(runId string, stageRunId string) string {
	return filepath.Join(w.logicalDataRoot, "ci", "runs", runId, "stages", stageRunId+".log")
}

func (w *Workspace) DockerStageMounts(ctx context.Context, projectCode string, runID string) ([]ciport.VolumeMount, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return nil, err
	}
	return []ciport.VolumeMount{
		{HostPath: filepath.Join(physicalDataRoot, "ci", projectCode, "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		{HostPath: filepath.Join(physicalDataRoot, "ci", "runs", runID, "artifacts"), ContainerPath: "/artifacts", Mode: "rw"},
	}, nil
}

func (w *Workspace) PhysicalDataRoot(ctx context.Context) (string, error) {
	w.physicalOnce.Do(func() {
		w.physicalDataRoot, w.physicalErr = w.resolver(ctx, w.logicalDataRoot)
	})
	return w.physicalDataRoot, w.physicalErr
}
