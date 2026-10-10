package localrunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestCanceledWorkspaceDoesNotPublishWhileFileLockIsHeld(t *testing.T) {
	runtime := NewRuntime(nil)
	root := t.TempDir()
	directory := filepath.Join(root, "service")
	unlock, err := runtime.lock(context.Background(), directory)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- runtime.StageWorkspace(ctx, localTarget(root), deploymentport.Workspace{
			Location: deploymentport.ServiceLocation{Code: "api", Directory: directory}, Compose: "services: {}\n",
		})
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled workspace returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("workspace waited for previous file publisher")
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled workspace was published: %v", err)
	}
}

func TestRuntimeStagesLocalWorkspaceAndResolvesDockerPath(t *testing.T) {
	root := t.TempDir()
	runtime := NewRuntime(func(_ context.Context, source string) (string, error) {
		return "/daemon/" + filepath.Base(source), nil
	})
	target := localTarget(root)
	serviceDir, err := runtime.ServiceDir(target, deploymentport.ServiceLocation{Code: "api-default", Directory: filepath.Join(root, "deployment", "api-default")})
	if err != nil {
		t.Fatal(err)
	}
	mountSource, err := runtime.ComposeMountSourceDir(context.Background(), target, deploymentport.ServiceLocation{Code: "api-default", Directory: filepath.Join(root, "deployment", "api-default")})
	if err != nil {
		t.Fatal(err)
	}
	if mountSource != "/daemon/api-default" {
		t.Fatalf("Docker mount source = %q", mountSource)
	}
	configPath := filepath.Join(serviceDir, "config", "app.env")
	if err := runtime.StageWorkspace(context.Background(), target, deploymentport.Workspace{
		Location:    deploymentport.ServiceLocation{Code: "api-default", Directory: serviceDir},
		Directories: []string{filepath.Join(serviceDir, "data")},
		Files: []deploymentport.WorkspaceFile{{
			Path: configPath, Content: []byte("APP_MODE=local\n"), Mode: 0o640,
		}},
		Compose: "services: {}\n",
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "APP_MODE=local\n" {
		t.Fatalf("staged file = %q", content)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "docker-compose.yml")); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeExpandsHomeWorkspaceRootAtUseTime(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		t.Fatalf("user home directory is required: %v", err)
	}
	runtime := NewRuntime(nil)
	serviceDir, err := runtime.ServiceDir(localTarget("~/.pomelo-orbit"), deploymentport.ServiceLocation{Code: "api-default", Directory: "~/.pomelo-orbit/deployment/api-default"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".pomelo-orbit", "deployment", "api-default")
	if serviceDir != want {
		t.Fatalf("service dir = %q, want %q", serviceDir, want)
	}
}

func TestRuntimeSyncFilesReplacesSnapshotAndRemovesStagedFiles(t *testing.T) {
	root := t.TempDir()
	runtime := NewRuntime(nil)
	target := localTarget(root)
	directory := filepath.Join(root, "deployment", "traefik-default", "gateway", "dynamic")
	filePath := filepath.Join(directory, "routes.yaml")
	for _, content := range []string{"http:\n  routers: {}\n", "http:\n  routers:\n    api: {}\n"} {
		if err := runtime.SyncFiles(context.Background(), target, directory, []deploymentport.WorkspaceFile{{
			Path: filePath, Content: []byte(content), Mode: 0o600,
		}}, ""); err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filePath)
		if err != nil || string(actual) != content {
			t.Fatalf("snapshot = %q, error = %v", actual, err)
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "routes.yaml" {
		t.Fatalf("snapshot directory = %+v, error = %v", entries, err)
	}
}

func TestRuntimeRejectsNonLocalTargetsAndMissingResolver(t *testing.T) {
	sshTarget := environmentport.Target{Environment: model.Environment{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH:        &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux},
	}}
	runtime := NewRuntime(nil)
	if _, err := runtime.ServiceDir(sshTarget, deploymentport.ServiceLocation{Code: "api-default", Directory: "/custom/api-default"}); err == nil {
		t.Fatal("expected non-local target to be rejected")
	}
	if _, err := runtime.ComposeMountSourceDir(context.Background(), localTarget(t.TempDir()), deploymentport.ServiceLocation{Code: "api-default", Directory: "/custom/api-default"}); err == nil {
		t.Fatal("expected missing Docker daemon path resolver to be rejected")
	}
}

func TestRuntimeUpdatesExistingCompose(t *testing.T) {
	ctx := context.Background()
	runtime := NewRuntime(nil)
	target := localTarget(t.TempDir())
	directory := filepath.Join(t.TempDir(), "custom-service")
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	compose := filepath.Join(directory, "docker-compose.yml")
	if err := os.WriteFile(compose, []byte("services:\n  api:\n    image: api:old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	workspace := deploymentport.Workspace{Location: deploymentport.ServiceLocation{Code: "api", Directory: directory}, Compose: "services: {}\n"}
	if err := runtime.StageWorkspace(ctx, target, workspace); err != nil {
		t.Fatal(err)
	}
	target.FileScope = directory
	if body, err := runtime.ReadFile(ctx, target, filepath.Join(directory, "docker-compose.yml")); err != nil || string(body) != workspace.Compose {
		t.Fatalf("custom workspace content=%q err=%v", body, err)
	}
}

func TestCustomWorkspaceRejectsSymlinkEscapes(t *testing.T) {
	runtime := NewRuntime(nil)
	directory, outside := t.TempDir(), t.TempDir()
	linked := filepath.Join(directory, "linked")
	if err := os.Symlink(outside, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	target := localTarget(directory)
	target.FileScope = directory
	if _, err := runtime.ReadFile(context.Background(), target, filepath.Join(linked, "config")); err == nil {
		t.Fatal("read escaped the workspace")
	}
	if err := runtime.SyncFiles(context.Background(), target, linked, nil, ""); err == nil {
		t.Fatal("sync escaped the workspace")
	}
	composeOutside := filepath.Join(outside, "compose.yml")
	if err := os.WriteFile(composeOutside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(composeOutside, filepath.Join(directory, "docker-compose.yml")); err != nil {
		t.Fatal(err)
	}
	if err := runtime.StageWorkspace(context.Background(), target, deploymentport.Workspace{
		Location: deploymentport.ServiceLocation{Code: "api", Directory: directory}, Compose: "replacement",
	}); err == nil {
		t.Fatal("Compose escaped the workspace")
	}
	if body, err := os.ReadFile(composeOutside); err != nil || string(body) != "original" {
		t.Fatalf("outside Compose=%q err=%v", body, err)
	}
}

func localTarget(root string) environmentport.Target {
	return environmentport.Target{FileScope: filepath.Join(root, "deployment", "traefik-default"), Environment: model.Environment{TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: root}}
}
