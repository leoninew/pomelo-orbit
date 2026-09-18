// Package workspacepath defines the on-disk layout shared by CI and CD.
package workspacepath

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	PipelineDirName   = "pipeline"
	DeploymentDirName = "deployment"
)

func PipelineRoot(root string) string {
	return filepath.Join(root, PipelineDirName)
}

func DeploymentRoot(root string) string {
	return filepath.Join(root, DeploymentDirName)
}

func ServiceRoot(root, serviceCode string) string {
	return filepath.Join(DeploymentRoot(root), serviceCode)
}

func RemoteServiceRoot(root, serviceCode string) string {
	return path.Join(path.Join(root, DeploymentDirName), serviceCode)
}

// ExpandLocalHomePath resolves the supported home-relative workspace syntax.
// Other paths are only cleaned; callers remain responsible for validating
// whether they are absolute or otherwise suitable for their target platform.
func ExpandLocalHomePath(value string) (string, error) {
	if !IsHomeWorkspaceRoot(value) {
		return filepath.Clean(value), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	home = strings.TrimSpace(home)
	if home == "" {
		return "", errors.New("user home directory is required")
	}
	if value == "~" {
		return filepath.Clean(home), nil
	}
	return filepath.Clean(filepath.Join(home, strings.TrimPrefix(value, "~/"))), nil
}

func IsHomeWorkspaceRoot(value string) bool {
	return value == "~" || strings.HasPrefix(value, "~/")
}
