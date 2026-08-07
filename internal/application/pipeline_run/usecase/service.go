package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	applicationport "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/port"
	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	pipelinesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/usecase"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	repositoryport "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	store             stores
	executionStore    pipelineExecutionStore
	versionForker     applicationport.BuildVersionForker
	transactionRunner pipelinerunport.TransactionRunner
	dispatcher        pipelinerunport.PipelineRunDispatcher
	workspace         pipelinerunport.Workspace
	logStore          pipelinerunport.LogReader
	executionLogStore pipelinerunport.ExecutionLogStore
	secretKey         string
	logger            *slog.Logger
	executionTimeout  time.Duration
	runner            pipelinerunport.ContainerRunner
	localSource       repositoryport.LocalDirectorySource
}

type pipelineExecutionStore interface {
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	CreateArtifact(ctx context.Context, artifact model.Artifact) error
	CommandArtifactByRunStageAndName(ctx context.Context, pipelineRunId string, pipelineStageId string, name string) (model.Artifact, error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error)
	PipelineRunVersionBinding(ctx context.Context, pipelineRunId string) (model.PipelineRunVersionBinding, error)
	CompletePipelineRunVersionBinding(ctx context.Context, pipelineRunId string, generatedVersionId string, generatedVersionLabel string) error
}

type stores struct {
	project     repository.ProjectReader
	credential  repository.CredentialStore
	repository  repository.RepositoryStore
	pipeline    repository.PipelineStore
	pipelineRun repository.PipelineRunStore
	application repository.ApplicationStore
}

func New(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, pipeline repository.PipelineStore, pipelineRun repository.PipelineRunStore, application repository.ApplicationStore, versionForker applicationport.BuildVersionForker, transactionRunner pipelinerunport.TransactionRunner, dispatcher pipelinerunport.PipelineRunDispatcher, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.LogReader, localSource repositoryport.LocalDirectorySource) Service {
	store := stores{project: project, credential: credential, repository: repos, pipeline: pipeline, pipelineRun: pipelineRun, application: application}
	return Service{store: store, executionStore: store, versionForker: versionForker, transactionRunner: transactionRunner, dispatcher: dispatcher, workspace: workspace, secretKey: secretKey, logger: logger, runner: runner, logStore: logStore, localSource: localSource}
}

