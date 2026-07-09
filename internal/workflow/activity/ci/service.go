package cisvc

import (
	"context"
	"io"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type PipelineExecutionStore interface {
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertStageRun(ctx context.Context, stage model.StageRun) error
	UpdateStageRun(ctx context.Context, stage model.StageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error
}

type ExecutionLogStore interface {
	Writer(logPath string) (io.WriteCloser, error)
	Read(logPath string, offset int) ([]byte, int, error)
}

type Service struct {
	executionStore PipelineExecutionStore
	workspace      *ciworkspace.Workspace
	logStore       ExecutionLogStore
	secretKey      string
	logger         *slog.Logger
	runner         ContainerRunner
}

func NewExecutionService(store PipelineExecutionStore, dataRoot string, secretKey string, logger *slog.Logger, runner ContainerRunner, logStore ExecutionLogStore) Service {
	workspace := ciworkspace.New(dataRoot)
	return Service{executionStore: store, workspace: workspace, logStore: logStore, secretKey: secretKey, logger: logger, runner: runner}
}
