package cisvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	"backend/internal/status"
)

type PipelineRunTriggerInput struct {
	RepositoryId string
	TemplateId   string
	TriggerRef   string
	Variables    map[string]string
}

type PipelineRunListInput struct {
	ProjectId    string
	RepositoryId string
	TemplateId   string
	DateFrom     string
	DateTo       string
	Page         int
	PerPage      int
}

type PipelineRunDetail struct {
	Run               model.PipelineRun
	VariablesSnapshot []model.VariableDeclaration
	StageRuns         []model.StageRun
}

type PipelineStageLog struct {
	Logs       string
	Offset     int
	IsComplete bool
}

func (s Service) ListRepositoryRuns(ctx context.Context, userId string, repositoryId string, page int, perPage int) (repository.Page[PipelineRunDetail], error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRunsByRepository(ctx, repo.Id, page, perPage)
	if err != nil {
		return repository.Page[PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list repository runs", err)
	}
	return s.pipelineRunDetails(ctx, items, false)
}

func (s Service) TriggerRepository(ctx context.Context, userId string, input PipelineRunTriggerInput) (PipelineRunDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, input.RepositoryId)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	templateId := strings.TrimSpace(input.TemplateId)
	if templateId == "" {
		return PipelineRunDetail{}, apperror.New(apperror.KindValidation, "template_id is required")
	}
	template, err := s.store.PipelineTemplate(ctx, templateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	snapshot, err := s.getOrCreatePipelineSnapshot(ctx, template)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	triggerRef := strings.TrimSpace(input.TriggerRef)
	if triggerRef == "" {
		triggerRef = repo.DefaultBranch
	}
	variablesSnapshot, err := buildPipelineRunVariables(repo, template, snapshot, triggerRef, input.Variables)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	run := model.PipelineRun{Id: repository.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "manual", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: status.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRun(ctx, run); err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": run.Id}); err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	created, err := s.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, created, false)
}

func (s Service) ListPipelineRuns(ctx context.Context, userId string, input PipelineRunListInput) (repository.Page[PipelineRunDetail], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[PipelineRunDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	dateFrom, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	dateTo, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	if err := s.ensurePipelineRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	if err := s.ensurePipelineRunTemplateFilter(ctx, projectId, input.TemplateId); err != nil {
		return repository.Page[PipelineRunDetail]{}, err
	}
	items, err := s.store.ListPipelineRuns(ctx, projectId, input.RepositoryId, input.TemplateId, dateFrom, dateTo, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[PipelineRunDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline runs", err)
	}
	return s.pipelineRunDetails(ctx, items, false)
}

func (s Service) PipelineRunForUser(ctx context.Context, userId string, runId string) (PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return PipelineRunDetail{}, err
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

func (s Service) PipelineStageLog(ctx context.Context, userId string, runId string, stageRunId string, offset int) (PipelineStageLog, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return PipelineStageLog{}, err
	}
	if offset < 0 {
		return PipelineStageLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	stageRun, err := s.store.StageRun(ctx, strings.TrimSpace(stageRunId))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineStageLog{Offset: offset, IsComplete: true}, nil
		}
		return PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load stage run", err)
	}
	if stageRun.PipelineRunId != run.Id {
		return PipelineStageLog{Offset: offset, IsComplete: true}, nil
	}
	logPath := s.workspace.StageLogPath(run.Id, stageRun.Id)
	content, newOffset, err := s.logStore.Read(logPath, offset)
	if err != nil {
		return PipelineStageLog{}, apperror.Wrap(apperror.KindInternal, "Failed to read stage log", err)
	}
	return PipelineStageLog{Logs: string(content), Offset: newOffset, IsComplete: pipelineRunStatusComplete(stageRun.Status)}, nil
}

func (s Service) CancelPipelineRun(ctx context.Context, userId string, runId string) (PipelineRunDetail, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	if run.Status != status.WorkStatusWaitingToRun && run.Status != status.WorkStatusRunning {
		return PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot cancel run with status "+run.Status)
	}
	if err := s.store.CancelPipelineRun(ctx, run.Id); err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel pipeline run", err)
	}
	updated, err := s.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, updated, true)
}

func (s Service) RetryPipelineRun(ctx context.Context, userId string, runId string) (PipelineRunDetail, error) {
	original, err := s.loadPipelineRunForUser(ctx, userId, runId)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	if original.Status != status.WorkStatusFaulted && original.Status != status.WorkStatusRanToCompletion {
		return PipelineRunDetail{}, apperror.New(apperror.KindValidation, "Cannot retry run with status "+original.Status)
	}
	snapshot, err := s.store.PipelineSnapshot(ctx, original.SnapshotId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Snapshot "+original.SnapshotId+" not found")
		}
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	repo, err := s.store.Repository(ctx, original.RepositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineRunDetail{}, apperror.New(apperror.KindNotFound, "Repository "+original.RepositoryId+" not found")
		}
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	template := model.PipelineTemplate{Id: original.TemplateId, ProjectId: original.ProjectId, Name: original.TemplateName, VariableDeclarations: snapshot.VariablesSnapshot, Version: original.TemplateVersion}
	variablesSnapshot, err := buildPipelineRunVariables(repo, template, snapshot, original.TriggerRef, nil)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	newRun := model.PipelineRun{Id: repository.NewId(), ProjectId: original.ProjectId, RepositoryId: original.RepositoryId, RepositoryName: repo.Name, SnapshotId: original.SnapshotId, TemplateId: original.TemplateId, TemplateName: original.TemplateName, TemplateVersion: original.TemplateVersion, Trigger: original.Trigger, TriggerRef: original.TriggerRef, VariablesSnapshot: variablesSnapshot, Status: status.WorkStatusWaitingToRun, RetryOf: &original.Id}
	if err := s.store.CreatePipelineRun(ctx, newRun); err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create retry pipeline run", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": newRun.Id}); err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	created, err := s.store.PipelineRun(ctx, newRun.Id)
	if err != nil {
		return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline run", err)
	}
	return s.pipelineRunDetail(ctx, created, true)
}

