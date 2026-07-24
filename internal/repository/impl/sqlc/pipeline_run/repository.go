package pipelinerunrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	pipelinerunsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/pipeline_run"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.PipelineRunStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *pipelinerunsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *pipelinerunsqlc.Queries {
		return pipelinerunsqlc.New(dbtx)
	})
}

func optionalFilter(value string) (raw string, id string) {
	raw = strings.TrimSpace(value)
	return raw, raw
}

func nullTimeArg(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func (r Repository) ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	repoRaw, repoID := optionalFilter(repositoryId)
	tplRaw, tplID := optionalFilter(templateId)
	params := pipelinerunsqlc.CountPipelineRunsParams{
		ProjectID:        sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		RepositoryFilter: repoRaw,
		RepositoryID:     repoID,
		TemplateFilter:   tplRaw,
		TemplateID:       tplID,
		DateFrom:         nullTimeArg(dateFrom),
		DateTo:           nullTimeArg(dateTo),
	}
	q := r.q(ctx)
	total, err := q.CountPipelineRuns(ctx, params)
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("count pipeline runs: %w", err)
	}
	rows, err := q.ListPipelineRuns(ctx, pipelinerunsqlc.ListPipelineRunsParams{
		ProjectID:        params.ProjectID,
		RepositoryFilter: params.RepositoryFilter,
		RepositoryID:     params.RepositoryID,
		TemplateFilter:   params.TemplateFilter,
		TemplateID:       params.TemplateID,
		DateFrom:         params.DateFrom,
		DateTo:           params.DateTo,
		Offset:           int64((page - 1) * perPage),
		Limit:            int64(perPage),
	})
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("list pipeline runs: %w", err)
	}
	items := make([]model.PipelineRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineRunFrom(row.ID, row.ProjectID, row.RepositoryID, row.RepositoryName, row.SnapshotID, row.TemplateID, row.TemplateName, row.TemplateVersion, row.Trigger, row.TriggerRef, row.VariablesSnapshot, row.Status, row.RetryOf, row.StartedAt, row.FinishedAt, row.ErrorMessage, row.CreatedAt))
	}
	return repository.Page[model.PipelineRun]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	q := r.q(ctx)
	total, err := q.CountPipelineRunsByRepository(ctx, repositoryId)
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("count repository pipeline runs %s: %w", repositoryId, err)
	}
	rows, err := q.ListPipelineRunsByRepository(ctx, pipelinerunsqlc.ListPipelineRunsByRepositoryParams{
		RepositoryID: repositoryId, Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.PipelineRun]{}, fmt.Errorf("list repository pipeline runs %s: %w", repositoryId, err)
	}
	items := make([]model.PipelineRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineRunFrom(row.ID, row.ProjectID, row.RepositoryID, row.RepositoryName, row.SnapshotID, row.TemplateID, row.TemplateName, row.TemplateVersion, row.Trigger, row.TriggerRef, row.VariablesSnapshot, row.Status, row.RetryOf, row.StartedAt, row.FinishedAt, row.ErrorMessage, row.CreatedAt))
	}
	return repository.Page[model.PipelineRun]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	row, err := r.q(ctx).PipelineRunByID(ctx, id)
	if err != nil {
		return model.PipelineRun{}, fmt.Errorf("load pipeline run %s: %w", id, sqlcommon.TranslateError(err))
	}
	return pipelineRunFrom(row.ID, row.ProjectID, row.RepositoryID, row.RepositoryName, row.SnapshotID, row.TemplateID, row.TemplateName, row.TemplateVersion, row.Trigger, row.TriggerRef, row.VariablesSnapshot, row.Status, row.RetryOf, row.StartedAt, row.FinishedAt, row.ErrorMessage, row.CreatedAt), nil
}