func NewExecutionService(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, pipeline repository.PipelineStore, pipelineRun repository.PipelineRunStore, application repository.ApplicationStore, versionForker applicationport.BuildVersionForker, transactionRunner pipelinerunport.TransactionRunner, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, executionTimeout time.Duration, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.ExecutionLogStore, localSource repositoryport.LocalDirectorySource) Service {
	store := stores{project: project, credential: credential, repository: repos, pipeline: pipeline, pipelineRun: pipelineRun, application: application}
	return Service{store: store, executionStore: store, versionForker: versionForker, transactionRunner: transactionRunner, workspace: workspace, secretKey: secretKey, logger: logger, executionTimeout: executionTimeout, runner: runner, logStore: logStore, executionLogStore: logStore, localSource: localSource}
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, id)
}
func (s stores) Credential(ctx context.Context, id string) (model.Credential, error) {
	return s.credential.Credential(ctx, id)
}
func (s stores) Pipeline(ctx context.Context, id string) (model.Pipeline, error) {
	return s.pipeline.Pipeline(ctx, id)
}
func (s stores) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return s.pipeline.PipelineSnapshot(ctx, id)
}
func (s stores) LatestPipelineSnapshot(ctx context.Context, pipelineId string) (model.PipelineSnapshot, error) {
	return s.pipeline.LatestPipelineSnapshot(ctx, pipelineId)
}
func (s stores) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	return s.pipeline.CreatePipelineSnapshot(ctx, snapshot)
}
func (s stores) PipelineStages(ctx context.Context, pipelineId string) ([]model.PipelineStage, error) {
	return s.pipeline.PipelineStages(ctx, pipelineId)
}
func (s stores) Application(ctx context.Context, id string) (model.Application, error) {
	return s.application.Application(ctx, id)
}
func (s stores) Version(ctx context.Context, id string) (model.Version, error) {
	return s.application.Version(ctx, id)
}
func (s stores) LatestVersionByApplication(ctx context.Context, id string) (model.Version, error) {
	return s.application.LatestVersionByApplication(ctx, id)
}
func (s stores) VersionComponentsByVersion(ctx context.Context, id string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, id)
}
func (s stores) ListPipelineRuns(ctx context.Context, project, repositoryID, pipelineID string, from, to *time.Time, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRuns(ctx, project, repositoryID, pipelineID, from, to, page, perPage)
}
func (s stores) ListPipelineRunsByPipeline(ctx context.Context, pipelineID string, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRunsByPipeline(ctx, pipelineID, page, perPage)
}
func (s stores) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	return s.pipelineRun.PipelineRun(ctx, id)
}
func (s stores) ListPipelineStageRuns(ctx context.Context, id string) ([]model.PipelineStageRun, error) {
	return s.pipelineRun.ListPipelineStageRuns(ctx, id)
}
func (s stores) PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error) {
	return s.pipelineRun.PipelineStageRun(ctx, id)
}
func (s stores) ListArtifacts(ctx context.Context, project, repositoryID, pipelineID string, page, perPage int, search string) (repository.Page[model.Artifact], error) {
	return s.pipelineRun.ListArtifacts(ctx, project, repositoryID, pipelineID, page, perPage, search)
}
func (s stores) ListArtifactsByRun(ctx context.Context, project *string, runID string) ([]model.Artifact, error) {
	return s.pipelineRun.ListArtifactsByRun(ctx, project, runID)
}
func (s stores) Artifact(ctx context.Context, id string) (model.Artifact, error) {
	return s.pipelineRun.Artifact(ctx, id)
}
func (s stores) CreatePipelineRun(ctx context.Context, run model.PipelineRun, binding *model.PipelineRunVersionBinding) error {
	return s.pipelineRun.CreatePipelineRun(ctx, run, binding)
}
func (s stores) RepositoryHasRunningPipelineRun(ctx context.Context, id string) (bool, error) {
	return s.pipelineRun.RepositoryHasRunningPipelineRun(ctx, id)
}
func (s stores) PipelineRunVersionBinding(ctx context.Context, runID string) (model.PipelineRunVersionBinding, error) {
	return s.pipelineRun.PipelineRunVersionBinding(ctx, runID)
}
func (s stores) CancelPipelineRun(ctx context.Context, id string) error {
	return s.pipelineRun.CancelPipelineRun(ctx, id)
}
func (s stores) MarkPipelineRunRunning(ctx context.Context, id string) error {
	return s.pipelineRun.MarkPipelineRunRunning(ctx, id)
}
func (s stores) CompletePipelineRun(ctx context.Context, id, statusValue, message string) error {
	return s.pipelineRun.CompletePipelineRun(ctx, id, statusValue, message)
}
func (s stores) InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return s.pipelineRun.InsertPipelineStageRun(ctx, stage)
}
func (s stores) UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return s.pipelineRun.UpdatePipelineStageRun(ctx, stage)
}
func (s stores) CreateArtifact(ctx context.Context, artifact model.Artifact) error {
	return s.pipelineRun.CreateArtifact(ctx, artifact)
}
func (s stores) CommandArtifactByRunStageAndName(ctx context.Context, runID, stageID, name string) (model.Artifact, error) {
	return s.pipelineRun.CommandArtifactByRunStageAndName(ctx, runID, stageID, name)
}
func (s stores) CompletePipelineRunVersionBinding(ctx context.Context, runID, versionID, label string) error {
	return s.pipelineRun.CompletePipelineRunVersionBinding(ctx, runID, versionID, label)
}

func (s Service) TriggerPipeline(ctx context.Context, userId, pipelineId string, input pipelinerundto.PipelineRunTriggerInput) (pipelinerundto.PipelineRunDetail, error) {
	pipeline, err := s.pipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if pipeline.Kind != model.PipelineKindApplication {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "template pipelines cannot run")
	}
	return s.createPipelineRun(ctx, pipeline, input.TriggerRef, input.Variables, nil)
}

