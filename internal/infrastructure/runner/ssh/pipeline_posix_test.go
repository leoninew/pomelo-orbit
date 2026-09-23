package sshrunner

import (
	"context"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestLinuxPipelineWorkspacePathsAndSafety(t *testing.T) {
	workspace := &pipelineWorkspace{root: "/srv/orbit workspace/pipeline"}
	mounts, err := workspace.DockerStageMounts(context.Background(), "repository", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 2 || mounts[0].HostPath != "/srv/orbit workspace/pipeline/repository/workspace" ||
		mounts[1].HostPath != "/srv/orbit workspace/pipeline/runs/run-1/artifacts" {
		t.Fatalf("Linux remote mounts = %+v", mounts)
	}
	location, err := workspace.ArtifactLocation("run-1", "nested/report.json")
	if err != nil || location != "/srv/orbit workspace/pipeline/runs/run-1/artifacts/nested/report.json" {
		t.Fatalf("Linux artifact location = %q, err = %v", location, err)
	}
	logs := &pipelineLogs{workspace: workspace}
	if !logs.validLogPath(workspace.StageLogPath("run-1", "stage-1")) {
		t.Fatal("expected valid Linux stage log")
	}
	for _, logPath := range []string{
		"/srv/orbit workspace/pipeline/runs/run-1/../secret.log",
		"/srv/orbit workspace/pipeline/runs/run-1/artifacts/stage-1.log",
	} {
		if logs.validLogPath(logPath) {
			t.Fatalf("accepted invalid Linux log path %q", logPath)
		}
	}
	for _, artifact := range []string{"../secret", "nested/../../secret", "/absolute", "C:/other"} {
		if _, err := workspace.ArtifactLocation("run-1", artifact); err == nil {
			t.Fatalf("accepted invalid Linux artifact path %q", artifact)
		}
	}
	if _, err := workspace.DockerStageMounts(context.Background(), "../other-project", "run-1"); err == nil {
		t.Fatal("accepted repository traversal")
	}
}

func TestLinuxPipelineDockerArgsUsePosixPathsAndSSHStdin(t *testing.T) {
	runner := &pipelineDockerRunner{root: "/srv/orbit workspace/pipeline", target: environmentport.Target{Environment: model.Environment{
		SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux},
	}}}
	options := pipelinerunport.RunOptions{
		ContainerName: "pomelo-orbit-stage-stage-1", Image: "docker:29.4", Script: "echo secret value",
		Volumes: []pipelinerunport.VolumeMount{
			{HostPath: "/srv/orbit workspace/pipeline/repository/workspace", ContainerPath: "/workspace"},
			{HostPath: "/srv/orbit workspace/pipeline/runs/run-1/artifacts", ContainerPath: "/artifacts"},
		},
		Environment: []string{"SECRET=redacted"},
	}
	args, err := runner.dockerArgs(options, options.ContainerName)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := remoteCommand(model.EnvironmentPlatformLinux, runner.root, "docker", args...)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"sh -lc", "'docker'", "'/srv/orbit workspace/pipeline'", "/srv/orbit workspace/pipeline/repository/workspace", "/var/run/docker.sock"} {
		if !strings.Contains(command, expected) {
			t.Fatalf("Linux command missing %q: %s", expected, command)
		}
	}
	if strings.Contains(command, options.Script) {
		t.Fatal("stage script must be passed on SSH stdin, not embedded in the Linux shell command")
	}
	options.Volumes[0].HostPath = "D:/control-plane/path"
	if _, err := runner.dockerArgs(options, options.ContainerName); err == nil {
		t.Fatal("accepted a Windows bind mount on Linux")
	}
}
