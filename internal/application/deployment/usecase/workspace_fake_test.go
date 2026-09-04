package deploymentsvc

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"path/filepath"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type workspaceFake struct {
	dataRoot      string
	hasServiceDir bool
	queryOutput   string
	queryErr      error
	queryCalled   bool
	staged        []deploymentport.RemoteWorkspace
	removedLog    workspaceRemovedLog
	removeLogErr  error
}

type workspaceRemovedLog struct {
	serviceCode  string
	deploymentID string
}

func testWorkspace(dataRoot string) *workspaceFake {
	return &workspaceFake{dataRoot: dataRoot}
}

func (w *workspaceFake) ServiceDir(_ environmentport.SSHTarget, serviceCode string) (string, error) {
	return filepath.ToSlash(filepath.Join(w.dataRoot, "cd", serviceCode)), nil
}

func (w *workspaceFake) ServiceDirExists(context.Context, environmentport.SSHTarget, string) (bool, error) {
	return w.hasServiceDir, nil
}

func (w *workspaceFake) StageWorkspace(_ context.Context, _ environmentport.SSHTarget, workspace deploymentport.RemoteWorkspace) error {
	w.staged = append(w.staged, workspace)
	return nil
}

func (w *workspaceFake) Run(context.Context, environmentport.SSHTarget, string, io.Writer, string, ...string) error {
	return w.queryErr
}

func (w *workspaceFake) Query(context.Context, environmentport.SSHTarget, string, string, ...string) (string, error) {
	w.queryCalled = true
	return w.queryOutput, w.queryErr
}

func (w *workspaceFake) QueryAtEnvironmentRoot(context.Context, environmentport.SSHTarget, string, ...string) (string, error) {
	w.queryCalled = true
	return w.queryOutput, w.queryErr
}

func (w *workspaceFake) Writer(string, string) (io.WriteCloser, error) {
	return nopWriteCloser{Writer: &bytes.Buffer{}}, nil
}

func (w *workspaceFake) Read(string, string, int) ([]byte, int, error) {
	return nil, 0, nil
}

func (w *workspaceFake) Remove(serviceCode string, deploymentID string) error {
	w.removedLog = workspaceRemovedLog{serviceCode: serviceCode, deploymentID: deploymentID}
	return w.removeLogErr
}

func (w *workspaceFake) SetMissingDeploymentLog() {
	w.removeLogErr = fs.ErrNotExist
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

type staticTargetResolver struct {
	target environmentport.SSHTarget
	err    error
}

func (r staticTargetResolver) ResolveProjectTarget(context.Context, string) (environmentport.SSHTarget, error) {
	return r.target, r.err
}

func testSSHTarget(projectID string) environmentport.SSHTarget {
	revision := int64(1)
	status := "succeeded"
	return environmentport.SSHTarget{Environment: model.Environment{
		Id: "environment-1", ProjectId: projectID, State: model.EnvironmentStateActive,
		WorkspaceRoot: "/srv/orbit", TargetRevision: revision,
		SSHCredentialId: "credential-1", SSHCredentialRevision: revision,
		LastProbeRevision: &revision, LastProbeStatus: &status,
	}}
}

func (w *workspaceFake) SyncFiles(context.Context, environmentport.SSHTarget, string, []deploymentport.RemoteFile, string) error {
	return nil
}
