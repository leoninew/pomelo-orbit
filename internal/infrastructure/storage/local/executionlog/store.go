package executionlog

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

type Store struct{}

func (Store) Writer(logPath string) (io.WriteCloser, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &logWriter{file: file}, nil
}

type logWriter struct {
	mu   sync.Mutex
	file *os.File
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Write(p)
}

func (w *logWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (Store) Read(logPath string, offset int) ([]byte, int, error) {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, offset, nil
		}
		return nil, offset, err
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, offset, err
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, offset, err
	}
	return content, offset + len(content), nil
}

// DeploymentStore keeps control-plane execution logs separate from the remote
// service workspace. Docker and deployment files never use this local root.
type DeploymentStore struct {
	Root string
}

func NewDeploymentStore(root string) DeploymentStore {
	return DeploymentStore{Root: filepath.Clean(root)}
}

func (s DeploymentStore) Writer(serviceCode string, deploymentID string) (io.WriteCloser, error) {
	return Store{}.Writer(s.path(serviceCode, deploymentID))
}

func (s DeploymentStore) Read(serviceCode string, deploymentID string, offset int) ([]byte, int, error) {
	return Store{}.Read(s.path(serviceCode, deploymentID), offset)
}

func (s DeploymentStore) Remove(serviceCode string, deploymentID string) error {
	return os.Remove(s.path(serviceCode, deploymentID))
}

func (s DeploymentStore) path(serviceCode string, deploymentID string) string {
	return filepath.Join(s.Root, "logs", serviceCode, deploymentID+".log")
}