// PreviewPipelineRunVariables resolves a manual trigger's variable values
// without creating a snapshot, run, task, or version binding.
func (s Service) PreviewPipelineRunVariables(ctx context.Context, userID, pipelineID string, input pipelinerundto.PipelineRunVariablePreviewInput) (pipelinerundto.PipelineRunVariablePreview, error) {
	pipeline, err := s.pipelineForUser(ctx, userID, pipelineID)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	if pipeline.Kind != model.PipelineKindApplication {
		return pipelinerundto.PipelineRunVariablePreview{}, apperror.New(apperror.KindValidation, "template pipelines cannot run")
	}
	if pipeline.RepositoryId == nil {
		return pipelinerundto.PipelineRunVariablePreview{}, apperror.New(apperror.KindValidation, "application pipeline identity is incomplete")
	}
	repo, err := s.repositoryForPipeline(ctx, pipeline)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	ref := strings.TrimSpace(input.TriggerRef)
	if ref == "" {
		ref = repo.DefaultBranch
	}
	ref, err = s.pipelineRunRef(ctx, repo, ref)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	stages, err := s.store.PipelineStages(ctx, pipeline.Id)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	declarations, err := pipelinevariable.ResolvePipelineVariableDeclarations(stages, pipeline.VariableDeclarations)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	snapshot, _, err := resolvePipelineRunVariableSnapshot(repo, pipeline, ref, input.Variables, declarations)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	resolved, err := pipelineRunVariables(snapshot)
	if err != nil {
		return pipelinerundto.PipelineRunVariablePreview{}, err
	}
	return pipelinerundto.PipelineRunVariablePreview{TriggerRef: ref, VariableDeclarations: resolved}, nil
}

func (s Service) RetryPipelineRun(ctx context.Context, userId, runId string) (pipelinerundto.PipelineRunDetail, error) {
	original, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if original.Status != status.WorkStatusFaulted && original.Status != status.WorkStatusRanToCompletion {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot retry run with status "+original.Status)
	}
	pipeline, err := s.pipelineForUser(ctx, userId, original.PipelineId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	overrides, err := pipelineRunRuntimeOverrides(original.VariablesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline run variables", err)
	}
	return s.createPipelineRun(ctx, pipeline, original.TriggerRef, overrides, &original.Id)
}

func (s Service) createPipelineRun(ctx context.Context, pipeline model.Pipeline, requestedRef string, overrides map[string]string, retryOf *string) (pipelinerundto.PipelineRunDetail, error) {
	if pipeline.RepositoryId == nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "application pipeline identity is incomplete")
	}
	repo, err := s.repositoryForPipeline(ctx, pipeline)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if err := s.ensureRepositoryHasNoRunningPipelineRun(ctx, repo.Id); err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	snapshot, err := pipelinesvc.GetOrCreatePipelineSnapshot(ctx, s.store, pipeline)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	ref := strings.TrimSpace(requestedRef)
	if ref == "" {
		ref = repo.DefaultBranch
	}
	ref, err = s.pipelineRunRef(ctx, repo, ref)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	variables, err := buildPipelineRunVariables(repo, pipeline, snapshot, ref, overrides)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	binding, err := s.resolvePipelineRunVersionBinding(ctx, snapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	run := model.PipelineRun{Id: idutil.NewId(), ProjectId: pipeline.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, PipelineId: pipeline.Id, PipelineName: pipeline.Name, PipelineVersion: pipeline.Version, Trigger: "manual", TriggerRef: ref, VariablesSnapshot: variables, Status: status.WorkStatusWaitingToRun, RetryOf: retryOf}
	if err := s.store.CreatePipelineRun(ctx, run, binding); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if s.dispatcher != nil {
		if err := s.dispatcher.DispatchPipelineRun(ctx, pipelinerundto.PipelineRunDispatchInput{PipelineRunId: run.Id}); err != nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
		}
	}
	return s.pipelineRunDetail(ctx, run, false)
}

