package pipelineworkspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
)

type PhysicalWorkspaceResolver func(ctx context.Context, logicalWorkspaceRoot string) (string, error)

type Workspace struct {
	logicalWorkspaceRoot string
	resolver             PhysicalWorkspaceResolver

	physicalOnce          sync.Once
	physicalWorkspaceRoot string
	physicalErr           error
}

func NewWithResolver(workspaceRoot string, resolver PhysicalWorkspaceResolver) *Workspace {
	return &Workspace{logicalWorkspaceRoot: filepath.Clean(workspaceRoot), resolver: resolver}
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
	return filepath.Join(w.logicalWorkspaceRoot, projectCode, "workspace")
}

func (w *Workspace) ArtifactsPath(runId string) string {
	return filepath.Join(w.logicalWorkspaceRoot, "runs", runId, "artifacts")
}

func (w *Workspace) ArtifactExists(runId string, artifactPath string) (bool, error) {
	path, err := w.artifactPath(runId, artifactPath)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (w *Workspace) artifactPath(runId string, artifactPath string) (string, error) {
	clean := filepath.Clean(artifactPath)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid artifact path %s", artifactPath)
	}
	return filepath.Join(w.ArtifactsPath(runId), clean), nil
}

func (w *Workspace) StageLogPath(runId string, pipelineStageRunId string) string {
	return filepath.Join(w.logicalWorkspaceRoot, "runs", runId, "stages", pipelineStageRunId+".log")
}

func (w *Workspace) RemoveRunFiles(runId string) error {
	runPath := filepath.Dir(w.ArtifactsPath(runId))
	if _, err := os.Lstat(runPath); err != nil {
		return err
	}
	return os.RemoveAll(runPath)
}

func (w *Workspace) DockerStageMounts(ctx context.Context, projectCode string, runId string) ([]pipelinerunport.VolumeMount, error) {
	physicalWorkspaceRoot, err := w.PhysicalWorkspaceRoot(ctx)
	if err != nil {
		return nil, err
	}
	return []pipelinerunport.VolumeMount{
		{HostPath: filepath.Join(physicalWorkspaceRoot, projectCode, "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		{HostPath: filepath.Join(physicalWorkspaceRoot, "runs", runId, "artifacts"), ContainerPath: "/artifacts", Mode: "rw"},
	}, nil
}

func (w *Workspace) PhysicalWorkspaceRoot(ctx context.Context) (string, error) {
	w.physicalOnce.Do(func() {
		if w.resolver == nil {
			w.physicalWorkspaceRoot = w.logicalWorkspaceRoot
			return
		}
		w.physicalWorkspaceRoot, w.physicalErr = w.resolver(ctx, w.logicalWorkspaceRoot)
	})
	return w.physicalWorkspaceRoot, w.physicalErr
}
