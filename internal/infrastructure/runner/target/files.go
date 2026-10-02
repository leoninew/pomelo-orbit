package targetrunner

import (
	"context"
	"errors"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

func (r Runtime) fileRuntime(target environmentport.Target) (deploymentport.WorkspaceFiles, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return nil, err
	}
	files, ok := runtime.(deploymentport.WorkspaceFiles)
	if !ok {
		return nil, errors.New("target workspace file operations are not configured")
	}
	return files, nil
}

func (r Runtime) ReadFile(ctx context.Context, target environmentport.Target, value string) ([]byte, error) {
	files, err := r.fileRuntime(target)
	if err != nil {
		return nil, err
	}
	return files.ReadFile(ctx, target, value)
}

func (r Runtime) ListFiles(ctx context.Context, target environmentport.Target, value string) ([]string, error) {
	files, err := r.fileRuntime(target)
	if err != nil {
		return nil, err
	}
	return files.ListFiles(ctx, target, value)
}

func (r Runtime) RemoveFile(ctx context.Context, target environmentport.Target, value string) error {
	files, err := r.fileRuntime(target)
	if err != nil {
		return err
	}
	return files.RemoveFile(ctx, target, value)
}