func (s Service) loadPipelineRunForUser(ctx context.Context, userId string, runId string) (model.PipelineRun, error) {
	runId = strings.TrimSpace(runId)
	run, err := s.store.PipelineRun(ctx, runId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

func (s Service) pipelineRunDetails(ctx context.Context, page repository.Page[model.PipelineRun], includeStages bool) (repository.Page[PipelineRunDetail], error) {
	items := make([]PipelineRunDetail, 0, len(page.Items))
	for _, item := range page.Items {
		detail, err := s.pipelineRunDetail(ctx, item, includeStages)
		if err != nil {
			return repository.Page[PipelineRunDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[PipelineRunDetail]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}, nil
}

func (s Service) pipelineRunDetail(ctx context.Context, item model.PipelineRun, includeStages bool) (PipelineRunDetail, error) {
	variables, err := pipelineRunVariables(item.VariablesSnapshot)
	if err != nil {
		return PipelineRunDetail{}, err
	}
	stageRuns := []model.StageRun{}
	if includeStages {
		items, err := s.store.ListStageRuns(ctx, item.Id)
		if err != nil {
			return PipelineRunDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to list stage runs", err)
		}
		stageRuns = items
	}
	return PipelineRunDetail{Run: item, VariablesSnapshot: variables, StageRuns: stageRuns}, nil
}

func (s Service) ensurePipelineRunRepositoryFilter(ctx context.Context, projectId string, repositoryId string) error {
	repositoryId = strings.TrimSpace(repositoryId)
	if repositoryId == "" {
		return nil
	}
	repo, err := s.store.Repository(ctx, repositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
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
	if err := json.Unmarshal([]byte(value), &variables); err == nil {
		return variables, nil
	}
	var legacy map[string]any
	if err := json.Unmarshal([]byte(value), &legacy); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline run variables")
	}
	variables = make([]model.VariableDeclaration, 0, len(legacy))
	for name, value := range legacy {
		variables = append(variables, model.VariableDeclaration{Name: name, Value: value, Source: "runtime", Editable: true})
	}
	return variables, nil
}
