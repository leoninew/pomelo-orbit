package targetrunner

import (
	"context"
	"errors"
	"io"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// Runtime dispatches exclusively by the persisted Environment target type.
// It never infers local execution from an SSH hostname or incomplete fields.
type Runtime struct {
	local deploymentport.Runtime
	ssh   deploymentport.Runtime
}

func New(local deploymentport.Runtime, ssh deploymentport.Runtime) Runtime {
	return Runtime{local: local, ssh: ssh}
}

func (r Runtime) OpenSession(ctx context.Context, target environmentport.Target) (context.Context, func(), error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return nil, nil, err
	}
	sessions, ok := runtime.(deploymentport.RuntimeSessions)
	if !ok {
		return nil, nil, errors.New("target runtime sessions are not configured")
	}
	return sessions.OpenSession(ctx, target)
}

func (r Runtime) ServiceDir(target environmentport.Target, location deploymentport.ServiceLocation) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.ServiceDir(target, location)
}

func (r Runtime) ServiceDirExists(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation) (bool, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return false, err
	}
	return runtime.ServiceDirExists(ctx, target, location)
}

func (r Runtime) ComposeMountSourceDir(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.ComposeMountSourceDir(ctx, target, location)
}

func (r Runtime) StageWorkspace(ctx context.Context, target environmentport.Target, workspace deploymentport.Workspace) error {
	runtime, err := r.forTarget(target)
	if err != nil {
		return err
	}
	return runtime.StageWorkspace(ctx, target, workspace)
}

func (r Runtime) Run(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation, log io.Writer, name string, args ...string) error {
	runtime, err := r.forTarget(target)
	if err != nil {
		return err
	}
	return runtime.Run(ctx, target, location, log, name, args...)
}

func (r Runtime) StreamAtEnvironmentRoot(ctx context.Context, target environmentport.Target, output io.Writer, name string, args ...string) error {
	runtime, err := r.forTarget(target)
	if err != nil {
		return err
	}
	return runtime.StreamAtEnvironmentRoot(ctx, target, output, name, args...)
}

func (r Runtime) Query(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation, name string, args ...string) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.Query(ctx, target, location, name, args...)
}

func (r Runtime) QueryAtEnvironmentRoot(ctx context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.QueryAtEnvironmentRoot(ctx, target, name, args...)
}

func (r Runtime) QueryAtEnvironmentRootInput(ctx context.Context, target environmentport.Target, stdin []byte, name string, args ...string) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.QueryAtEnvironmentRootInput(ctx, target, stdin, name, args...)
}

func (r Runtime) SyncFiles(ctx context.Context, target environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	runtime, err := r.forTarget(target)
	if err != nil {
		return err
	}
	return runtime.SyncFiles(ctx, target, directory, files, pruneSuffix)
}

func (r Runtime) forTarget(target environmentport.Target) (deploymentport.Runtime, error) {
	switch target.Environment.TargetType {
	case model.EnvironmentTargetTypeLocal:
		if !target.Environment.IsLocal() || r.local == nil {
			return nil, errors.New("local deployment runtime is not configured")
		}
		return r.local, nil
	case model.EnvironmentTargetTypeSSH:
		if !target.Environment.IsSSH() || r.ssh == nil {
			return nil, errors.New("SSH deployment runtime is not configured")
		}
		return r.ssh, nil
	default:
		return nil, errors.New("environment target type is invalid")
	}
}

func (r Runtime) ResolveDirectory(ctx context.Context, target environmentport.Target, location deploymentport.ServiceLocation) (string, error) {
	runtime, err := r.forTarget(target)
	if err != nil {
		return "", err
	}
	return runtime.ResolveDirectory(ctx, target, location)
}
