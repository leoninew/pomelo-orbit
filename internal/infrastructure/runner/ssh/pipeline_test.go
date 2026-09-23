package sshrunner

import (
	"context"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestWindowsPipelineWorkspacePathsAndSafety(t *testing.T) {
	workspace := &pipelineWorkspace{root: "D:/Orbit Workspace/pipeline"}
	mounts, err := workspace.DockerStageMounts(context.Background(), "repository", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(mounts) != 2 || mounts[0].HostPath != "D:/Orbit Workspace/pipeline/repository/workspace" ||
		mounts[1].HostPath != "D:/Orbit Workspace/pipeline/runs/run-1/artifacts" ||
		mounts[0].ContainerPath != "/workspace" || mounts[1].ContainerPath != "/artifacts" {
		t.Fatalf("remote mounts = %+v", mounts)
	}
	location, err := workspace.ArtifactLocation("run-1", "nested/report.json")
	if err != nil || location != "D:/Orbit Workspace/pipeline/runs/run-1/artifacts/nested/report.json" {
		t.Fatalf("remote artifact location = %q, err = %v", location, err)
	}
	if path := workspace.StageLogPath("run-1", "stage-1"); path != "D:/Orbit Workspace/pipeline/runs/run-1/stages/stage-1.log" {
		t.Fatalf("remote log path = %q", path)
	}
	logs := &pipelineLogs{workspace: workspace}
	if !logs.validLogPath(workspace.StageLogPath("run-1", "stage-1")) {
		t.Fatal("expected valid stage log")
	}
	for _, logPath := range []string{"D:/Orbit Workspace/pipeline/runs/../secret.log", "D:/Orbit Workspace/pipeline/runs/run-1/artifacts/stage-1.log"} {
		if logs.validLogPath(logPath) {
			t.Fatalf("accepted invalid stage log path %q", logPath)
		}
	}
	for _, artifact := range []string{"../secret", "nested/../../secret", "/absolute", "C:/other"} {
		if _, err := workspace.ArtifactLocation("run-1", artifact); err == nil {
			t.Fatalf("accepted invalid artifact %q", artifact)
		}
	}
	if _, err := workspace.DockerStageMounts(context.Background(), "../other-project", "run-1"); err == nil {
		t.Fatal("accepted repository traversal")
	}
}

func TestWindowsPipelineDockerArgsKeepScriptOutOfShellCommand(t *testing.T) {
	runner := &pipelineDockerRunner{root: "D:/Orbit Workspace/pipeline", target: environmentport.Target{Environment: model.Environment{SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformWindows}}}}
	options := pipelinerunport.RunOptions{
		ContainerName: "pomelo-orbit-stage-stage-1", Image: "alpine:latest", Script: "echo secret value",
		Volumes: []pipelinerunport.VolumeMount{
			{HostPath: "D:/Orbit Workspace/pipeline/repository/workspace", ContainerPath: "/workspace"},
			{HostPath: "D:/Orbit Workspace/pipeline/runs/run-1/artifacts", ContainerPath: "/artifacts"},
		},
		Environment: []string{"SECRET=redacted"},
	}
	args, err := runner.dockerArgs(options, options.ContainerName)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := remoteCommand("windows", runner.root, "docker", args...)
	if err != nil {
		t.Fatal(err)
	}
	script := decodePowerShellEncodedCommand(t, command)
	for _, expected := range []string{"docker.exe", "'-i'", "'-s'", "D:/Orbit Workspace/pipeline/repository/workspace", "/var/run/docker.sock"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("Windows command missing %q: %q", expected, script)
		}
	}
	if strings.Contains(script, options.Script) {
		t.Fatal("stage script must be passed on SSH stdin, not embedded in the PowerShell command")
	}
	options.Volumes[0].HostPath = "/local/control-plane/path"
	if _, err := runner.dockerArgs(options, options.ContainerName); err == nil {
		t.Fatal("accepted a control-plane Linux bind mount")
	}
}

func TestWindowsPipelineRejectsNonWindowsTargetBeforeConnection(t *testing.T) {
	runtime := NewPipelineRuntime()
	if _, _, _, err := runtime.RuntimeForTarget(context.Background(), environmentport.Target{}); err == nil {
		t.Fatal("non-Windows or missing credential target must be rejected")
	}
}
