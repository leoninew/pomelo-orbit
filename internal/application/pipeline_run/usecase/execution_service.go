package pipelinerunsvc

import (
	"log/slog"
	"time"

	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	repositoryport "github.com/leoninew/pomelo-orbit/internal/application/repository/port"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// ExecutionService owns queued execution and terminal cleanup, with no command API.
type ExecutionService struct {
	executionStore    pipelineExecutionStore
	environments      repository.EnvironmentStore
	targetResolver    environmentport.TargetResolver
	versionForker     applicationport.BuildVersionForker
	transactionRunner pipelinerunport.TransactionRunner
	workspace         pipelinerunport.Workspace
	executionLogStore pipelinerunport.ExecutionLogStore
	secretKey         string
	logger            *slog.Logger
	executionTimeout  time.Duration
	pollInterval      time.Duration
	runner            pipelinerunport.ContainerRunner
	remoteRuntime     pipelinerunport.RemoteRuntimeFactory
	localSource       repositoryport.LocalDirectorySource
}

func NewExecutionService(credential repository.CredentialStore, repos repository.RepositoryStore, pipeline repository.PipelineStore, pipelineRun repository.PipelineRunStore, environments repository.EnvironmentStore, targetResolver environmentport.TargetResolver, versionForker applicationport.BuildVersionForker, transactionRunner pipelinerunport.TransactionRunner, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, executionTimeout time.Duration, pollInterval time.Duration, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.ExecutionLogStore, localSource repositoryport.LocalDirectorySource) ExecutionService {
	store := stores{credential: credential, repository: repos, pipeline: pipeline, pipelineRun: pipelineRun}
	return ExecutionService{executionStore: store, environments: environments, targetResolver: targetResolver, versionForker: versionForker, transactionRunner: transactionRunner, workspace: workspace, secretKey: secretKey, logger: logger, executionTimeout: executionTimeout, pollInterval: pollInterval, runner: runner, executionLogStore: logStore, localSource: localSource}
}

func (s ExecutionService) WithRemoteRuntime(runtime pipelinerunport.RemoteRuntimeFactory) ExecutionService {
	s.remoteRuntime = runtime
	return s
}
