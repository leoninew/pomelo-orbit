package deploymentworkspace

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
	return &Workspace{logicalDataRoot: filepath.Clean(dataRoot), resolver: resolver}
}

func (w *Workspace) AppDir(appCode string) string {
	return filepath.Join(w.logicalDataRoot, deploymentDataDir, appCode)
}

func (w *Workspace) ServiceDir(appCode string, envCode string, instanceKey string) string {
	return filepath.Join(w.AppDir(appCode), envCode, instanceKey)
}

func (w *Workspace) DeploymentLogPath(appCode string, envCode string, instanceKey string, deploymentID string) string {
	return filepath.Join(w.ServiceDir(appCode, envCode, instanceKey), "deployments", deploymentID+".log")
}

func (w *Workspace) WriteConfig(appCode string, envCode string, instanceKey string, path string, content string) error {
	path = filepath.Join(w.ServiceDir(appCode, envCode, instanceKey), path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		content = normalizeShellScriptLineEndings(content)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		return os.Chmod(path, 0o755)
	}
	return nil
}

func (w *Workspace) RemoveAppDir(appCode string) error {
	return os.RemoveAll(w.AppDir(appCode))
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

func (w *Workspace) PhysicalServiceDir(ctx context.Context, appCode string, envCode string, instanceKey string) (string, error) {
	physicalDataRoot, err := w.PhysicalDataRoot(ctx)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(physicalDataRoot, deploymentDataDir, appCode, envCode, instanceKey)), nil
}

func normalizeShellScriptLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}
