package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"strings"
	"time"

	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	pipelinesvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline/usecase"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	repositoryport "github.com/leoninew/pomelo-orbit/internal/application/repository/port"
	"github.com/leoninew/pomelo-orbit/internal/application/variableview"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type Service struct {
	store             stores
	environments      repository.EnvironmentStore
	targetResolver    environmentport.TargetResolver
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
	pollInterval      time.Duration
	runner            pipelinerunport.ContainerRunner
	remoteRuntime     pipelinerunport.RemoteRuntimeFactory
	localSource       repositoryport.LocalDirectorySource
}

type pipelineExecutionStore interface {
	PipelineRun(ctx context.Context, projectId string, id string) (model.PipelineRun, error)
	ListPipelineStageRuns(ctx context.Context, projectId string, runId string) ([]model.PipelineStageRun, error)
	Repository(ctx context.Context, projectId string, id string) (model.Repository, error)
	Credential(ctx context.Context, projectId string, id string) (model.Credential, error)
	PipelineSnapshot(ctx context.Context, projectId string, id string) (model.PipelineSnapshot, error)
	BeginPipelineRun(ctx context.Context, projectId string, id string) (bool, error)
	CompletePipelineRun(ctx context.Context, projectId string, id string, status string, message string) (bool, error)
	BeginPipelineStageRun(ctx context.Context, projectId string, id string) (bool, error)
	CompletePipelineStageRun(ctx context.Context, projectId string, stage model.PipelineStageRun) (bool, error)
	CreateArtifact(ctx context.Context, artifact model.Artifact) error
	CommandArtifactByRunStageAndName(ctx context.Context, projectId string, pipelineRunId string, pipelineStageId string, name string) (model.Artifact, error)
	ListArtifactsByRun(ctx context.Context, projectId string, runId string) ([]model.Artifact, error)
	PipelineRunVersionBinding(ctx context.Context, projectId string, pipelineRunId string) (model.PipelineRunVersionBinding, error)
	CompletePipelineRunVersionBinding(ctx context.Context, projectId string, pipelineRunId string, generatedVersionId string, generatedVersionLabel string) error
}

type stores struct {
	project     repository.ProjectReader
	credential  repository.CredentialStore
	repository  repository.RepositoryStore
	pipeline    repository.PipelineStore
	pipelineRun repository.PipelineRunStore
	application repository.ApplicationStore
}

func New(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, pipeline repository.PipelineStore, pipelineRun repository.PipelineRunStore, application repository.ApplicationStore, environments repository.EnvironmentStore, targetResolver environmentport.TargetResolver, versionForker applicationport.BuildVersionForker, transactionRunner pipelinerunport.TransactionRunner, dispatcher pipelinerunport.PipelineRunDispatcher, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.LogReader, localSource repositoryport.LocalDirectorySource) Service {
	store := stores{project: project, credential: credential, repository: repos, pipeline: pipeline, pipelineRun: pipelineRun, application: application}
	return Service{store: store, executionStore: store, environments: environments, targetResolver: targetResolver, versionForker: versionForker, transactionRunner: transactionRunner, dispatcher: dispatcher, workspace: workspace, secretKey: secretKey, logger: logger, runner: runner, logStore: logStore, localSource: localSource}
}

func NewExecutionService(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, pipeline repository.PipelineStore, pipelineRun repository.PipelineRunStore, application repository.ApplicationStore, environments repository.EnvironmentStore, targetResolver environmentport.TargetResolver, versionForker applicationport.BuildVersionForker, transactionRunner pipelinerunport.TransactionRunner, workspace pipelinerunport.Workspace, secretKey string, logger *slog.Logger, executionTimeout time.Duration, pollInterval time.Duration, runner pipelinerunport.ContainerRunner, logStore pipelinerunport.ExecutionLogStore, localSource repositoryport.LocalDirectorySource) Service {
	store := stores{project: project, credential: credential, repository: repos, pipeline: pipeline, pipelineRun: pipelineRun, application: application}
	return Service{store: store, executionStore: store, environments: environments, targetResolver: targetResolver, versionForker: versionForker, transactionRunner: transactionRunner, workspace: workspace, secretKey: secretKey, logger: logger, executionTimeout: executionTimeout, pollInterval: pollInterval, runner: runner, logStore: logStore, executionLogStore: logStore, localSource: localSource}
}

