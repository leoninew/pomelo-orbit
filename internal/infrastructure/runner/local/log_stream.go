package localrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/stream"
)

func (r *Runtime) Stream(ctx context.Context, target environmentport.Target, serviceCode string, output io.Writer, name string, args ...string) error {
	directory, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(directory, "docker-compose.yml")); os.IsNotExist(err) {
		return deploymentport.ErrLogNotReady
	} else if err != nil {
		return err
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
