package pipelinerunrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
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
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *pipelinerunsqlc.Queries {
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
	repoRaw, repoId := optionalFilter(repositoryId)
	tplRaw, tplId := optionalFilter(templateId)
	params := pipelinerunsqlc.CountPipelineRunsParams{
		ProjectID:        sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		RepositoryFilter: repoRaw,
		RepositoryID:     repoId,
		TemplateFilter:   tplRaw,
		TemplateID:       tplId,
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
	if err := r.createPipelineRun(ctx, run); err != nil {
		return err
	}
	return nil
}

func (r Repository) createPipelineRun(ctx context.Context, run model.PipelineRun) error {
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

func (r Repository) CreatePipelineRunWithBuildVersionBindings(ctx context.Context, run model.PipelineRun, bindings []model.PipelineRunBuildVersionBinding) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.createPipelineRun(txCtx, run); err != nil {
			return err
		}
		q := r.q(txCtx)
		for _, binding := range bindings {
			if err := q.InsertPipelineRunBuildVersionBinding(txCtx, pipelinerunsqlc.InsertPipelineRunBuildVersionBindingParams{
				PipelineRunID: run.Id, PipelineStageID: binding.PipelineStageId, ApplicationID: binding.ApplicationId, ApplicationName: binding.ApplicationName,
				ComponentName: binding.ComponentName, SourceVersionID: binding.SourceVersionId, SourceVersionLabel: binding.SourceVersionLabel,
				GeneratedVersionID: dbmodel.NullString(binding.GeneratedVersionId), GeneratedVersionLabel: dbmodel.NullString(binding.GeneratedVersionLabel), ArtifactID: dbmodel.NullString(binding.ArtifactId),
			}); err != nil {
				return fmt.Errorf("create pipeline run build version binding %s: %w", binding.PipelineStageId, err)
			}
		}
		return nil
	})
}

func (r Repository) RepositoryHasRunningPipelineRun(ctx context.Context, repositoryId string) (bool, error) {
	exists, err := r.q(ctx).RepositoryHasRunningPipelineRun(ctx, repositoryId)
	if err != nil {
		return false, fmt.Errorf("check running pipeline runs for repository %s: %w", repositoryId, err)
	}
	return exists != 0, nil
}

func (r Repository) PipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string) (model.PipelineRunBuildVersionBinding, error) {
	row, err := r.q(ctx).PipelineRunBuildVersionBindingByRunAndStage(ctx, pipelinerunsqlc.PipelineRunBuildVersionBindingByRunAndStageParams{PipelineRunID: pipelineRunId, PipelineStageID: pipelineStageId})
	if err != nil {
		return model.PipelineRunBuildVersionBinding{}, fmt.Errorf("load pipeline run build version binding %s/%s: %w", pipelineRunId, pipelineStageId, sqlcommon.TranslateError(err))
	}
	return pipelineRunBuildVersionBindingFrom(row), nil
}

func (r Repository) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return tx.RunInTx(ctx, r.db, fn)
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
	repoRaw, repoId := optionalFilter(repositoryId)
	tplRaw, tplId := optionalFilter(templateId)
	searchRaw, pattern := dbmodel.SearchPattern(search)
	q := r.q(ctx)
	total, err := q.CountArtifacts(ctx, pipelinerunsqlc.CountArtifactsParams{
		ProjectID:        sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		RepositoryFilter: repoRaw, RepositoryID: repoId, TemplateFilter: tplRaw, TemplateID: tplId,
		SearchFilter: searchRaw, SearchPattern: pattern,
	})
	if err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("count artifacts: %w", err)
	}
	rows, err := q.ListArtifacts(ctx, pipelinerunsqlc.ListArtifactsParams{
		ProjectID:        sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		RepositoryFilter: repoRaw, RepositoryID: repoId, TemplateFilter: tplRaw, TemplateID: tplId,
		SearchFilter: searchRaw, SearchPattern: pattern,
		Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Artifact]{}, fmt.Errorf("list artifacts: %w", err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, artifactFrom(row.ID, row.ProjectID, row.PipelineRunID, row.RepositoryID, row.RepositoryName, row.TemplateID, row.TemplateName, row.PipelineStageID, row.StageName, row.Collector, row.Name, row.Location, row.Value, row.ValueFormat, row.ImageRef, row.LocalImageSha256, row.SourceCommitSha, row.ApplicationID, row.ApplicationName, row.SourceVersionID, row.SourceVersionLabel, row.GeneratedVersionID, row.GeneratedVersionLabel, row.ComponentName, row.ComponentID, row.CreatedAt))
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
		items = append(items, artifactFrom(row.ID, row.ProjectID, row.PipelineRunID, row.RepositoryID, row.RepositoryName, row.TemplateID, row.TemplateName, row.PipelineStageID, row.StageName, row.Collector, row.Name, row.Location, row.Value, row.ValueFormat, row.ImageRef, row.LocalImageSha256, row.SourceCommitSha, row.ApplicationID, row.ApplicationName, row.SourceVersionID, row.SourceVersionLabel, row.GeneratedVersionID, row.GeneratedVersionLabel, row.ComponentName, row.ComponentID, row.CreatedAt))
	}
	return items, nil
}

