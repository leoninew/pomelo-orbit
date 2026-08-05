package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pipelinesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/usecase"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListRepositoryRuns(ctx context.Context, userId string, repositoryId string, page int, perPage int) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRunsByRepository(ctx, repo.Id, page, perPage)
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list repository runs", err)
	}
	return s.pipelineRunDetails(ctx, items, false)
}

func (s Service) TriggerRepository(ctx context.Context, userId string, input pipelinerundto.PipelineRunTriggerInput) (pipelinerundto.PipelineRunDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, input.RepositoryId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	templateId := strings.TrimSpace(input.TemplateId)
	if templateId == "" {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "template_id is required")
	}
	template, err := s.store.PipelineTemplate(ctx, templateId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	snapshot, err := pipelinesvc.GetOrCreatePipelineSnapshot(ctx, s.store, template)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if err := s.ensureRepositoryHasNoRunningPipelineRun(ctx, repo.Id); err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	buildBindings, err := s.resolvePipelineRunBuildVersionBindings(ctx, repo.ProjectId, snapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	triggerRef := strings.TrimSpace(input.TriggerRef)
	if triggerRef == "" {
		triggerRef = repo.DefaultBranch
	}
	triggerRef, err = s.pipelineRunRef(ctx, repo, triggerRef)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	variablesSnapshot, err := buildPipelineRunVariables(repo, template, snapshot, triggerRef, input.Variables)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	run := model.PipelineRun{Id: idutil.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "manual", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: status.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRunWithBuildVersionBindings(ctx, run, buildBindings); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if err := s.dispatcher.DispatchPipelineRun(ctx, pipelinerundto.PipelineRunDispatchInput{PipelineRunId: run.Id}); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	created, err := s.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, created, false)
}

func (s Service) ListPipelineRuns(ctx context.Context, userId string, input pipelinerundto.PipelineRunListInput) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	dateFrom, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	dateTo, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	if err := s.ensurePipelineRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	if err := s.ensurePipelineRunTemplateFilter(ctx, projectId, input.TemplateId); err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRuns(ctx, projectId, input.RepositoryId, input.TemplateId, dateFrom, dateTo, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[pipelinerundto.PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline runs", err)
	}
	return s.pipelineRunDetails(ctx, items, false)
}

func (s Service) PipelineRunForUser(ctx context.Context, userId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	return s.pipelineRunDetail(ctx, run, true)
}

func (s Service) ListPipelineRunArtifacts(ctx context.Context, userId string, runId string) ([]model.Artifact, error) {
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

func (s Service) PipelineStageLog(ctx context.Context, userId string, runId string, pipelineStageRunId string, offset int) (pipelinerundto.PipelineStageLog, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, err
	}
	if offset < 0 {
		return pipelinerundto.PipelineStageLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	pipelineStageRun, err := s.store.PipelineStageRun(ctx, strings.TrimSpace(pipelineStageRunId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinerundto.PipelineStageLog{Offset: offset, IsComplete: true}, nil
		}
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage run", err)
	}
	if pipelineStageRun.PipelineRunId != run.Id {
		return pipelinerundto.PipelineStageLog{Offset: offset, IsComplete: true}, nil
	}
	logPath := s.workspace.StageLogPath(run.Id, pipelineStageRun.Id)
	content, newOffset, err := s.logStore.Read(logPath, offset)
	if err != nil {
		return pipelinerundto.PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to read stage log", err)
	}
	return pipelinerundto.PipelineStageLog{Logs: string(content), Offset: newOffset, IsComplete: pipelineRunStatusComplete(pipelineStageRun.Status)}, nil
}

func (s Service) CancelPipelineRun(ctx context.Context, userId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
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

func (s Service) RetryPipelineRun(ctx context.Context, userId string, runId string) (pipelinerundto.PipelineRunDetail, error) {
	original, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if original.Status != status.WorkStatusFaulted && original.Status != status.WorkStatusRanToCompletion {
		return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot retry run with status "+original.Status)
	}
	snapshot, err := s.store.PipelineSnapshot(ctx, original.SnapshotId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Snapshot "+original.SnapshotId+" not found")
		}
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	repo, err := s.store.Repository(ctx, original.RepositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinerundto.PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Repository "+original.RepositoryId+" not found")
		}
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	template := model.PipelineTemplate{Id: original.TemplateId, ProjectId: original.ProjectId, Name: original.TemplateName, VariableDeclarations: snapshot.VariablesSnapshot, Version: original.TemplateVersion}
	variablesSnapshot, err := buildPipelineRunVariables(repo, template, snapshot, original.TriggerRef, nil)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	if err := s.ensureRepositoryHasNoRunningPipelineRun(ctx, repo.Id); err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	buildBindings, err := s.resolvePipelineRunBuildVersionBindings(ctx, repo.ProjectId, snapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	newRun := model.PipelineRun{Id: idutil.NewId(), ProjectId: original.ProjectId, RepositoryId: original.RepositoryId, RepositoryName: repo.Name, SnapshotId: original.SnapshotId, TemplateId: original.TemplateId, TemplateName: original.TemplateName, TemplateVersion: original.TemplateVersion, Trigger: original.Trigger, TriggerRef: original.TriggerRef, VariablesSnapshot: variablesSnapshot, Status: status.WorkStatusWaitingToRun, RetryOf: &original.Id}
	if err := s.store.CreatePipelineRunWithBuildVersionBindings(ctx, newRun, buildBindings); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create retry pipeline run", err)
	}
	if err := s.dispatcher.DispatchPipelineRun(ctx, pipelinerundto.PipelineRunDispatchInput{PipelineRunId: newRun.Id}); err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	created, err := s.store.PipelineRun(ctx, newRun.Id)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, created, true)
}

func (s Service) loadPipelineRunForUser(ctx context.Context, userId string, runId string) (model.PipelineRun, error) {
	runId = strings.TrimSpace(runId)
	run, err := s.store.PipelineRun(ctx, runId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.PipelineRun{}, apperror.New(apperror.KindNotFound, "PipelineRun "+runId+" not found")
		}
		return model.PipelineRun{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	if run.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *run.ProjectId, userId); err != nil {
			return model.PipelineRun{}, err
		}
	}
	return run, nil
}

func (s Service) pipelineRunDetails(ctx context.Context, page repository.Page[model.PipelineRun], includeStages bool) (repository.Page[pipelinerundto.PipelineRunDetail], error) {
	items := make([]pipelinerundto.PipelineRunDetail, 0, len(page.Items))
	for _, item := range page.Items {
		detail, err := s.pipelineRunDetail(ctx, item, includeStages)
		if err != nil {
			return repository.Page[pipelinerundto.PipelineRunDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[pipelinerundto.PipelineRunDetail]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}

func (s Service) pipelineRunDetail(ctx context.Context, item model.PipelineRun, includeStages bool) (pipelinerundto.PipelineRunDetail, error) {
	variables, err := pipelineRunVariables(item.VariablesSnapshot)
	if err != nil {
		return pipelinerundto.PipelineRunDetail{}, err
	}
	pipelineStageRuns := []model.PipelineStageRun{}
	if includeStages {
		items, err := s.store.ListPipelineStageRuns(ctx, item.Id)
		if err != nil {
			return pipelinerundto.PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline stage runs", err)
		}
		pipelineStageRuns = items
	}
	return pipelinerundto.PipelineRunDetail{Run: item, VariablesSnapshot: variables, PipelineStageRuns: pipelineStageRuns}, nil
}

func (s Service) pipelineRunRef(ctx context.Context, repo model.Repository, ref string) (string, error) {
	repositoryType := repo.RepositoryType
	if repositoryType == "" {
		repositoryType = model.RepositoryTypeRemoteGit
	}
	switch repositoryType {
	case model.RepositoryTypeRemoteGit:
		if strings.TrimSpace(repo.RepositoryUrl) == "" {
			return "", apperror.New(apperror.KindValidation, "repository_url is required for remote Git repositories")
		}
		return ref, nil
	case model.RepositoryTypeLocalDirectory:
		if s.localSource == nil {
			return "", apperror.New(apperror.KindValidation, "Local directory sources are disabled")
		}
		revision, err := s.localSource.ResolveRevision(ctx, repo.RepositoryUrl, ref)
		if err != nil {
			return "", apperror.New(apperror.KindValidation, "Unable to resolve local source revision: "+err.Error())
		}
		return revision, nil
	default:
		return "", apperror.New(apperror.KindValidation, "Unsupported repository_type")
	}
}

func (s Service) ensurePipelineRunRepositoryFilter(ctx context.Context, projectId string, repositoryId string) error {
	repositoryId = strings.TrimSpace(repositoryId)
	if repositoryId == "" {
		return nil
	}
	repo, err := s.store.Repository(ctx, repositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId == nil || *repo.ProjectId != projectId {
		return apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
	}
	return nil
}

func (s Service) ensurePipelineRunTemplateFilter(ctx context.Context, projectId string, templateId string) error {
	templateId = strings.TrimSpace(templateId)
	if templateId == "" {
		return nil
	}
	template, err := s.store.PipelineTemplate(ctx, templateId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	if template.ProjectId == nil || *template.ProjectId != projectId {
		return apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
	}
	return nil
}

func parseOptionalRunTime(value string, name string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
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
