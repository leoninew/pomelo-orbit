package cdsvc

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error
}

type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	cmdLine := append([]string{name}, args...)
	_, _ = fmt.Fprintf(log, "$ %s\n", strings.Join(cmdLine, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if log != nil {
		_, _ = log.Write(output)
	}
	if err != nil {
		return fmt.Errorf("run %s: %w\n%s", name, err, string(output))
	}
	return nil
}