func (r Repository) Artifact(ctx context.Context, artifactId string) (model.Artifact, error) {
	row, err := r.q(ctx).ArtifactByID(ctx, strings.TrimSpace(artifactId))
	if err != nil {
		return model.Artifact{}, fmt.Errorf("load artifact %s: %w", artifactId, sqlcommon.TranslateError(err))
	}
	return artifactFrom(row.ID, row.ProjectID, row.PipelineRunID, row.RepositoryID, row.RepositoryName, row.TemplateID, row.TemplateName, row.PipelineStageID, row.StageName, row.Collector, row.Name, row.Location, row.Value, row.ValueFormat, row.ImageRef, row.LocalImageSha256, row.SourceCommitSha, row.ApplicationID, row.ApplicationName, row.SourceVersionID, row.SourceVersionLabel, row.GeneratedVersionID, row.GeneratedVersionLabel, row.ComponentName, row.ComponentID, row.CreatedAt), nil
}

func (r Repository) CreateArtifact(ctx context.Context, artifact model.Artifact) error {
	err := r.q(ctx).InsertArtifact(ctx, pipelinerunsqlc.InsertArtifactParams{
		ID: artifact.Id, ProjectID: dbmodel.NullString(artifact.ProjectId), PipelineRunID: artifact.PipelineRunId, RepositoryID: artifact.RepositoryId,
		RepositoryName: artifact.RepositoryName, TemplateID: artifact.TemplateId, TemplateName: artifact.TemplateName, PipelineStageID: artifact.PipelineStageId,
		StageName: artifact.StageName, Collector: artifact.Collector, Name: artifact.Name, Location: dbmodel.NullString(artifact.Location),
		Value: dbmodel.NullString(artifact.Value), ValueFormat: dbmodel.NullString(artifact.ValueFormat), ImageRef: dbmodel.NullString(artifact.ImageRef),
		LocalImageSha256: dbmodel.NullString(artifact.LocalImageSha256), SourceArtifactID: dbmodel.NullString(artifact.SourceArtifactId), CreatedAt: artifact.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create artifact %s: %w", artifact.Name, err)
	}
	return nil
}

func (r Repository) CommandArtifactByRunStageAndName(ctx context.Context, pipelineRunId string, pipelineStageId string, name string) (model.Artifact, error) {
	row, err := r.q(ctx).CommandArtifactByRunStageAndName(ctx, pipelinerunsqlc.CommandArtifactByRunStageAndNameParams{
		PipelineRunID: pipelineRunId, PipelineStageID: pipelineStageId, Name: name,
	})
	if err != nil {
		return model.Artifact{}, fmt.Errorf("load command artifact %s/%s/%s: %w", pipelineRunId, pipelineStageId, name, sqlcommon.TranslateError(err))
	}
	return model.Artifact{Id: row.ID, Value: dbmodel.StringPtr(row.Value), ValueFormat: dbmodel.StringPtr(row.ValueFormat)}, nil
}

func (r Repository) CompletePipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string, generatedVersionId string, generatedVersionLabel string, artifactId string) error {
	if err := r.q(ctx).UpdatePipelineRunBuildVersionBindingResult(ctx, pipelinerunsqlc.UpdatePipelineRunBuildVersionBindingResultParams{
		GeneratedVersionID: sql.NullString{String: generatedVersionId, Valid: true}, GeneratedVersionLabel: sql.NullString{String: generatedVersionLabel, Valid: true}, ArtifactID: sql.NullString{String: artifactId, Valid: true},
		PipelineRunID: pipelineRunId, PipelineStageID: pipelineStageId,
	}); err != nil {
		return fmt.Errorf("complete pipeline run build version binding %s/%s: %w", pipelineRunId, pipelineStageId, err)
	}
	return nil
}

