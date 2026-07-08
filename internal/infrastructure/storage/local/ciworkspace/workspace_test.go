package ciworkspace

import (
	"context"
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
		if strings.HasPrefix(filepath.ToSlash(mount.HostPath), "data/ci/") {
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
	wantWorkspace := filepath.Join(dataRoot, "ci", "repo", "workspace")
	wantArtifacts := filepath.Join(dataRoot, "ci", "runs", "run-1", "artifacts")
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

func resolveAbsolutePhysicalDataRoot(ctx context.Context, logicalDataRoot string) (string, error) {
	return filepath.Abs(logicalDataRoot)
}
