package sshrunner

import (
	"context"
	"errors"
	"io"
	"os"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/pkg/sftp"
)

func (r *Runtime) workspaceFileClient(ctx context.Context, target environmentport.Target, value string) (*sftp.Client, string, func(), error) {
	client, cleanup, err := r.openSFTP(ctx, target)
	if err != nil {
		return nil, "", nil, err
	}
	resolver := newSFTPPathResolver(client, target.Environment.SSH.Platform)
	root, err := resolver.resolve(target.FileScope)
	if err == nil {
		value, err = resolver.scoped(root, value)
	}
	if err == nil && value == root {
		err = errors.New("file operation is outside the deployment workspace")
	}
	if err != nil {
		cleanup()
		return nil, "", nil, err
	}
	return client, value, cleanup, nil
}

func (r *Runtime) ReadFile(ctx context.Context, target environmentport.Target, value string) ([]byte, error) {
	client, value, cleanup, err := r.workspaceFileClient(ctx, target, value)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	file, err := client.Open(value)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(file)
}

func (r *Runtime) ListFiles(ctx context.Context, target environmentport.Target, value string) ([]string, error) {
	client, value, cleanup, err := r.workspaceFileClient(ctx, target, value)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	entries, err := client.ReadDir(value)
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
	client, value, cleanup, err := r.workspaceFileClient(ctx, target, value)
	if err != nil {
		return err
	}
	defer cleanup()
	err = client.Remove(value)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
