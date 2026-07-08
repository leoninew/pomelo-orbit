package cisvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIWorkspaceResolvesRelativeDataRootToAbsolutePhysicalMounts(t *testing.T) {
	workspace := newCIWorkspaceWithResolver("data", resolveAbsolutePhysicalDataRoot)
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

func TestCIWorkspaceKeepsAbsolutePhysicalDataRootSemantics(t *testing.T) {
	dataRoot := t.TempDir()
	workspace := newCIWorkspaceWithResolver(dataRoot, resolveAbsolutePhysicalDataRoot)
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

func TestCIWorkspaceUsesResolvedPhysicalDataRootForDockerMounts(t *testing.T) {
	logicalRoot := filepath.Join(string(filepath.Separator), "app", "data")
	physicalRoot := filepath.Join(t.TempDir(), "data")
	workspace := newCIWorkspaceWithResolver(logicalRoot, func(ctx context.Context, logicalDataRoot string) (string, error) {
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

func TestDockerRunArgsUseBindMountSyntax(t *testing.T) {
	hostPath := filepath.Join(t.TempDir(), "workspace")
	args, err := dockerRunArgs(RunOptions{
		Image:  "alpine",
		Script: "echo ok",
		Volumes: []VolumeMount{
			{HostPath: hostPath, ContainerPath: "/workspace", Mode: "rw"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, " -v ") {
		t.Fatalf("expected --mount syntax instead of -v: %s", joined)
	}
	want := "type=bind,source=" + hostPath + ",target=/workspace"
	if !containsArg(args, "--mount", want) {
		t.Fatalf("expected bind mount %q in args %+v", want, args)
	}
}

func TestDockerRunArgsRejectRelativeHostPath(t *testing.T) {
	_, err := dockerRunArgs(RunOptions{
		Image:  "alpine",
		Script: "echo ok",
		Volumes: []VolumeMount{
			{HostPath: filepath.Join("data", "ci", "repo", "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		},
	})
	if err == nil {
		t.Fatal("expected relative Docker host path to be rejected")
	}
}

func containsArg(args []string, flag string, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
