package cisvc

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type VolumeMount struct {
	HostPath      string
	ContainerPath string
	Mode          string
}

type RunOptions struct {
	Image       string
	Script      string
	Environment []string
	Volumes     []VolumeMount
	LogFile     io.Writer
}

type ContainerRunner interface {
	Run(ctx context.Context, opts RunOptions) (int, string, error)
}

type DockerRunner struct{}

func (DockerRunner) Run(ctx context.Context, opts RunOptions) (int, string, error) {
	args := []string{"run", "--rm", "-w", "/workspace", "--entrypoint", "sh"}
	for _, env := range opts.Environment {
		args = append(args, "-e", env)
	}
	for _, volume := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s:%s", volume.HostPath, volume.ContainerPath, volume.Mode))
	}
	args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock:rw")
	args = append(args, opts.Image, "-x", "-c", opts.Script)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if opts.LogFile != nil {
		_, _ = opts.LogFile.Write(output)
	}
	if err == nil {
		return 0, string(output), nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), string(output), nil
	}
	return 1, string(output), fmt.Errorf("run container: %w", err)
}

func safeCommand(script string) string {
	return strings.ReplaceAll(script, "\r\n", "\n")
}
