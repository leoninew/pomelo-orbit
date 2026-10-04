package sshrunner

import (
	"context"
	"errors"
	"fmt"
	"io"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/stream"
)

func (r *Runtime) StreamAtEnvironmentRoot(ctx context.Context, target environmentport.Target, output io.Writer, name string, args ...string) error {
	if !target.Environment.IsSSH() {
		return errors.New("SSH deployment runtime received a non-SSH environment")
	}
	directory := normalizeRemotePath(target.Environment.WorkspaceRoot)
	if directory == "" {
		return errors.New("environment workspace root is required")
	}
	command, _, err := remoteCommand(target.Environment.SSH.Platform, directory, name, args...)
	if err != nil {
		return err
	}
	client, cleanup, err := r.openSSH(ctx, target)
	if err != nil {
		return err
	}
	defer cleanup()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	var diagnostic stream.Diagnostic
	session.Stdout, session.Stderr = output, &diagnostic
	if err := session.Run(command); err != nil {
		return fmt.Errorf("follow SSH container logs: %w: %s", err, diagnostic.String())
	}
	return nil
}