func pipelineRunFrom(id string, projectId sql.NullString, repositoryId, repositoryName, snapshotId, templateId, templateName string, templateVersion int64, trigger, triggerRef, variablesSnapshot, runStatus string, retryOf sql.NullString, startedAt, finishedAt sql.NullTime, errorMessage sql.NullString, createdAt time.Time) model.PipelineRun {
	return model.PipelineRun{
		Id: id, ProjectId: dbmodel.StringPtr(projectId), RepositoryId: repositoryId, RepositoryName: repositoryName,
		SnapshotId: snapshotId, TemplateId: templateId, TemplateName: templateName, TemplateVersion: int(templateVersion),
		Trigger: trigger, TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: runStatus,
		RetryOf: dbmodel.StringPtr(retryOf), StartedAt: dbmodel.TimePtr(startedAt), FinishedAt: dbmodel.TimePtr(finishedAt),
		ErrorMessage: dbmodel.StringPtr(errorMessage), CreatedAt: createdAt,
	}
}

func pipelineStageRunFrom(id, pipelineRunId, stageId, stageName, runStatus string, startedAt, finishedAt sql.NullTime, exitCode sql.NullInt64, errorMessage sql.NullString) model.PipelineStageRun {
	return model.PipelineStageRun{
		Id: id, PipelineRunId: pipelineRunId, StageId: stageId, StageName: stageName, Status: runStatus,
		StartedAt: dbmodel.TimePtr(startedAt), FinishedAt: dbmodel.TimePtr(finishedAt),
		ExitCode: dbmodel.IntPtrFromNullInt64(exitCode), ErrorMessage: dbmodel.StringPtr(errorMessage),
	}
}

func artifactFrom(id string, projectId sql.NullString, pipelineRunId, repositoryId, repositoryName, templateId, templateName, pipelineStageId, stageName, collector, name string, location, value, valueFormat, imageRef, localImageSha256, sourceCommitSha, applicationId, applicationName, sourceVersionId, sourceVersionLabel, generatedVersionId, generatedVersionLabel, versionComponentName, versionComponentId sql.NullString, createdAt time.Time) model.Artifact {
	return model.Artifact{
		Id: id, ProjectId: dbmodel.StringPtr(projectId), PipelineRunId: pipelineRunId, RepositoryId: repositoryId,
		RepositoryName: repositoryName, TemplateId: templateId, TemplateName: templateName, PipelineStageId: pipelineStageId, StageName: stageName,
		Collector: collector, Name: name, Location: dbmodel.StringPtr(location), Value: dbmodel.StringPtr(value), ValueFormat: dbmodel.StringPtr(valueFormat), ImageRef: dbmodel.StringPtr(imageRef), LocalImageSha256: dbmodel.StringPtr(localImageSha256),
		SourceCommitSha: dbmodel.StringPtr(sourceCommitSha), ApplicationId: dbmodel.StringPtr(applicationId), ApplicationName: dbmodel.StringPtr(applicationName),
		SourceVersionId: dbmodel.StringPtr(sourceVersionId), SourceVersionLabel: dbmodel.StringPtr(sourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(generatedVersionId), GeneratedVersionLabel: dbmodel.StringPtr(generatedVersionLabel),
		VersionComponentId: dbmodel.StringPtr(versionComponentId), VersionComponentName: dbmodel.StringPtr(versionComponentName), CreatedAt: createdAt,
	}
}

func pipelineRunBuildVersionBindingFrom(row pipelinerunsqlc.PipelineRunBuildVersionBinding) model.PipelineRunBuildVersionBinding {
	return model.PipelineRunBuildVersionBinding{
		PipelineRunId: row.PipelineRunID, PipelineStageId: row.PipelineStageID, ApplicationId: row.ApplicationID, ApplicationName: row.ApplicationName,
		ComponentName: row.ComponentName, SourceVersionId: row.SourceVersionID, SourceVersionLabel: row.SourceVersionLabel,
		GeneratedVersionId: dbmodel.StringPtr(row.GeneratedVersionID), GeneratedVersionLabel: dbmodel.StringPtr(row.GeneratedVersionLabel), ArtifactId: dbmodel.StringPtr(row.ArtifactID),
	}
}
