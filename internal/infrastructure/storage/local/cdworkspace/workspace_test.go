package cdworkspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceUsesLogicalDataRootForApplicationFiles(t *testing.T) {
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return t.TempDir(), nil
	})

	if got := workspace.AppDir("demo"); got != filepath.Join("data", "cd", "demo") {
		t.Fatalf("unexpected app dir: %q", got)
	}
	if got := workspace.DeploymentLogPath("demo", "deploy-1"); got != filepath.Join("data", "cd", "demo", "deployments", "deploy-1.log") {
		t.Fatalf("unexpected deployment log path: %q", got)
	}
}

func TestWorkspaceWritesConfigAndNormalizesInitScriptLineEndings(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "#!/bin/bash\r\nset -e\r\necho ok\r"

	if err := workspace.WriteConfig("demo", "init.sh", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.AppDir("demo"), "init.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), "\r") || string(written) != "#!/bin/bash\nset -e\necho ok\n" {
		t.Fatalf("unexpected init script content: %q", string(written))
	}
}

func TestWorkspaceWritesNonInitConfigWithoutChangingContent(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	content := "services:\r\n  app:\r\n    image: nginx\r\n"

	if err := workspace.WriteConfig("demo", "docker-compose.yml", content); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(workspace.AppDir("demo"), "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("unexpected compose content: %q", string(written))
	}
}

func TestWorkspaceRemovesApplicationDirectory(t *testing.T) {
	workspace := NewWithResolver(t.TempDir(), func(context.Context, string) (string, error) { return "", nil })
	if err := workspace.WriteConfig("demo", "docker-compose.yml", "services: {}\n"); err != nil {
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
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
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
	workspace := NewWithResolver("data", func(ctx context.Context, logicalDataRoot string) (string, error) {
		return "", wantErr
	})

	if _, err := workspace.PhysicalDir(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical dir error %v, got %v", wantErr, err)
	}
	if _, err := workspace.PhysicalAppDir(context.Background(), "demo"); !errors.Is(err, wantErr) {
		t.Fatalf("expected physical app dir error %v, got %v", wantErr, err)
	}
}
