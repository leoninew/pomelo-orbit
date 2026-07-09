package runner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}
