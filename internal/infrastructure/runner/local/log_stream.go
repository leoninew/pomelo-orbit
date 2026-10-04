package localrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/stream"
)

func (r *Runtime) StreamAtEnvironmentRoot(ctx context.Context, target environmentport.Target, output io.Writer, name string, args ...string) error {
	directory, err := r.localWorkspaceRoot(target)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return fmt.Errorf("create local environment workspace: %w", err)
	}
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = directory
	command.Stdout = output
	var diagnostic stream.Diagnostic
	command.Stderr = &diagnostic
	if err := command.Run(); err != nil {
		return fmt.Errorf("follow container logs: %w: %s", err, diagnostic.String())
	}
	return nil
}
