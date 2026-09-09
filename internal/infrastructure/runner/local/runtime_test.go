package localrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRuntimeStagesLocalWorkspaceAndResolvesDockerPath(t *testing.T) {
	root := t.TempDir()
	runtime := NewRuntime(root, func(_ context.Context, source string) (string, error) {
		return "/daemon/" + filepath.Base(source), nil
	})
	target := localTarget()
	serviceDir, err := runtime.ServiceDir(target, "api-default")
	if err != nil {
		t.Fatal(err)
	}
	mountSource, err := runtime.ComposeMountSourceDir(context.Background(), target, "api-default")
	if err != nil {
		t.Fatal(err)
	}
	if mountSource != "/daemon/api-default" {
		t.Fatalf("Docker mount source = %q", mountSource)
	}
	configPath := filepath.Join(serviceDir, "config", "app.env")
	if err := runtime.StageWorkspace(context.Background(), target, deploymentport.Workspace{
		ServiceCode: "api-default",
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

func TestRuntimeRejectsNonLocalTargetsAndMissingResolver(t *testing.T) {
	sshTarget := environmentport.Target{Environment: model.Environment{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH:        &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux},
	}}
	runtime := NewRuntime(t.TempDir(), nil)
	if _, err := runtime.ServiceDir(sshTarget, "api-default"); err == nil {
		t.Fatal("expected non-local target to be rejected")
	}
	if _, err := runtime.ComposeMountSourceDir(context.Background(), localTarget(), "api-default"); err == nil {
		t.Fatal("expected missing Docker daemon path resolver to be rejected")
	}
}

func localTarget() environmentport.Target {
	return environmentport.Target{Environment: model.Environment{TargetType: model.EnvironmentTargetTypeLocal}}
}