func (s Service) WithRemoteRuntime(runtime pipelinerunport.RemoteRuntimeFactory) Service {
	s.remoteRuntime = runtime
	return s
}
func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) Repository(ctx context.Context, projectId string, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, projectId, id)
}
func (s stores) Credential(ctx context.Context, projectId string, id string) (model.Credential, error) {
	return s.credential.Credential(ctx, projectId, id)
}
func (s stores) Pipeline(ctx context.Context, projectId string, id string) (model.Pipeline, error) {
	return s.pipeline.Pipeline(ctx, projectId, id)
}
func (s stores) PipelineSnapshot(ctx context.Context, projectId string, id string) (model.PipelineSnapshot, error) {
	return s.pipeline.PipelineSnapshot(ctx, projectId, id)
}
func (s stores) LatestPipelineSnapshot(ctx context.Context, projectId string, pipelineId string) (model.PipelineSnapshot, error) {
	return s.pipeline.LatestPipelineSnapshot(ctx, projectId, pipelineId)
}
func (s stores) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	return s.pipeline.CreatePipelineSnapshot(ctx, snapshot)
}
func (s stores) ApplicationPipelineStages(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStage, error) {
	return s.pipeline.ApplicationPipelineStages(ctx, projectId, pipelineId)
}
func (s stores) TemplatePipelineStageReferences(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStageReference, error) {
	return s.pipeline.TemplatePipelineStageReferences(ctx, projectId, pipelineId)
}
func (s stores) Application(ctx context.Context, projectId, id string) (model.Application, error) {
	return s.application.Application(ctx, projectId, id)
}
func (s stores) Version(ctx context.Context, projectId, id string) (model.Version, error) {
	return s.application.Version(ctx, projectId, id)
}
func (s stores) LatestVersionByApplication(ctx context.Context, projectId, id string) (model.Version, error) {
	return s.application.LatestVersionByApplication(ctx, projectId, id)
}
func (s stores) VersionComponentsByVersion(ctx context.Context, projectId, id string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, projectId, id)
}
func (s stores) ListPipelineRuns(ctx context.Context, project, repositoryId, pipelineId string, from, to *time.Time, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRuns(ctx, project, repositoryId, pipelineId, from, to, page, perPage)
}
func (s stores) ListPipelineRunsByPipeline(ctx context.Context, projectId string, pipelineId string, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRunsByPipeline(ctx, projectId, pipelineId, page, perPage)
}
func (s stores) PipelineRun(ctx context.Context, projectId string, id string) (model.PipelineRun, error) {
	return s.pipelineRun.PipelineRun(ctx, projectId, id)
}
func (s stores) DeletePipelineRun(ctx context.Context, projectId, id string) error {
	return s.pipelineRun.DeletePipelineRun(ctx, projectId, id)
}
func (s stores) ListPipelineStageRuns(ctx context.Context, projectId, id string) ([]model.PipelineStageRun, error) {
	return s.pipelineRun.ListPipelineStageRuns(ctx, projectId, id)
}
func (s stores) PipelineStageRun(ctx context.Context, projectId, id string) (model.PipelineStageRun, error) {
	return s.pipelineRun.PipelineStageRun(ctx, projectId, id)
}
func (s stores) ListArtifacts(ctx context.Context, project, repositoryId, pipelineId string, page, perPage int, search string) (repository.Page[model.Artifact], error) {
	return s.pipelineRun.ListArtifacts(ctx, project, repositoryId, pipelineId, page, perPage, search)
}
func (s stores) ListArtifactsByRun(ctx context.Context, projectId string, runId string) ([]model.Artifact, error) {
	return s.pipelineRun.ListArtifactsByRun(ctx, projectId, runId)
}
func (s stores) Artifact(ctx context.Context, projectId string, id string) (model.Artifact, error) {
	return s.pipelineRun.Artifact(ctx, projectId, id)
}
func (s stores) CreatePipelineRun(ctx context.Context, run model.PipelineRun, binding *model.PipelineRunVersionBinding, stageRuns []model.PipelineStageRun) error {
	return s.pipelineRun.CreatePipelineRun(ctx, run, binding, stageRuns)
}
func (s stores) RepositoryHasActivePipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	return s.pipelineRun.RepositoryHasActivePipelineRun(ctx, projectId, id)
}
func (s stores) HasCDConfigurationReferences(ctx context.Context, projectId string) (bool, error) {
	return s.pipelineRun.HasCDConfigurationReferences(ctx, projectId)
}

