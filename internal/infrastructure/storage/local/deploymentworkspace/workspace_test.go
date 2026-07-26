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

	if got := workspace.AppDir("demo"); got != filepath.Join("data", deploymentDataDir, "demo") {
		t.Fatalf("unexpected app dir: %q", got)
	}
	if got := workspace.ServiceDir("demo", "local", "default"); got != filepath.Join("data", deploymentDataDir, "demo", "local", "default") {
		t.Fatalf("unexpected service dir: %q", got)
	}
	if got := workspace.DeploymentLogPath("demo", "local", "default", "deploy-1"); got != filepath.Join("data", deploymentDataDir, "demo", "local", "default", "deployments", "deploy-1.log") {
		t.Fatalf("unexpected deployment log path: %q", got)
	}
}

func TestWorkspaceWritesInitScriptWithoutChangingContent(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "#!/bin/bash\r\nset -e\r\necho ok\r"

	if err := workspace.WriteConfig("demo", "local", "default", "init.sh", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.ServiceDir("demo", "local", "default"), "init.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("unexpected init script content: %q", string(written))
	}
}

func TestWorkspaceWritesNonInitConfigWithoutChangingContent(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "services:\r\n  app:\r\n    image: nginx\r\n"

	if err := workspace.WriteConfig("demo", "local", "default", "docker-compose.yml", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.ServiceDir("demo", "local", "default"), "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("unexpected compose content: %q", string(written))
	}
}

func TestWorkspaceRemovesApplicationDirectory(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	if err := workspace.WriteConfig("demo", "local", "default", "docker-compose.yml", "services: {}\n"); err != nil {
		t.Fatal(err)
	}

	if err := workspace.RemoveAppDir("demo"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(workspace.AppDir("demo")); !os.IsNotExist(err) {
		t.Fatalf("expected application directory removal, got %v", err)
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
	physicalServiceDir, err := workspace.PhysicalServiceDir(context.Background(), "demo", "local", "default")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Join(physicalRoot, deploymentDataDir, "demo", "local", "default"))
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
	if _, err := workspace.PhysicalServiceDir(context.Background(), "demo", "local", "default"); err != nil {
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
	if _, err := workspace.PhysicalServiceDir(context.Background(), "demo", "local", "default"); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical service dir error %v, got %v", wantErr, err)
	}
}