func (r Repository) CreatePipelineRun(ctx context.Context, run model.PipelineRun) error {
	err := r.q(ctx).CreatePipelineRun(ctx, pipelinerunsqlc.CreatePipelineRunParams{
		ID: run.Id, ProjectID: dbmodel.NullString(run.ProjectId), RepositoryID: run.RepositoryId, RepositoryName: run.RepositoryName,
		SnapshotID: run.SnapshotId, TemplateID: run.TemplateId, TemplateName: run.TemplateName, TemplateVersion: int64(run.TemplateVersion),
		Trigger: run.Trigger, TriggerRef: run.TriggerRef, VariablesSnapshot: run.VariablesSnapshot, Status: run.Status,
		RetryOf: dbmodel.NullString(run.RetryOf), CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("create pipeline run %s: %w", run.Id, err)
	}
	return nil
}

func (r Repository) CancelPipelineRun(ctx context.Context, id string) error {
	now := time.Now().UTC()
	err := r.q(ctx).CancelPipelineRun(ctx, pipelinerunsqlc.CancelPipelineRunParams{
		Status: status.WorkStatusCanceled, FinishedAt: sql.NullTime{Time: now, Valid: true}, ID: id,
	})
	if err != nil {
		return fmt.Errorf("cancel pipeline run %s: %w", id, err)
	}
	return nil
}

func (r Repository) MarkPipelineRunRunning(ctx context.Context, id string) error {
	now := time.Now().UTC()
	err := r.q(ctx).MarkPipelineRunRunning(ctx, pipelinerunsqlc.MarkPipelineRunRunningParams{
		Status: status.WorkStatusRunning, StartedAt: sql.NullTime{Time: now, Valid: true}, ID: id,
	})
	if err != nil {
		return fmt.Errorf("mark pipeline run running %s: %w", id, err)
	}
	return nil
}

func (r Repository) CompletePipelineRun(ctx context.Context, id string, runStatus string, message string) error {
	now := time.Now().UTC()
	err := r.q(ctx).CompletePipelineRun(ctx, pipelinerunsqlc.CompletePipelineRunParams{
		Status: runStatus, FinishedAt: sql.NullTime{Time: now, Valid: true}, NULLIF: message, ID: id,
	})
	if err != nil {
		return fmt.Errorf("complete pipeline run %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListPipelineStageRuns(ctx context.Context, runId string) ([]model.PipelineStageRun, error) {
	rows, err := r.q(ctx).ListPipelineStageRuns(ctx, runId)
	if err != nil {
		return nil, fmt.Errorf("list pipeline stage runs %s: %w", runId, err)
	}
	items := make([]model.PipelineStageRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineStageRunFrom(row.ID, row.PipelineRunID, row.StageID, row.StageName, row.Status, row.StartedAt, row.FinishedAt, row.ExitCode, row.ErrorMessage))
	}
	return items, nil
}

func (r Repository) PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error) {
	row, err := r.q(ctx).PipelineStageRunByID(ctx, id)
	if err != nil {
		return model.PipelineStageRun{}, fmt.Errorf("load pipeline stage run %s: %w", id, sqlcommon.TranslateError(err))
	}
	return pipelineStageRunFrom(row.ID, row.PipelineRunID, row.StageID, row.StageName, row.Status, row.StartedAt, row.FinishedAt, row.ExitCode, row.ErrorMessage), nil
}

func (r Repository) InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	err := r.q(ctx).InsertPipelineStageRun(ctx, pipelinerunsqlc.InsertPipelineStageRunParams{
		ID: stage.Id, PipelineRunID: stage.PipelineRunId, StageID: stage.StageId, StageName: stage.StageName,
		Status: stage.Status, StartedAt: dbmodel.NullTime(stage.StartedAt), FinishedAt: dbmodel.NullTime(stage.FinishedAt),
		ExitCode: dbmodel.NullInt64FromIntPtr(stage.ExitCode), ErrorMessage: dbmodel.NullString(stage.ErrorMessage),
	})
	if err != nil {
		return fmt.Errorf("insert pipeline stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	err := r.q(ctx).UpdatePipelineStageRun(ctx, pipelinerunsqlc.UpdatePipelineStageRunParams{
		Status: stage.Status, StartedAt: dbmodel.NullTime(stage.StartedAt), FinishedAt: dbmodel.NullTime(stage.FinishedAt),
		ExitCode: dbmodel.NullInt64FromIntPtr(stage.ExitCode), ErrorMessage: dbmodel.NullString(stage.ErrorMessage), ID: stage.Id,
	})
	if err != nil {
		return fmt.Errorf("update pipeline stage run %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (repository.Page[model.Artifact], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	repoRaw, repoID := optionalFilter(repositoryId)
	tplRaw, tplID := optionalFilter(templateId)
	searchRaw, pattern := dbmodel.SearchPattern(search)
	q := r.q(ctx)
	total, err := q.CountArtifacts(ctx, pipelinerunsqlc.CountArtifactsParams{
		ProjectID: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:   repoRaw, RepositoryID: repoID, Column4: tplRaw, TemplateID: tplID,
		Column6: searchRaw, Name: pattern, Path: sql.NullString{String: pattern, Valid: pattern != ""},
	})
	if err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("count artifacts: %w", err)
	}
	rows, err := q.ListArtifacts(ctx, pipelinerunsqlc.ListArtifactsParams{
		ProjectID: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:   repoRaw, RepositoryID: repoID, Column4: tplRaw, TemplateID: tplID,
		Column6: searchRaw, Name: pattern, Path: sql.NullString{String: pattern, Valid: pattern != ""},
		Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("list artifacts: %w", err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, artifactFrom(row.ID, row.ProjectID, row.PipelineRunID, row.RepositoryID, row.RepositoryName, row.TemplateID, row.TemplateName, row.StageName, row.Type, row.Name, row.Path, row.CreatedAt))
	}
	return repository.Page[model.Artifact]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error) {
	var projectNS sql.NullString
	if projectId != nil && strings.TrimSpace(*projectId) != "" {
		projectNS = sql.NullString{String: strings.TrimSpace(*projectId), Valid: true}
	}
	rows, err := r.q(ctx).ListArtifactsByRun(ctx, pipelinerunsqlc.ListArtifactsByRunParams{
		PipelineRunID: runId, ProjectID: projectNS,
	})
	if err != nil {
		return nil, fmt.Errorf("list run artifacts %s: %w", runId, err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, artifactFrom(row.ID, row.ProjectID, row.PipelineRunID, row.RepositoryID, row.RepositoryName, row.TemplateID, row.TemplateName, row.StageName, row.Type, row.Name, row.Path, row.CreatedAt))
	}
	return items, nil
}

func (r Repository) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	err := r.q(ctx).InsertArtifact(ctx, pipelinerunsqlc.InsertArtifactParams{
		ID: idutil.NewId(), ProjectID: dbmodel.NullString(projectId), PipelineRunID: run.Id, RepositoryID: run.RepositoryId,
		RepositoryName: run.RepositoryName, TemplateID: run.TemplateId, TemplateName: run.TemplateName, StageName: stageName,
		Type: artifact.Type, Name: artifact.Name, Path: sql.NullString{String: path, Valid: path != ""}, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("insert artifact %s: %w", artifact.Name, err)
	}
	return nil
}

func pipelineRunFrom(id string, projectID sql.NullString, repositoryID, repositoryName, snapshotID, templateID, templateName string, templateVersion int64, trigger, triggerRef, variablesSnapshot, runStatus string, retryOf sql.NullString, startedAt, finishedAt sql.NullTime, errorMessage sql.NullString, createdAt time.Time) model.PipelineRun {
	return model.PipelineRun{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), RepositoryId: repositoryID, RepositoryName: repositoryName,
		SnapshotId: snapshotID, TemplateId: templateID, TemplateName: templateName, TemplateVersion: int(templateVersion),
		Trigger: trigger, TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: runStatus,
		RetryOf: dbmodel.StringPtr(retryOf), StartedAt: dbmodel.TimePtr(startedAt), FinishedAt: dbmodel.TimePtr(finishedAt),
		ErrorMessage: dbmodel.StringPtr(errorMessage), CreatedAt: createdAt,
	}
}

func pipelineStageRunFrom(id, pipelineRunID, stageID, stageName, runStatus string, startedAt, finishedAt sql.NullTime, exitCode sql.NullInt64, errorMessage sql.NullString) model.PipelineStageRun {
	return model.PipelineStageRun{
		Id: id, PipelineRunId: pipelineRunID, StageId: stageID, StageName: stageName, Status: runStatus,
		StartedAt: dbmodel.TimePtr(startedAt), FinishedAt: dbmodel.TimePtr(finishedAt),
		ExitCode: dbmodel.IntPtrFromNullInt64(exitCode), ErrorMessage: dbmodel.StringPtr(errorMessage),
	}
}

func artifactFrom(id string, projectID sql.NullString, pipelineRunID, repositoryID, repositoryName, templateID, templateName, stageName, typ, name string, path sql.NullString, createdAt time.Time) model.Artifact {
	return model.Artifact{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), PipelineRunId: pipelineRunID, RepositoryId: repositoryID,
		RepositoryName: repositoryName, TemplateId: templateID, TemplateName: templateName, StageName: stageName,
		Type: typ, Name: name, Path: dbmodel.StringPtr(path), CreatedAt: createdAt,
	}
}
