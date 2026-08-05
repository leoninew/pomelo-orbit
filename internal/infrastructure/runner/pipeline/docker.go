package pipelinerunner

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
)

type DockerRunner struct{}

func (DockerRunner) Run(ctx context.Context, opts pipelinerunport.RunOptions) (int, string, error) {
	args, err := runArgs(opts)
	if err != nil {
		return 1, "", err
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = opts.LogWriter
	cmd.Stderr = opts.LogWriter
	err = cmd.Run()
	if err == nil {
		return 0, "", nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		if cleanupErr := removeContainer(opts.ContainerName); cleanupErr != nil {
			return 1, "", fmt.Errorf("%w; remove timed-out container: %v", ctxErr, cleanupErr)
		}
		return 1, "", ctxErr
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), "", nil
	}
	return 1, "", fmt.Errorf("run container: %w", err)
}

func runArgs(opts pipelinerunport.RunOptions) ([]string, error) {
	args := []string{"run", "--rm", "-w", "/workspace", "--entrypoint", "sh"}
	if opts.ContainerName != "" {
		args = append(args, "--name", opts.ContainerName)
	}
	for _, env := range opts.Environment {
		args = append(args, "-e", env)
	}
	for _, volume := range opts.Volumes {
		mount, err := bindMountArg(volume)
		if err != nil {
			return nil, err
		}
		args = append(args, "--mount", mount)
	}
	args = append(args, "--mount", "type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock")
	args = append(args, opts.Image, "-x", "-c", opts.Script)
	return args, nil
}

func removeContainer(containerName string) error {
	if containerName == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", "rm", "--force", containerName).CombinedOutput()
	if err != nil && !strings.Contains(string(output), "No such container") {
		return fmt.Errorf("docker rm --force %s: %w", containerName, err)
	}
	return nil
}

func bindMountArg(volume pipelinerunport.VolumeMount) (string, error) {
	hostPath := strings.TrimSpace(volume.HostPath)
	containerPath := strings.TrimSpace(volume.ContainerPath)
	mode := strings.TrimSpace(volume.Mode)
	if hostPath == "" || containerPath == "" {
		return "", fmt.Errorf("docker bind mount requires host and container paths")
	}
	if !filepath.IsAbs(hostPath) {
		return "", fmt.Errorf("docker bind mount host path must be absolute: %s", hostPath)
	}
	mount := fmt.Sprintf("type=bind,source=%s,target=%s", hostPath, containerPath)
	if mode == "ro" {
		mount += ",readonly"
	}
	return mount, nil
}