func (s Service) ListPipelineRuns(ctx context.Context, userId string, input pipelinerundto.PipelineRunListInput) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	projectID := strings.TrimSpace(input.ProjectId)
	if projectID == "" {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectID, userId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	from, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	to, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	if err := s.ensureRunRepositoryFilter(ctx, projectID, input.RepositoryId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	if err := s.ensureRunPipelineFilter(ctx, projectID, input.PipelineId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRuns(ctx, projectID, input.RepositoryId, input.PipelineId, from, to, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline runs", err)
	}
	return s.pipelineRunDetails(ctx, items, false)
}

func (s Service) PipelineRunForUser(ctx context.Context, userId, runId string) (pipelinerundto.PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	return s.pipelineRunDetail(ctx, run, true)
}

func (s Service) ListPipelineRunArtifacts(ctx context.Context, userId, runId string) ([]model.Artifact, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListArtifactsByRun(ctx, run.ProjectId, run.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list artifacts", err)
	}
	return items, nil
}

func (s Service) CancelPipelineRun(ctx context.Context, userId, runId string) (pipelinerundto.PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if run.Status != status.WorkStatusWaitingToRun && run.Status != status.WorkStatusRunning {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot cancel run with status "+run.Status)
	}
	if err := s.store.CancelPipelineRun(ctx, run.Id); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel pipeline run", err)
	}
	updated, err := s.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, updated, true)
}

func (s Service) PipelineStageLog(ctx context.Context, userId, runID, stageRunID string, offset int) (pipelinerundto.PipelineStageLog, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runID)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, err
	}
	if offset < 0 {
		return pipelinerundto.PipelineStageLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	stageRun, err := s.store.PipelineStageRun(ctx, strings.TrimSpace(stageRunID))
	if errors.Is(err, repository.ErrNotFound) || (err == nil && stageRun.PipelineRunId != run.Id) {
		return pipelinerundto.PipelineStageLog{Offset: offset, IsComplete: true}, nil
	}
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage run", err)
	}
	content, next, err := s.logStore.Read(s.workspace.StageLogPath(run.Id, stageRun.Id), offset)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to read stage log", err)
	}
	return pipelinerundto.PipelineStageLog{Logs: string(content), Offset: next, IsComplete: pipelineRunStatusComplete(stageRun.Status)}, nil
}

