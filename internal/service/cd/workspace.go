package cdsvc

import (
	"context"
	"path/filepath"
	"sync"

	"gitee.com/leoninew/pomelo-orbit/internal/runtimepath"
)

type cdPhysicalDataRootResolver func(ctx context.Context, logicalDataRoot string) (string, error)

type Workspace struct {
	logicalDataRoot string
	resolver        cdPhysicalDataRootResolver

	physicalOnce     sync.Once
	physicalDataRoot string
	physicalErr      error
}

func NewWorkspace(dataRoot string) *Workspace {
	return newWorkspaceWithResolver(dataRoot, runtimepath.ResolvePhysicalDataRoot)
}

func newWorkspaceWithResolver(dataRoot string, resolver cdPhysicalDataRootResolver) *Workspace {
	return &Workspace{logicalDataRoot: filepath.Clean(dataRoot), resolver: resolver}
}

func (w *Workspace) AppDir(appCode string) string {
	return filepath.Join(w.logicalDataRoot, "cd", appCode)
}

func (w *Workspace) DeploymentLogPath(appCode string, deploymentId string) string {
	return filepath.Join(w.AppDir(appCode), "deployments", deploymentId+".log")
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

func (w *Workspace) PhysicalAppDir(ctx context.Context, appCode string) (string, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalDataRoot, "cd", appCode)), nil
}
