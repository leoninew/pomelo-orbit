package deploymentsvc

import (
	"context"
	"path/filepath"
)

type workspaceWrite struct {
	appCode     string
	envCode     string
	instanceKey string
	path        string
	content     string
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

func (w *workspaceFake) ServiceDir(appCode string, envCode string, instanceKey string) string {
	return filepath.Join(w.AppDir(appCode), envCode, instanceKey)
}

func (w *workspaceFake) DeploymentLogPath(appCode string, envCode string, instanceKey string, deploymentID string) string {
	return filepath.Join(w.ServiceDir(appCode, envCode, instanceKey), "deployments", deploymentID+".log")
}

func (w *workspaceFake) PhysicalDir(context.Context) (string, error) {
	if w.physicalErr != nil {
		return "", w.physicalErr
	}
	return filepath.ToSlash(w.physicalRoot), nil
}

func (w *workspaceFake) PhysicalServiceDir(ctx context.Context, appCode string, envCode string, instanceKey string) (string, error) {
	physicalRoot, err := w.PhysicalDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalRoot, "cd", appCode, envCode, instanceKey)), nil
}

func (w *workspaceFake) WriteConfig(appCode string, envCode string, instanceKey string, path string, content string) error {
	w.writes = append(w.writes, workspaceWrite{
		appCode:     appCode,
		envCode:     envCode,
		instanceKey: instanceKey,
		path:        path,
		content:     content,
	})
	return nil
}

func (w *workspaceFake) RemoveAppDir(appCode string) error {
	w.removedApps = append(w.removedApps, appCode)
	return nil
}

func (w *workspaceFake) Config(appCode string, envCode string, instanceKey string, path string) (string, bool) {
	for index := len(w.writes) - 1; index >= 0; index-- {
		write := w.writes[index]
		if write.appCode == appCode && write.envCode == envCode && write.instanceKey == instanceKey && write.path == path {
			return write.content, true
		}
	}
	return "", false
}
