package sshrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/stream"
	"github.com/pkg/sftp"
)

func (r *Runtime) Stream(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation, output io.Writer, name string, args ...string) error {
	directory, err := r.ServiceDir(target, location)
	if err != nil {
		return err
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
	files, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	configPath, resolveErr := newSFTPPathResolver(files, target.Environment.SSH.Platform).resolve(path.Join(normalizeRemotePath(directory), "docker-compose.yml"))
	if resolveErr != nil {
		_ = files.Close()
		return resolveErr
	}
	_, statErr := files.Stat(configPath)
	_ = files.Close()
	if os.IsNotExist(statErr) {
		return deploymentport.ErrLogNotReady
	}
	if statErr != nil {
		return statErr
	}
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