// EnsureNoCDConfigurationReferences prevents replacing deployment
// configuration that remains referenced by Pipeline Run bindings.
func (s Service) EnsureNoCDConfigurationReferences(ctx context.Context, userId, projectId string) error {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return err
	}
	referenced, err := s.store.HasCDConfigurationReferences(ctx, projectId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check Pipeline Run CD configuration references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Project Pipeline Run history still references the current CD configuration")
	}
	return nil
}
func (s stores) PipelineRunVersionBinding(ctx context.Context, projectId, runId string) (model.PipelineRunVersionBinding, error) {
	return s.pipelineRun.PipelineRunVersionBinding(ctx, projectId, runId)
}
func (s stores) CancelPipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	return s.pipelineRun.CancelPipelineRun(ctx, projectId, id)
}
func (s stores) CancelRunningPipelineStageRuns(ctx context.Context, projectId, id string) error {
	return s.pipelineRun.CancelRunningPipelineStageRuns(ctx, projectId, id)
}
func (s stores) BeginPipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	return s.pipelineRun.BeginPipelineRun(ctx, projectId, id)
}
func (s stores) CompletePipelineRun(ctx context.Context, projectId, id, statusValue, message string) (bool, error) {
	return s.pipelineRun.CompletePipelineRun(ctx, projectId, id, statusValue, message)
}
func (s stores) BeginPipelineStageRun(ctx context.Context, projectId, id string) (bool, error) {
	return s.pipelineRun.BeginPipelineStageRun(ctx, projectId, id)
}
func (s stores) CompletePipelineStageRun(ctx context.Context, projectId string, stage model.PipelineStageRun) (bool, error) {
	return s.pipelineRun.CompletePipelineStageRun(ctx, projectId, stage)
}
func (s stores) CreateArtifact(ctx context.Context, artifact model.Artifact) error {
	return s.pipelineRun.CreateArtifact(ctx, artifact)
}
func (s stores) CommandArtifactByRunStageAndName(ctx context.Context, projectId, runId, stageId, name string) (model.Artifact, error) {
	return s.pipelineRun.CommandArtifactByRunStageAndName(ctx, projectId, runId, stageId, name)
}
func (s stores) CompletePipelineRunVersionBinding(ctx context.Context, projectId, runId, versionId, label string) error {
	return s.pipelineRun.CompletePipelineRunVersionBinding(ctx, projectId, runId, versionId, label)
}

func (s Service) TriggerPipeline(ctx context.Context, userId string, projectId string, pipelineId string, repositoryRef string) (pipelinerundto.PipelineRunDetail, error) {
	pipeline, err := s.pipelineForUser(ctx, userId, projectId, pipelineId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if pipeline.Kind != model.PipelineKindApplication {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "template pipelines cannot run")
	}
	overrides := pipelinevariable.RuntimeVariableOverrides{}
	if ref := strings.TrimSpace(repositoryRef); ref != "" {
		overrides.Global = map[string]string{"repository_ref": ref}
	}
	return s.createPipelineRun(ctx, projectId, pipeline, overrides, nil)
}

