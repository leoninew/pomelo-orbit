package deploymentworkspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceUsesLogicalDataRootForServiceFiles(t *testing.T) {
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return t.TempDir(), nil
	})

	if got := workspace.ServiceDir("sc"); got != filepath.Join("data", deploymentDataDir, "sc") {
		t.Fatalf("unexpected service dir: %q", got)
	}
	if got := workspace.DeploymentLogPath("sc", "deploy-1"); got != filepath.Join("data", deploymentDataDir, "sc", "deployments", "deploy-1.log") {
		t.Fatalf("unexpected deployment log path: %q", got)
	}
}

func TestWorkspaceWritesInitScriptWithoutChangingContent(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "#!/bin/bash\r\nset -e\r\necho ok\r"

	if err := workspace.WriteConfig("sc", "init.sh", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.ServiceDir("sc"), "init.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("unexpected init script content: %q", string(written))
	}
}

func TestWorkspaceReportsServiceDirectoryExistence(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })

	exists, err := workspace.ServiceDirExists("sc")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("service directory should not exist before deployment")
	}
	if err := os.MkdirAll(workspace.ServiceDir("sc"), 0o755); err != nil {
		t.Fatal(err)
	}
	exists, err = workspace.ServiceDirExists("sc")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("service directory should exist after creation")
	}
}

func TestWorkspaceWritesNonInitConfigWithoutChangingContent(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "services:\r\n  app:\r\n    image: nginx\r\n"

	if err := workspace.WriteConfig("sc", "docker-compose.yml", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.ServiceDir("sc"), "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("unexpected compose content: %q", string(written))
	}
}

func TestWorkspaceUsesPhysicalDataRootForComposePaths(t *testing.T) {
	physicalRoot := filepath.Join(t.TempDir(), "host-data")
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
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
	physicalServiceDir, err := workspace.PhysicalServiceDir(context.Background(), "sc")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Join(physicalRoot, deploymentDataDir, "sc"))
	if physicalServiceDir != want {
		t.Fatalf("unexpected physical service dir: %q", physicalServiceDir)
	}
}

func TestWorkspaceCachesPhysicalDataRootResolver(t *testing.T) {
	calls := 0
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		calls++
		return t.TempDir(), nil
	})

	if _, err := workspace.PhysicalDir(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.PhysicalServiceDir(context.Background(), "sc"); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", calls)
	}
}

func TestWorkspaceReturnsPhysicalDataRootError(t *testing.T) {
	wantErr := errors.New("missing host mount")
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return "", wantErr
	})

	if _, err := workspace.PhysicalDir(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical dir error %v, got %v", wantErr, err)
	}
	if _, err := workspace.PhysicalServiceDir(context.Background(), "sc"); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical service dir error %v, got %v", wantErr, err)
	}
}
