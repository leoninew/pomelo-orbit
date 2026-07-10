package cisvc

import (
	"io"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type ExecutionLogStore interface {
	Writer(logPath string) (io.WriteCloser, error)
	Read(logPath string, offset int) ([]byte, int, error)
}

type Service struct {
	executionStore repository.PipelineExecutionStore
	workspace      *ciworkspace.Workspace
	logStore       ExecutionLogStore
	secretKey      string
	logger         *slog.Logger
	runner         ContainerRunner
}

func NewExecutionService(store repository.PipelineExecutionStore, dataRoot string, secretKey string, logger *slog.Logger, runner ContainerRunner, logStore ExecutionLogStore) Service {
	workspace := ciworkspace.New(dataRoot)
	return Service{executionStore: store, workspace: workspace, logStore: logStore, secretKey: secretKey, logger: logger, runner: runner}
}