func (s Service) RetryPipelineRun(ctx context.Context, userId string, projectId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
	original, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	pipeline, err := s.pipelineForUser(ctx, userId, projectId, original.PipelineId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	declarations, _, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(original.VariablesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline run variables", err)
	}
	overrides := pipelinevariable.RuntimeVariableOverrides{Global: make(map[string]string), Stage: make(map[string]map[string]string)}
	for _, declaration := range declarations {
		if declaration.Source == "system" || !pipelinevariable.HasRuntimeValue(declaration.Value) {
			continue
		}
		if declaration.StageId != "" {
			if overrides.Stage[declaration.StageId] == nil {
				overrides.Stage[declaration.StageId] = make(map[string]string)
			}
			overrides.Stage[declaration.StageId][declaration.Name] = fmt.Sprint(declaration.Value)
		} else {
			overrides.Global[declaration.Name] = fmt.Sprint(declaration.Value)
		}
	}
	return s.createPipelineRun(ctx, projectId, pipeline, overrides, &original.Id)
}

func (s Service) createPipelineRun(ctx context.Context, projectId string, pipeline model.Pipeline, overrides pipelinevariable.RuntimeVariableOverrides, retryOf *string) (pipelinerundto.PipelineRunDetail, error) {
	if pipeline.RepositoryId == nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "application pipeline identity is incomplete")
	}
	repo, err := s.repositoryForPipeline(ctx, projectId, pipeline)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if err := s.ensureRepositoryHasNoRunningPipelineRun(ctx, projectId, repo.Id); err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, projectId, pipeline.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	if _, _, err := pipelinevariable.ResolveRuntimeVariablesFromPipelineStages(repo, pipeline, stages, overrides); err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if target.Environment.IsSSH() {
		if repo.RepositoryType == model.RepositoryTypeLocalDirectory {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Local directory repositories cannot run on SSH environments; use an HTTPS Git repository")
		}
		remoteURL, err := url.Parse(repo.RepositoryUrl)
		if repo.RepositoryType != model.RepositoryTypeRemoteGit || err != nil || remoteURL.Scheme != "https" || remoteURL.Hostname() == "" {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "SSH pipeline runs require an HTTPS Git repository URL")
		}
		if (target.Environment.SSH.Platform != model.EnvironmentPlatformWindows && target.Environment.SSH.Platform != model.EnvironmentPlatformLinux) || s.remoteRuntime == nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "SSH pipeline execution is not installed for this environment")
		}
	}
	if target.Environment.IsLocal() {
		if err := s.ensureWorkspaceReady(ctx, target.Environment); err != nil {
			return pipelinerundto.PipelineRunDetail{}, err
		}
	}
	snapshot, err := pipelinesvc.GetOrCreatePipelineSnapshot(ctx, s.store, projectId, pipeline, repo)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	variables, ref, err := buildPipelineRunVariables(repo, pipeline, snapshot, overrides)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	binding, err := s.resolvePipelineRunVersionBinding(ctx, projectId, snapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	run := model.PipelineRun{Id: idutil.NewId(), ProjectId: pipeline.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, PipelineId: pipeline.Id, PipelineName: pipeline.Name, PipelineVersion: pipeline.Version, Trigger: "manual", RepositoryRef: ref, VariablesSnapshot: variables, Status: status.WorkStatusWaitingToRun, RetryOf: retryOf}
	applyRunTargetSnapshot(&run, target)
	stageRuns, err := pipelineStageRuns(run.Id, snapshot.StagesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if err := s.store.CreatePipelineRun(ctx, run, binding, stageRuns); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if s.dispatcher != nil {
		if err := s.dispatcher.DispatchPipelineRun(ctx, pipelinerundto.PipelineRunDispatchInput{PipelineRunId: run.Id, ProjectId: projectId}); err != nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
		}
	}
	return s.pipelineRunDetail(ctx, projectId, run, false)
}

func (s Service) ListPipelineRuns(ctx context.Context, userId string, input pipelinerundto.PipelineRunListInput) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	projectId := input.ProjectId
	if projectId == "" {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
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
	if err := s.ensureRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	if err := s.ensureRunPipelineFilter(ctx, projectId, input.PipelineId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRuns(ctx, projectId, input.RepositoryId, input.PipelineId, from, to, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline runs", err)
	}
	return s.pipelineRunDetails(ctx, projectId, items, false)
}

func (s Service) PipelineRunForUser(ctx context.Context, userId string, projectId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	return s.pipelineRunDetail(ctx, projectId, run, true)
}

func (s Service) DeletePipelineRun(ctx context.Context, userId string, projectId string, runId string) error {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return err
	}
	if !status.WorkStatusIsComplete(run.Status) {
		return apperror.New(apperror.KindValidation, "Cannot delete pipeline run with status "+run.Status)
	}
	workspace, err := s.workspaceForRun(ctx, projectId, run)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to resolve pipeline workspace", err)
	}
	if run.EnvironmentTargetType != nil && *run.EnvironmentTargetType == model.EnvironmentTargetTypeSSH {
		if err := workspace.RemoveRunFiles(run.Id); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return apperror.Wrap(apperror.KindInternal, "Failed to delete remote pipeline run files", err)
		}
	}
	deleteRecord := func(txCtx context.Context) error {
		return s.store.DeletePipelineRun(txCtx, projectId, run.Id)
	}
	var deleteErr error
	if s.transactionRunner != nil {
		deleteErr = s.transactionRunner.RunInTransaction(ctx, deleteRecord)
	} else {
		deleteErr = deleteRecord(ctx)
	}
	if deleteErr != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline run", deleteErr)
	}
	if run.EnvironmentTargetType == nil || *run.EnvironmentTargetType != model.EnvironmentTargetTypeSSH {
		if err := workspace.RemoveRunFiles(run.Id); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				s.warnPipelineRunFileCleanupSkipped(run.Id, "pipeline run file or directory does not exist")
			} else {
				return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline run files", err)
			}
		}
	}
	return nil
}