func (s Service) loadPipelineRunForUser(ctx context.Context, userId, runID string) (model.PipelineRun, error) {
	run, err := s.store.PipelineRun(ctx, strings.TrimSpace(runID))
	if errors.Is(err, repository.ErrNotFound) {
		return model.PipelineRun{}, apperror.New(apperror.KindNotFound, "Pipeline run "+runID+" not found")
	}
	if err != nil {
		return model.PipelineRun{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	projectId, err := requiredProjectID(run.ProjectId, "Pipeline run")
	if err != nil {
		return model.PipelineRun{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.PipelineRun{}, err
	}
	return run, nil
}

func (s Service) pipelineForUser(ctx context.Context, userID, pipelineID string) (model.Pipeline, error) {
	pipeline, err := s.store.Pipeline(ctx, strings.TrimSpace(pipelineID))
	if errors.Is(err, repository.ErrNotFound) {
		return model.Pipeline{}, apperror.New(apperror.KindNotFound, "Pipeline "+pipelineID+" not found")
	}
	if err != nil {
		return model.Pipeline{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline", err)
	}
	projectId, err := requiredProjectID(pipeline.ProjectId, "Pipeline")
	if err != nil {
		return model.Pipeline{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userID); err != nil {
		return model.Pipeline{}, err
	}
	return pipeline, nil
}

func (s Service) repositoryForPipeline(ctx context.Context, pipeline model.Pipeline) (model.Repository, error) {
	if pipeline.RepositoryId == nil {
		return model.Repository{}, apperror.New(apperror.KindValidation, "application pipeline has no repository")
	}
	repo, err := s.store.Repository(ctx, *pipeline.RepositoryId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Repository{}, apperror.New(apperror.KindNotFound, "Pipeline repository not found")
	}
	if err != nil {
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline repository", err)
	}
	if repo.ProjectId == nil || pipeline.ProjectId == nil || *repo.ProjectId != *pipeline.ProjectId {
		return model.Repository{}, apperror.New(apperror.KindValidation, "Pipeline repository must belong to pipeline project")
	}
	return repo, nil
}

func (s Service) resolvePipelineRunVersionBinding(ctx context.Context, snapshot model.PipelineSnapshot) (*model.PipelineRunVersionBinding, error) {
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	mappings := componentMappings(stages)
	if len(mappings) == 0 {
		return nil, nil
	}
	if snapshot.ApplicationId == nil || snapshot.ApplicationName == nil {
		return nil, apperror.New(apperror.KindValidation, "component-bound artifacts require an application binding")
	}
	app, err := s.store.Application(ctx, *snapshot.ApplicationId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(apperror.KindNotFound, "Pipeline application not found")
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline application", err)
	}
	if app.ProjectId == nil || snapshot.ProjectId == nil || *app.ProjectId != *snapshot.ProjectId {
		return nil, apperror.New(apperror.KindValidation, "Pipeline application must belong to pipeline project")
	}
	var source model.Version
	switch strategy := snapshot.VersionForkStrategy; {
	case strategy == nil:
		return nil, apperror.New(apperror.KindValidation, "component-bound artifacts require a version strategy")
	case *strategy == model.VersionForkStrategyLatest:
		source, err = s.store.LatestVersionByApplication(ctx, *snapshot.ApplicationId)
	case *strategy == model.VersionForkStrategyFixed && snapshot.FixedVersionId != nil:
		source, err = s.store.Version(ctx, *snapshot.FixedVersionId)
	default:
		return nil, apperror.New(apperror.KindValidation, "invalid pipeline version strategy")
	}
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(apperror.KindNotFound, "Pipeline source version not found")
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline source version", err)
	}
	if source.ApplicationId != *snapshot.ApplicationId {
		return nil, apperror.New(apperror.KindValidation, "Pipeline source version does not belong to its application")
	}
	components, err := s.store.VersionComponentsByVersion(ctx, source.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load source version components", err)
	}
	for _, mapping := range mappings {
		if !versionContainsComponent(components, mapping.ComponentName) {
			return nil, apperror.New(apperror.KindValidation, "Source version does not contain component "+mapping.ComponentName)
		}
	}
	return &model.PipelineRunVersionBinding{ApplicationId: app.Id, ApplicationName: app.Name, SourceVersionId: source.Id, SourceVersionLabel: source.Label}, nil
}

func (s Service) ensureRepositoryHasNoRunningPipelineRun(ctx context.Context, repositoryID string) error {
	running, err := s.store.RepositoryHasRunningPipelineRun(ctx, repositoryID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check running pipeline runs", err)
	}
	if running {
		return apperror.New(apperror.KindValidation, "Repository already has a running pipeline run")
	}
	return nil
}
func (s Service) ensureRunRepositoryFilter(ctx context.Context, projectID, repositoryID string) error {
	if strings.TrimSpace(repositoryID) == "" {
		return nil
	}
	repo, err := s.store.Repository(ctx, repositoryID)
	if errors.Is(err, repository.ErrNotFound) || repo.ProjectId == nil || *repo.ProjectId != projectID {
		return apperror.New(apperror.KindNotFound, "Repository "+repositoryID+" not found")
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return nil
}
func (s Service) ensureRunPipelineFilter(ctx context.Context, projectID, pipelineID string) error {
	if strings.TrimSpace(pipelineID) == "" {
		return nil
	}
	pipeline, err := s.store.Pipeline(ctx, pipelineID)
	if errors.Is(err, repository.ErrNotFound) || pipeline.ProjectId == nil || *pipeline.ProjectId != projectID {
		return apperror.New(apperror.KindNotFound, "Pipeline "+pipelineID+" not found")
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline", err)
	}
	return nil
}
func (s Service) pipelineRunDetails(ctx context.Context, page repository.Page[model.PipelineRun], includeStages bool) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	items := make([]pipelinerundto.PipelineRunDetail, 0, len(page.Items))
	for _, run := range page.Items {
		detail, err := s.pipelineRunDetail(ctx, run, includeStages)
		if err != nil {
			return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[pipelinerundto.PipelineRunDetail]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}
func (s Service) pipelineRunDetail(ctx context.Context, run model.PipelineRun, includeStages bool) (pipelinerundto.PipelineRunDetail, error) {
	variables, err := pipelineRunVariables(run.VariablesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	detail := pipelinerundto.PipelineRunDetail{Run: run, VariablesSnapshot: variables}
	if includeStages {
		stages, err := s.store.ListPipelineStageRuns(ctx, run.Id)
		if err != nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage runs", err)
		}
		detail.PipelineStageRuns = stages
	}
	binding, err := s.store.PipelineRunVersionBinding(ctx, run.Id)
	if err == nil {
		detail.VersionBinding = &binding
	} else if !errors.Is(err, repository.ErrNotFound) {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load run version binding", err)
	}
	return detail, nil
}
func (s Service) pipelineRunRef(ctx context.Context, repo model.Repository, ref string) (string, error) {
	if repo.RepositoryType == "" || repo.RepositoryType == model.RepositoryTypeRemoteGit {
		if strings.TrimSpace(repo.RepositoryUrl) == "" {
			return "", apperror.New(apperror.KindValidation, "repository_url is required for remote Git repositories")
		}
		return ref, nil
	}
	if repo.RepositoryType == model.RepositoryTypeLocalDirectory {
		if s.localSource == nil {
			return "", apperror.New(apperror.KindValidation, "Local directory sources are disabled")
		}
		revision, err := s.localSource.ResolveRevision(ctx, repo.RepositoryUrl, ref)
		if err != nil {
			return "", apperror.New(apperror.KindValidation, "Unable to resolve local source revision: "+err.Error())
		}
		return revision, nil
	}
	return "", apperror.New(apperror.KindValidation, "Unsupported repository_type")
}
func parseOptionalRunTime(value, name string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, name+" must be ISO 8601")
	}
	return &parsed, nil
}
func pipelineRunStatusComplete(value string) bool {
	return value == status.WorkStatusRanToCompletion || value == status.WorkStatusFaulted || value == status.WorkStatusCanceled
}
func pipelineRunVariables(value string) ([]model.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []model.VariableDeclaration{}, nil
	}
	var variables []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline run variables")
	}
	return variables, nil
}
func buildPipelineRunVariables(repo model.Repository, pipeline model.Pipeline, snapshot model.PipelineSnapshot, ref string, overrides map[string]string) (string, error) {
	declarations, err := pipelinevariable.CompleteSnapshotVariableDeclarations(snapshot, pipeline)
	if err != nil {
		return "", err
	}
	data, variables, err := resolvePipelineRunVariableSnapshot(repo, pipeline, ref, overrides, declarations)
	if err != nil {
		return "", err
	}
	if err := pipelinevariable.ValidateRuntimeVariables(variables, declarations); err != nil {
		return "", err
	}
	return data, nil
}

func resolvePipelineRunVariableSnapshot(repo model.Repository, pipeline model.Pipeline, ref string, overrides map[string]string, declarations []model.VariableDeclaration) (string, map[string]any, error) {
	variables, err := pipelinevariable.BuildRuntimeVariables(repo, pipeline, ref, overrides, declarations)
	if err != nil {
		return "", nil, err
	}
	data, err := pipelinevariable.MarshalRuntimeVariableSnapshot(variables, declarations)
	if err != nil {
		return "", nil, err
	}
	return data, variables, nil
}
func pipelineRunRuntimeOverrides(value string) (map[string]string, error) {
	declarations, err := pipelineRunVariables(value)
	if err != nil {
		return nil, err
	}
	overrides := map[string]string{}
	for _, declaration := range declarations {
		if !pipelinevariable.IsPipelineBuiltinVariable(declaration.Name) && pipelinevariable.HasRuntimeValue(declaration.Value) {
			overrides[declaration.Name] = fmt.Sprint(declaration.Value)
		}
	}
	return overrides, nil
}
func versionContainsComponent(components []model.VersionComponent, name string) bool {
	for _, component := range components {
		if component.Name == name {
			return true
		}
	}
	return false
}

type componentMapping struct {
	Stage         model.StageDefinition
	Artifact      model.ArtifactConfig
	ComponentName string
}

func componentMappings(stages []model.StageDefinition) []componentMapping {
	mappings := make([]componentMapping, 0)
	for _, stage := range stages {
		for _, artifact := range stage.Artifacts {
			if artifact.Collector == "docker_image" && artifact.ComponentName != nil {
				mappings = append(mappings, componentMapping{Stage: stage, Artifact: artifact, ComponentName: *artifact.ComponentName})
			}
		}
	}
	return mappings
}
