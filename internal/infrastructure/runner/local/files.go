package localrunner

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

func (r *Runtime) workspaceFilePath(ctx context.Context, target environmentport.Target, value string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root, err := r.localWorkspaceRoot(target)
	if err != nil {
		return "", err
	}
	value = filepath.Clean(value)
	if value == root || !pathWithin(root, value) {
		return "", errors.New("file operation is outside the deployment workspace")
	}
	return value, nil
}

func (r *Runtime) ReadFile(ctx context.Context, target environmentport.Target, value string) ([]byte, error) {
	value, err := r.workspaceFilePath(ctx, target, value)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(value)
}

func (r *Runtime) ListFiles(ctx context.Context, target environmentport.Target, value string) ([]string, error) {
	value, err := r.workspaceFilePath(ctx, target, value)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(value)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names, nil
}

func (r *Runtime) RemoveFile(ctx context.Context, target environmentport.Target, value string) error {
	value, err := r.workspaceFilePath(ctx, target, value)
	if err != nil {
		return err
	}
	err = os.Remove(value)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