func (s Service) ensureWorkspaceReady(ctx context.Context, environment model.Environment) error {
	if _, err := s.workspaceForEnvironment(ctx, environment); err != nil {
		return apperror.New(apperror.KindValidation, err.Error())
	}
	return nil
}

func (s Service) ListPipelineRunArtifacts(ctx context.Context, userId string, projectId string, runId string) ([]model.Artifact, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListArtifactsByRun(ctx, projectId, run.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list artifacts", err)
	}
	return items, nil
}

func (s Service) CancelPipelineRun(ctx context.Context, userId string, projectId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if run.Status != status.WorkStatusWaitingToRun && run.Status != status.WorkStatusRunning {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot cancel run with status "+run.Status)
	}
	canceled, err := s.store.CancelPipelineRun(ctx, projectId, run.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel pipeline run", err)
	}
	if !canceled {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot cancel run with status "+run.Status)
	}
	updated, err := s.store.PipelineRun(ctx, projectId, run.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, projectId, updated, true)
}

func (s Service) PipelineStageLog(ctx context.Context, userId string, projectId string, runId string, stageRunId string, offset int) (pipelinerundto.PipelineStageLog, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, err
	}
	if offset < 0 {
		return pipelinerundto.PipelineStageLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	stageRun, err := s.store.PipelineStageRun(ctx, projectId, stageRunId)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && stageRun.PipelineRunId != run.Id) {
		return pipelinerundto.PipelineStageLog{Offset: offset, IsComplete: true}, nil
	}
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage run", err)
	}
	runtime, err := s.runtimeForRun(ctx, projectId, run)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to resolve pipeline workspace", err)
	}
	content, next, err := runtime.reader.Read(runtime.workspace.StageLogPath(run.Id, stageRun.Id), offset)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to read stage log", err)
	}
	return pipelinerundto.PipelineStageLog{Logs: string(content), Offset: next, IsComplete: status.WorkStatusIsComplete(stageRun.Status)}, nil
}

