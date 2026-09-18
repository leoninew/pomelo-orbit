package localrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/common/workspacepath"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// PhysicalPathResolver maps a control-plane-visible path to the equivalent
// Docker-daemon-visible path when Orbit itself runs in a container.
type PhysicalPathResolver func(context.Context, string) (string, error)

type Runtime struct {
	pathResolver PhysicalPathResolver
	locks        sync.Map
}

func NewRuntime(pathResolver PhysicalPathResolver) *Runtime {
	return &Runtime{pathResolver: pathResolver}
}

func (r *Runtime) ServiceDir(target environmentport.Target, serviceCode string) (string, error) {
	root, err := r.localWorkspaceRoot(target)
	if err != nil {
		return "", err
	}
	if !safePathSegment(serviceCode) {
		return "", errors.New("invalid service code")
	}
	return workspacepath.ServiceRoot(root, serviceCode), nil
}

func (r *Runtime) ServiceDirExists(ctx context.Context, target environmentport.Target, serviceCode string) (bool, error) {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(serviceDir)
	if err == nil {
		return info.IsDir(), nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (r *Runtime) ComposeMountSourceDir(ctx context.Context, target environmentport.Target, serviceCode string) (string, error) {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return "", err
	}
	if r.pathResolver == nil {
		return "", errors.New("local Docker daemon path resolver is not configured")
	}
	return r.pathResolver(ctx, serviceDir)
}

func (r *Runtime) StageWorkspace(ctx context.Context, target environmentport.Target, workspace deploymentport.Workspace) error {
	serviceDir, err := r.ServiceDir(target, workspace.ServiceCode)
	if err != nil {
		return err
	}
	unlock := r.lock(serviceDir)
	defer unlock()
	if err := os.MkdirAll(serviceDir, 0o750); err != nil {
		return fmt.Errorf("create local service workspace: %w", err)
	}
	for _, directory := range workspace.Directories {
		if err := ensureWorkspacePath(serviceDir, directory); err != nil {
			return err
		}
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return fmt.Errorf("create local mount directory: %w", err)
		}
	}
	for _, file := range workspace.Files {
		if err := r.writeFile(serviceDir, file); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "docker-compose.yml"), []byte(workspace.Compose), 0o644); err != nil {
		return fmt.Errorf("write local compose configuration: %w", err)
	}
	return nil
}

func (r *Runtime) Run(ctx context.Context, target environmentport.Target, serviceCode string, log io.Writer, name string, args ...string) error {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return err
	}
	return run(ctx, serviceDir, log, name, args...)
}

func (r *Runtime) Query(ctx context.Context, target environmentport.Target, serviceCode string, name string, args ...string) (string, error) {
	serviceDir, err := r.ServiceDir(target, serviceCode)
	if err != nil {
		return "", err
	}
	return query(ctx, serviceDir, nil, name, args...)
}

func (r *Runtime) QueryAtEnvironmentRoot(ctx context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	root, err := r.localWorkspaceRoot(target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", fmt.Errorf("create local environment workspace: %w", err)
	}
	return query(ctx, root, nil, name, args...)
}

func (r *Runtime) QueryAtEnvironmentRootInput(ctx context.Context, target environmentport.Target, stdin []byte, name string, args ...string) (string, error) {
	root, err := r.localWorkspaceRoot(target)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return "", fmt.Errorf("create local environment workspace: %w", err)
	}
	return query(ctx, root, stdin, name, args...)
}

