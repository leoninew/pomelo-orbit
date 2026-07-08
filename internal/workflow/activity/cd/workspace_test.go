package cdsvc

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestWorkspaceUsesLogicalDataRootForApplicationFiles(t *testing.T) {
	workspace := newWorkspaceWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return t.TempDir(), nil
	})

	if got := workspace.AppDir("demo"); got != filepath.Join("data", "cd", "demo") {
		t.Fatalf("unexpected app dir: %q", got)
	}
	if got := workspace.DeploymentLogPath("demo", "deploy-1"); got != filepath.Join("data", "cd", "demo", "deployments", "deploy-1.log") {
		t.Fatalf("unexpected deployment log path: %q", got)
	}
}

func TestWorkspaceUsesPhysicalDataRootForComposePaths(t *testing.T) {
	physicalRoot := filepath.Join(t.TempDir(), "host-data")
	workspace := newWorkspaceWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		if logicalDataRoot != filepath.Clean("data") {
			t.Fatalf("unexpected logical data root: %q", logicalDataRoot)
		}
		return physicalRoot, nil
	})

	physicalDir, err := workspace.PhysicalDir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if physicalDir != filepath.ToSlash(physicalRoot) {
		t.Fatalf("unexpected physical dir: %q", physicalDir)
	}
	physicalAppDir, err := workspace.PhysicalAppDir(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Join(physicalRoot, "cd", "demo"))
	if physicalAppDir != want {
		t.Fatalf("unexpected physical app dir: %q", physicalAppDir)
	}
}

func TestWorkspaceCachesPhysicalDataRootResolver(t *testing.T) {
	calls := 0
	workspace := newWorkspaceWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		calls++
		return t.TempDir(), nil
	})

	if _, err := workspace.PhysicalDir(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.PhysicalAppDir(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", calls)
	}
}

func TestWorkspaceReturnsPhysicalDataRootError(t *testing.T) {
	wantErr := errors.New("missing host mount")
	workspace := newWorkspaceWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return "", wantErr
	})

	if _, err := workspace.PhysicalDir(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical dir error %v, got %v", wantErr, err)
	}
	if _, err := workspace.PhysicalAppDir(context.Background(), "demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical app dir error %v, got %v", wantErr, err)
	}
}
