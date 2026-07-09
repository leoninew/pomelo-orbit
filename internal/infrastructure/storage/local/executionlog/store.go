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
