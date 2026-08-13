package pipelinerunner

import (
	"path/filepath"
	"strings"
	"testing"

	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
)

func TestRunArgsUseBindMountSyntax(t *testing.T) {
	hostPath := filepath.Join(t.TempDir(), "workspace")
	args, err := runArgs(pipelinerunport.RunOptions{
		ContainerName: "pipeline-stage-1",
		Image:         "alpine",
		Script:        "echo ok",
		Volumes: []pipelinerunport.VolumeMount{
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
	if !containsArg(args, "--name", "pipeline-stage-1") {
		t.Fatalf("expected container name in args %+v", args)
	}
}

func TestRunArgsRejectRelativeHostPath(t *testing.T) {
	_, err := runArgs(pipelinerunport.RunOptions{
		Image:  "alpine",
		Script: "echo ok",
		Volumes: []pipelinerunport.VolumeMount{
			{HostPath: filepath.Join("data", "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		},
	})
	if err == nil {
		t.Fatal("expected relative Docker host path to be rejected")
	}
}

func TestCommandArgsUseStageWorkspaceWithoutContainerName(t *testing.T) {
	hostPath := filepath.Join(t.TempDir(), "workspace")
	args, err := commandArgs(pipelinerunport.RunOptions{
		ContainerName: "pipeline-stage-1",
		Image:         "alpine/git",
		Volumes: []pipelinerunport.VolumeMount{
			{HostPath: hostPath, ContainerPath: "/workspace", Mode: "rw"},
		},
	}, "git rev-parse HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if containsArg(args, "--name", "pipeline-stage-1") {
		t.Fatalf("command collector must not reuse the stage container name: %+v", args)
	}
	if !containsArg(args, "--entrypoint", "sh") {
		t.Fatalf("expected shell entrypoint in command collector args: %+v", args)
	}
	if args[len(args)-2] != "-c" || args[len(args)-1] != "git rev-parse HEAD" {
		t.Fatalf("unexpected command collector args: %+v", args)
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
