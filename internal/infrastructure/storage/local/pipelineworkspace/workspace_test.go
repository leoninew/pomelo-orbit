package pipelineworkspace

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceResolvesRelativeDataRootToAbsolutePhysicalMounts(t *testing.T) {
	workspace := NewWithResolver("data", resolveAbsolutePhysicalDataRoot)
	mounts, err := workspace.DockerStageMounts(context.Background(), "repo", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 2 {
		t.Fatalf("expected workspace and artifacts mounts, got %d", len(mounts))
	}
	for _, mount := range mounts {
		if !filepath.IsAbs(mount.HostPath) {
			t.Fatalf("expected absolute host path, got %q", mount.HostPath)
		}
		if strings.HasPrefix(filepath.ToSlash(mount.HostPath), "data/"+pipelineDataDir+"/") {
			t.Fatalf("expected physical host path, got relative data path %q", mount.HostPath)
		}
	}
	if mounts[0].ContainerPath != "/workspace" || mounts[1].ContainerPath != "/artifacts" {
		t.Fatalf("unexpected container paths: %+v", mounts)
	}
}

func TestWorkspaceKeepsAbsolutePhysicalDataRootSemantics(t *testing.T) {
	dataRoot := t.TempDir()
	workspace := NewWithResolver(dataRoot, resolveAbsolutePhysicalDataRoot)
	mounts, err := workspace.DockerStageMounts(context.Background(), "repo", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	wantWorkspace := filepath.Join(dataRoot, pipelineDataDir, "repo", "workspace")
	wantArtifacts := filepath.Join(dataRoot, pipelineDataDir, "runs", "run-1", "artifacts")
	if mounts[0].HostPath != wantWorkspace || mounts[1].HostPath != wantArtifacts {
		t.Fatalf("unexpected mounts: %+v", mounts)
	}
}

func TestWorkspaceUsesResolvedPhysicalDataRootForDockerMounts(t *testing.T) {
	logicalRoot := filepath.Join(string(filepath.Separator), "app", "data")
	physicalRoot := filepath.Join(t.TempDir(), "data")
	workspace := NewWithResolver(logicalRoot, func(ctx context.Context, logicalDataRoot string) (string, error) {
		if logicalDataRoot != filepath.Clean(logicalRoot) {
			t.Fatalf("unexpected logical data root: %s", logicalDataRoot)
		}
		return physicalRoot, nil
	})
	mounts, err := workspace.DockerStageMounts(context.Background(), "repo", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(mounts[0].HostPath, physicalRoot) || !strings.HasPrefix(mounts[1].HostPath, physicalRoot) {
		t.Fatalf("expected physical root %q, got %+v", physicalRoot, mounts)
	}
}

func TestRemoveRunFilesRemovesOnlyRunDirectory(t *testing.T) {
	dataRoot := t.TempDir()
	workspace := NewWithResolver(dataRoot, nil)
	stageLog := workspace.StageLogPath("run-1", "stage-run-1")
	artifact := filepath.Join(workspace.ArtifactsPath("run-1"), "output.tar")
	projectWorkspaceFile := filepath.Join(workspace.WorkspacePath("repo-1"), "source.txt")
	for path, content := range map[string]string{
		stageLog:             "stage log",
		artifact:             "artifact",
		projectWorkspaceFile: "source",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	if err := workspace.RemoveRunFiles("run-1"); err != nil {
		t.Fatalf("RemoveRunFiles returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(workspace.ArtifactsPath("run-1"))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("run directory stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(projectWorkspaceFile); err != nil {
		t.Fatalf("project workspace must remain: %v", err)
	}
}

func resolveAbsolutePhysicalDataRoot(ctx context.Context, logicalDataRoot string) (string, error) {
	return filepath.Abs(logicalDataRoot)
}
