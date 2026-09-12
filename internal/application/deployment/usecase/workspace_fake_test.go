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
	staged        []deploymentport.Workspace
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

func (w *workspaceFake) ServiceDir(_ environmentport.Target, serviceCode string) (string, error) {
	return filepath.ToSlash(filepath.Join(w.dataRoot, "cd", serviceCode)), nil
}

func (w *workspaceFake) ServiceDirExists(context.Context, environmentport.Target, string) (bool, error) {
	return w.hasServiceDir, nil
}

func (w *workspaceFake) ComposeMountSourceDir(_ context.Context, target environmentport.Target, serviceCode string) (string, error) {
	return w.ServiceDir(target, serviceCode)
}

func (w *workspaceFake) StageWorkspace(_ context.Context, _ environmentport.Target, workspace deploymentport.Workspace) error {
	w.staged = append(w.staged, workspace)
	return nil
}

func (w *workspaceFake) Run(context.Context, environmentport.Target, string, io.Writer, string, ...string) error {
	return w.queryErr
}

func (w *workspaceFake) Query(context.Context, environmentport.Target, string, string, ...string) (string, error) {
	w.queryCalled = true
	return w.queryOutput, w.queryErr
}

func (w *workspaceFake) QueryAtEnvironmentRoot(context.Context, environmentport.Target, string, ...string) (string, error) {
	w.queryCalled = true
	return w.queryOutput, w.queryErr
}

func (w *workspaceFake) QueryAtEnvironmentRootInput(context.Context, environmentport.Target, []byte, string, ...string) (string, error) {
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
	target environmentport.Target
	err    error
}

func (r staticTargetResolver) ResolveProjectTarget(context.Context, string) (environmentport.Target, error) {
	return r.target, r.err
}

func testSSHTarget(projectID string) environmentport.Target {
	revision := int64(1)
	status := "succeeded"
	return environmentport.Target{Environment: model.Environment{
		Id: "environment-1", ProjectId: projectID, State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeSSH, TargetRevision: revision,
		WorkspaceRoot:     "/srv/orbit",
		LastProbeRevision: &revision, LastProbeStatus: &status,
		SSH: &model.EnvironmentSSHTarget{
			Platform: model.EnvironmentPlatformLinux, Host: "host.example.test", Port: 22, Username: "orbit",
			CredentialId: "credential-1", CredentialRevision: revision, HostKeyFingerprint: "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde=",
		},
	}}
}

func (w *workspaceFake) SyncFiles(context.Context, environmentport.Target, string, []deploymentport.WorkspaceFile, string) error {
	return nil
}
