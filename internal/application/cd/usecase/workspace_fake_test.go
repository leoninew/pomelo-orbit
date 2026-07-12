package cdsvc

import (
	"context"
	"path/filepath"
)

type workspaceWrite struct {
	appCode string
	path    string
	content string
}

type workspaceFake struct {
	dataRoot     string
	physicalRoot string
	physicalErr  error
	writes       []workspaceWrite
	removedApps  []string
}

func testWorkspace(dataRoot string) *workspaceFake {
	return testWorkspaceWithPhysicalRoot(dataRoot, dataRoot, nil)
}

func testWorkspaceWithPhysicalRoot(dataRoot string, physicalRoot string, physicalErr error) *workspaceFake {
	return &workspaceFake{dataRoot: dataRoot, physicalRoot: physicalRoot, physicalErr: physicalErr}
}

func (w *workspaceFake) AppDir(appCode string) string {
	return filepath.Join(w.dataRoot, "cd", appCode)
}

func (w *workspaceFake) DeploymentLogPath(appCode string, deploymentID string) string {
	return filepath.Join(w.AppDir(appCode), "deployments", deploymentID+".log")
}

func (w *workspaceFake) PhysicalDir(context.Context) (string, error) {
	if w.physicalErr != nil {
		return "", w.physicalErr
	}
	return filepath.ToSlash(w.physicalRoot), nil
}

func (w *workspaceFake) PhysicalAppDir(ctx context.Context, appCode string) (string, error) {
	physicalRoot, err := w.PhysicalDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalRoot, "cd", appCode)), nil
}

func (w *workspaceFake) WriteConfig(appCode string, path string, content string) error {
	w.writes = append(w.writes, workspaceWrite{appCode: appCode, path: path, content: content})
	return nil
}

func (w *workspaceFake) RemoveAppDir(appCode string) error {
	w.removedApps = append(w.removedApps, appCode)
	return nil
}

func (w *workspaceFake) Config(appCode string, path string) (string, bool) {
	for index := len(w.writes) - 1; index >= 0; index-- {
		write := w.writes[index]
		if write.appCode == appCode && write.path == path {
			return write.content, true
		}
	}
	return "", false
}

var _ interface {
	AppDir(string) string
	DeploymentLogPath(string, string) string
	PhysicalDir(context.Context) (string, error)
	PhysicalAppDir(context.Context, string) (string, error)
	WriteConfig(string, string, string) error
	RemoveAppDir(string) error
} = (*workspaceFake)(nil)