func (s Service) loadPipelineRunForUser(ctx context.Context, userId string, projectId string, runId string) (model.PipelineRun, error) {
	if projectId == "" {
		return model.PipelineRun{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.PipelineRun{}, err
	}
	run, err := s.store.PipelineRun(ctx, projectId, runId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.PipelineRun{}, apperror.New(apperror.KindNotFound, "Pipeline run "+runId+" not found")
	}
	if err != nil {
		return model.PipelineRun{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return run, nil
}

func (s Service) warnPipelineRunFileCleanupSkipped(runId string, reason string) {
	if s.logger != nil {
		s.logger.Warn("skipped pipeline run file cleanup", "run_id", runId, "reason", reason)
	}
}

func (s Service) pipelineForUser(ctx context.Context, userId string, projectId string, pipelineId string) (model.Pipeline, error) {
	if projectId == "" {
		return model.Pipeline{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Pipeline{}, err
	}
	pipeline, err := s.store.Pipeline(ctx, projectId, pipelineId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Pipeline{}, apperror.New(apperror.KindNotFound, "Pipeline "+pipelineId+" not found")
	}
	if err != nil {
		return model.Pipeline{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline", err)
	}
	return pipeline, nil
}

func (s Service) repositoryForPipeline(ctx context.Context, projectId string, pipeline model.Pipeline) (model.Repository, error) {
	if pipeline.RepositoryId == nil {
		return model.Repository{}, apperror.New(apperror.KindValidation, "application pipeline has no repository")
	}
	repo, err := s.store.Repository(ctx, projectId, *pipeline.RepositoryId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Repository{}, apperror.New(apperror.KindNotFound, "Pipeline repository not found")
	}
	if err != nil {
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline repository", err)
	}
	return repo, nil
}

func (s Service) resolvePipelineRunVersionBinding(ctx context.Context, projectId string, snapshot model.PipelineSnapshot) (*model.PipelineRunVersionBinding, error) {
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
	app, err := s.store.Application(ctx, projectId, *snapshot.ApplicationId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(apperror.KindNotFound, "Pipeline application not found")
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline application", err)
	}
	var source model.Version
	switch strategy := snapshot.VersionForkStrategy; {
	case strategy == nil:
		return nil, apperror.New(apperror.KindValidation, "component-bound artifacts require a version strategy")
	case *strategy == model.VersionForkStrategyLatest:
		source, err = s.store.LatestVersionByApplication(ctx, projectId, *snapshot.ApplicationId)
	case *strategy == model.VersionForkStrategyFixed && snapshot.FixedVersionId != nil:
		source, err = s.store.Version(ctx, projectId, *snapshot.FixedVersionId)
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
	components, err := s.store.VersionComponentsByVersion(ctx, projectId, source.Id)
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

func (s Service) ensureRepositoryHasNoRunningPipelineRun(ctx context.Context, projectId, repositoryId string) error {
	running, err := s.store.RepositoryHasActivePipelineRun(ctx, projectId, repositoryId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check running pipeline runs", err)
	}
	if running {
		return apperror.New(apperror.KindConflict, "Repository already has an active pipeline run")
	}
	return nil
}

func pipelineStageRuns(runId, stagesSnapshot string) ([]model.PipelineStageRun, error) {
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(stagesSnapshot), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	runs := make([]model.PipelineStageRun, 0, len(stages))
	for _, stage := range stages {
		runs = append(runs, model.PipelineStageRun{
			Id:            idutil.NewId(),
			PipelineRunId: runId,
			StageId:       stage.Id,
			StageName:     stage.Name,
			Status:        status.WorkStatusWaitingToRun,
		})
	}
	return runs, nil
}
func (s Service) ensureRunRepositoryFilter(ctx context.Context, projectId, repositoryId string) error {
	if repositoryId == "" {
		return nil
	}
	_, err := s.store.Repository(ctx, projectId, repositoryId)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return nil
}
func (s Service) ensureRunPipelineFilter(ctx context.Context, projectId, pipelineId string) error {
	if pipelineId == "" {
		return nil
	}
	_, err := s.store.Pipeline(ctx, projectId, pipelineId)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(apperror.KindNotFound, "Pipeline "+pipelineId+" not found")
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline", err)
	}
	return nil
}
func (s Service) pipelineRunDetails(ctx context.Context, projectId string, page repository.Page[model.PipelineRun], includeStages bool) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	items := make([]pipelinerundto.PipelineRunDetail, 0, len(page.Items))
	for _, run := range page.Items {
		detail, err := s.pipelineRunDetail(ctx, projectId, run, includeStages)
		if err != nil {
			return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[pipelinerundto.PipelineRunDetail]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}
func (s Service) pipelineRunDetail(ctx context.Context, projectId string, run model.PipelineRun, includeStages bool) (pipelinerundto.PipelineRunDetail, error) {
	variables, _, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(run.VariablesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	snapshot, err := s.store.PipelineSnapshot(ctx, projectId, run.SnapshotId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	var snapshotStages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &snapshotStages); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	variableViews, err := variableview.Snapshot(snapshotStages, variables)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	detail := pipelinerundto.PipelineRunDetail{Run: run, Variables: variableViews}
	if includeStages {
		stages, err := s.store.ListPipelineStageRuns(ctx, projectId, run.Id)
		if err != nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage runs", err)
		}
		detail.PipelineStageRuns = stages
	}
	binding, err := s.store.PipelineRunVersionBinding(ctx, projectId, run.Id)
	if err == nil {
		detail.VersionBinding = &binding
	} else if !errors.Is(err, repository.ErrNotFound) {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load run version binding", err)
	}
	return detail, nil
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
func buildPipelineRunVariables(repo model.Repository, pipeline model.Pipeline, snapshot model.PipelineSnapshot, overrides pipelinevariable.RuntimeVariableOverrides) (string, string, error) {
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return "", "", apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	declarations, variables, err := pipelinevariable.ResolveRuntimeVariables(repo, pipeline, stages, overrides)
	if err != nil {
		return "", "", err
	}
	data, err := pipelinevariable.MarshalRuntimeVariableSnapshot(variables, declarations)
	if err != nil {
		return "", "", err
	}
	ref := fmt.Sprint(variables.Global["repository_ref"])
	return data, ref, nil
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
