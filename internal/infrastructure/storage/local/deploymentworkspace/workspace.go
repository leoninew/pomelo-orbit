package deploymentworkspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const deploymentDataDir = "deployment"

type PhysicalDataRootResolver func(ctx context.Context, logicalDataRoot string) (string, error)

type Workspace struct {
	logicalDataRoot string
	resolver        PhysicalDataRootResolver

	physicalOnce     sync.Once
	physicalDataRoot string
	physicalErr      error
}

func NewWithResolver(dataRoot string, resolver PhysicalDataRootResolver) *Workspace {
	return &Workspace{logicalDataRoot: dataRoot, resolver: resolver}
}

func (w *Workspace) ServiceDir(serviceCode string) string {
	return filepath.Join(w.logicalDataRoot, deploymentDataDir, serviceCode)
}

func (w *Workspace) ServiceDirExists(serviceCode string) (bool, error) {
	info, err := os.Stat(w.ServiceDir(serviceCode))
	if err == nil {
		return info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (w *Workspace) DeploymentLogPath(serviceCode string, deploymentId string) string {
	return filepath.Join(w.ServiceDir(serviceCode), "deployments", deploymentId+".log")
}

func (w *Workspace) WriteConfig(serviceCode string, path string, content string) error {
	path = filepath.Join(w.ServiceDir(serviceCode), path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		return os.Chmod(path, 0o755)
	}
	return nil
}

func (w *Workspace) PhysicalDataRoot(ctx context.Context) (string, error) {
	w.physicalOnce.Do(func() {
		w.physicalDataRoot, w.physicalErr = w.resolver(ctx, w.logicalDataRoot)
	})
	return w.physicalDataRoot, w.physicalErr
}

func (w *Workspace) PhysicalDir(ctx context.Context) (string, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(physicalDataRoot), nil
}

func (w *Workspace) PhysicalServiceDir(ctx context.Context, serviceCode string) (string, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalDataRoot, deploymentDataDir, serviceCode)), nil
}
