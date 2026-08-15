package deploymentworkspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDeploymentLogRemovesOnlyDeploymentLog(t *testing.T) {
	dataRoot := t.TempDir()
	workspace := NewWithResolver(dataRoot, nil)
	logPath := workspace.DeploymentLogPath("demo-default", "deployment-1")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("create deployment log directory: %v", err)
	}
	if err := os.WriteFile(logPath, []byte("deployment output"), 0o644); err != nil {
		t.Fatalf("write deployment log: %v", err)
	}

	if err := workspace.RemoveDeploymentLog("demo-default", "deployment-1"); err != nil {
		t.Fatalf("RemoveDeploymentLog returned error: %v", err)
	}
	if _, err := os.Stat(logPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("log stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Dir(logPath)); err != nil {
		t.Fatalf("deployment log directory must remain: %v", err)
	}
}