func (r *Runtime) SyncFiles(ctx context.Context, target environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	root, err := r.localWorkspaceRoot(target)
	if err != nil {
		return err
	}
	directory = filepath.Clean(directory)
	if directory == "." || !pathWithin(root, directory) {
		return errors.New("local sync directory is outside the deployment workspace")
	}
	unlock := r.lock(directory)
	defer unlock()
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return fmt.Errorf("create local sync directory: %w", err)
	}
	keep := make(map[string]struct{}, len(files))
	for _, file := range files {
		if filepath.Dir(filepath.Clean(file.Path)) != directory || !safePathSegment(filepath.Base(file.Path)) {
			return errors.New("local sync file must be a direct child of the sync directory")
		}
		keep[filepath.Base(file.Path)] = struct{}{}
		if err := r.writeFile(directory, file); err != nil {
			return err
		}
	}
	if pruneSuffix == "" {
		return nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("inspect local sync directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), pruneSuffix) {
			continue
		}
		if _, found := keep[entry.Name()]; found {
			continue
		}
		if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
			return fmt.Errorf("prune local sync file: %w", err)
		}
	}
	return nil
}

func (r *Runtime) ProbeLocal(ctx context.Context, environment model.Environment) error {
	root, err := r.localWorkspaceRoot(environmentport.Target{Environment: environment})
	if err != nil {
		return localProbeError{diagnostic: "Local Environment workspace_root must be configured."}
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return localProbeError{diagnostic: "Local deployment workspace is unavailable."}
	}
	output, err := query(ctx, root, nil, "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return localProbeError{diagnostic: "Local Docker daemon is unavailable: " + strings.TrimSpace(output)}
	}
	output, err = query(ctx, root, nil, "docker", "compose", "version")
	if err != nil {
		return localProbeError{diagnostic: "Local Docker Compose is unavailable: " + strings.TrimSpace(output)}
	}
	return nil
}

type localProbeError struct{ diagnostic string }

func (e localProbeError) Error() string           { return e.diagnostic }
func (e localProbeError) ProbeDiagnostic() string { return e.diagnostic }

func (r *Runtime) localWorkspaceRoot(target environmentport.Target) (string, error) {
	if r == nil {
		return "", errors.New("local deployment runtime is not configured")
	}
	if !target.Environment.IsLocal() {
		return "", errors.New("local deployment runtime received a non-local environment")
	}
	root := strings.TrimSpace(target.Environment.WorkspaceRoot)
	if root == "" {
		return "", errors.New("local Environment workspace_root is required")
	}
	expanded, err := workspacepath.ExpandLocalHomePath(root)
	if err != nil {
		return "", err
	}
	return expanded, nil
}

func (r *Runtime) writeFile(serviceDir string, file deploymentport.WorkspaceFile) error {
	path := filepath.Clean(file.Path)
	if err := ensureWorkspacePath(serviceDir, path); err != nil {
		return err
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fmt.Errorf("local workspace path %s is a directory", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect local workspace file: %w", err)
	}
	if file.IgnoreIfExists {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect local workspace file: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create local workspace file parent: %w", err)
	}
	mode := fs.FileMode(file.Mode)
	if mode == 0 {
		mode = 0o644
	}
	if err := os.WriteFile(path, file.Content, mode); err != nil {
		return fmt.Errorf("write local workspace file: %w", err)
	}
	return nil
}

func (r *Runtime) lock(key string) func() {
	value, _ := r.locks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func ensureWorkspacePath(root, value string) error {
	if !pathWithin(root, value) {
		return errors.New("local workspace path is outside the service directory")
	}
	return nil
}

func pathWithin(root, value string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(value))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func safePathSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "/\\\r\n")
}

func run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("command is required")
	}
	commandText := strings.Join(append([]string{name}, args...), " ")
	if _, err := fmt.Fprintf(log, "Running: %s\n", commandText); err != nil {
		return err
	}
	var output bytes.Buffer
	writer := io.MultiWriter(log, &output)
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = cwd
	command.Stdout = writer
	command.Stderr = writer
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w\n%s", commandText, err, strings.TrimSpace(output.String()))
	}
	return nil
}

func query(ctx context.Context, cwd string, stdin []byte, name string, args ...string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("command is required")
	}
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = cwd
	if len(stdin) > 0 {
		command.Stdin = bytes.NewReader(stdin)
	}
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	return output.String(), err
}
