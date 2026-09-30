package sshrunner

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	logport "github.com/leoninew/pomelo-orbit/internal/application/logstream/port"
	"github.com/pkg/sftp"
)

type stageLogReader struct {
	client   *sftp.Client
	path     string
	platform string
	cleanup  func()
	once     sync.Once
}

func (logs *pipelineLogs) OpenReader(ctx context.Context, logPath string) (logport.Reader, error) {
	if !logs.validLogPath(logPath) {
		return nil, errors.New("invalid remote stage log path")
	}
	client, cleanup, err := logs.workspace.runtime.openSFTP(ctx, logs.workspace.target)
	if err != nil {
		return nil, err
	}
	return &stageLogReader{client: client, path: logPath, platform: logs.workspace.target.Environment.SSH.Platform, cleanup: cleanup}, nil
}

func (r *stageLogReader) Read(ctx context.Context, offset int64, limit int) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	file, err := openStageLogFile(ctx, r.platform, func() (*sftp.File, error) { return r.client.Open(r.path) })
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, true, err
	}
	if offset > info.Size() {
		return nil, true, errors.New("log file was truncated")
	}
	buffer := make([]byte, limit)
	n, err := file.ReadAt(buffer, offset)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return buffer[:n], true, err
}

func (r *stageLogReader) Close() error { r.once.Do(r.cleanup); return nil }
