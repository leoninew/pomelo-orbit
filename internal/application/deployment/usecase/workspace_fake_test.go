package deploymentsvc

import (
	"context"
	"io/fs"
	"path/filepath"
)

type workspaceWrite struct {
	serviceCode string
	path        string
	content     string
}

type workspaceFake struct {
	dataRoot      string
	physicalRoot  string
	physicalErr   error
	hasServiceDir bool
	writes        []workspaceWrite
	removedLog    workspaceRemovedLog
	removeLogErr  error
}

type workspaceRemovedLog struct {
	serviceCode  string
	deploymentID string
}

func testWorkspace(dataRoot string) *workspaceFake {
	return testWorkspaceWithPhysicalRoot(dataRoot, dataRoot, nil)
}

func testWorkspaceWithPhysicalRoot(dataRoot string, physicalRoot string, physicalErr error) *workspaceFake {
	return &workspaceFake{dataRoot: dataRoot, physicalRoot: physicalRoot, physicalErr: physicalErr}
}

func (w *workspaceFake) ServiceDir(serviceCode string) string {
	return filepath.Join(w.dataRoot, "cd", serviceCode)
}

func (w *workspaceFake) ServiceDirExists(string) (bool, error) {
	return w.hasServiceDir, nil
}

func (w *workspaceFake) DeploymentLogPath(serviceCode string, deploymentId string) string {
	return filepath.Join(w.ServiceDir(serviceCode), "deployments", deploymentId+".log")
}

func (w *workspaceFake) RemoveDeploymentLog(serviceCode string, deploymentID string) error {
	w.removedLog = workspaceRemovedLog{serviceCode: serviceCode, deploymentID: deploymentID}
	return w.removeLogErr
}

func (w *workspaceFake) PhysicalWorkspaceRoot(context.Context) (string, error) {
	if w.physicalErr != nil {
		return "", w.physicalErr
	}
	return filepath.ToSlash(w.physicalRoot), nil
}

func (w *workspaceFake) ComposeMountSourceDir(ctx context.Context, serviceCode string) (string, error) {
	physicalRoot, err := w.PhysicalWorkspaceRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalRoot, serviceCode)), nil
}

func (w *workspaceFake) WriteConfig(serviceCode string, path string, content string) error {
	w.writes = append(w.writes, workspaceWrite{
		serviceCode: serviceCode,
		path:        path,
		content:     content,
	})
	return nil
}

func (w *workspaceFake) SetMissingDeploymentLog() {
	w.removeLogErr = fs.ErrNotExist
}

func (w *workspaceFake) Config(serviceCode string, path string) (string, bool) {
	for index := len(w.writes) - 1; index >= 0; index-- {
		write := w.writes[index]
		if write.serviceCode == serviceCode && write.path == path {
			return write.content, true
		}
	}
	return "", false
}
