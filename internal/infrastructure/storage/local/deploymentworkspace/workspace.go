package deploymentworkspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type PhysicalWorkspaceResolver func(ctx context.Context, logicalWorkspaceRoot string) (string, error)

type Workspace struct {
	logicalWorkspaceRoot string
	resolver             PhysicalWorkspaceResolver

	physicalOnce          sync.Once
	physicalWorkspaceRoot string
	physicalErr           error
}

func NewWithResolver(workspaceRoot string, resolver PhysicalWorkspaceResolver) *Workspace {
	return &Workspace{logicalWorkspaceRoot: filepath.Clean(workspaceRoot), resolver: resolver}
}

func (w *Workspace) ServiceDir(serviceCode string) string {
	return filepath.Join(w.logicalWorkspaceRoot, serviceCode)
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

func (w *Workspace) RemoveDeploymentLog(serviceCode string, deploymentId string) error {
	return os.Remove(w.DeploymentLogPath(serviceCode, deploymentId))
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

func (w *Workspace) PhysicalWorkspaceRoot(ctx context.Context) (string, error) {
	w.physicalOnce.Do(func() {
		if w.resolver == nil {
			w.physicalWorkspaceRoot = w.logicalWorkspaceRoot
			return
		}
		w.physicalWorkspaceRoot, w.physicalErr = w.resolver(ctx, w.logicalWorkspaceRoot)
	})
	return w.physicalWorkspaceRoot, w.physicalErr
}

func (w *Workspace) PhysicalServiceDir(ctx context.Context, serviceCode string) (string, error) {
	physicalWorkspaceRoot, err := w.PhysicalWorkspaceRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalWorkspaceRoot, serviceCode)), nil
}

// ComposeMountSourceDir returns a host path only when Orbit runs in a container.
func (w *Workspace) ComposeMountSourceDir(ctx context.Context, serviceCode string) (string, error) {
	if w.resolver == nil {
		return "", nil
	}
	return w.PhysicalServiceDir(ctx, serviceCode)
}
