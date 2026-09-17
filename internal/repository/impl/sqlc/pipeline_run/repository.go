package pipelinerunrepo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	pipelinerunsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/pipeline_run"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.PipelineRunStore = Repository{}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return Repository{db: db} }
func (r Repository) q(ctx context.Context) *pipelinerunsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *pipelinerunsqlc.Queries { return pipelinerunsqlc.New(dbtx) })
}

func (r Repository) ListPipelineRuns(ctx context.Context, projectId, repositoryId, pipelineId string, from, to *time.Time, page, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	params := pipelinerunsqlc.ListPipelineRunsParams{
		ProjectId:    nullableArgument(projectId),
		RepositoryId: nullableArgument(repositoryId),
		PipelineId:   nullableArgument(pipelineId),
		FromAt:       nullableTimeArgument(from),
		ToAt:         nullableTimeArgument(to),
		Limit:        int32(perPage),
		Offset:       int32((page - 1) * perPage),
	}
	count, err := r.q(ctx).CountPipelineRuns(ctx, pipelinerunsqlc.CountPipelineRunsParams{
		ProjectId:    params.ProjectId,
		RepositoryId: params.RepositoryId,
		PipelineId:   params.PipelineId,
		FromAt:       params.FromAt,
		ToAt:         params.ToAt,
	})
	if err != nil {
		return repository.Page[model.PipelineRun]{}, translate(err)
	}
	rows, err := r.q(ctx).ListPipelineRuns(ctx, params)
	if err != nil {
		return repository.Page[model.PipelineRun]{}, translate(err)
	}
	items := make([]model.PipelineRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineRunModel(row))
	}
	return repository.Page[model.PipelineRun]{Items: items, Total: int(count), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListPipelineRunsByPipeline(ctx context.Context, projectId string, pipelineId string, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return r.ListPipelineRuns(ctx, projectId, "", pipelineId, nil, nil, page, perPage)
}

func (r Repository) PipelineRun(ctx context.Context, projectId string, id string) (model.PipelineRun, error) {
	item, err := r.q(ctx).PipelineRunById(ctx, pipelinerunsqlc.PipelineRunByIdParams{Id: id, ProjectId: nullableArgument(projectId)})
	return pipelineRunModel(item), translate(err)
}

func (r Repository) DeletePipelineRun(ctx context.Context, projectId, id string) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		deleteParams := pipelinerunsqlc.DeletePipelineRunArtifactsParams{PipelineRunId: id, ProjectId: requiredArgument(projectId)}
		if err := q.DeletePipelineRunArtifacts(txCtx, deleteParams); err != nil {
			return translate(err)
		}
		if err := q.DeletePipelineStageRuns(txCtx, pipelinerunsqlc.DeletePipelineStageRunsParams(deleteParams)); err != nil {
			return translate(err)
		}
		if err := q.DeletePipelineRunVersionBinding(txCtx, pipelinerunsqlc.DeletePipelineRunVersionBindingParams(deleteParams)); err != nil {
			return translate(err)
		}
		return translate(q.DeletePipelineRun(txCtx, pipelinerunsqlc.DeletePipelineRunParams{Id: id, ProjectId: requiredArgument(projectId)}))
	})
}

func (r Repository) ListPipelineStageRuns(ctx context.Context, projectId, runId string) ([]model.PipelineStageRun, error) {
	rows, err := r.q(ctx).ListPipelineStageRuns(ctx, pipelinerunsqlc.ListPipelineStageRunsParams{PipelineRunId: runId, ProjectId: requiredArgument(projectId)})
	if err != nil {
		return nil, translate(err)
	}
	items := make([]model.PipelineStageRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineStageRunModel(row))
	}
	return items, nil
}

func (r Repository) PipelineStageRun(ctx context.Context, projectId, id string) (model.PipelineStageRun, error) {
	item, err := r.q(ctx).PipelineStageRunById(ctx, pipelinerunsqlc.PipelineStageRunByIdParams{Id: id, ProjectId: requiredArgument(projectId)})
	return pipelineStageRunModel(item), translate(err)
}

func (r Repository) ListArtifacts(ctx context.Context, projectId, repositoryId, pipelineId string, page, perPage int, search string) (repository.Page[model.Artifact], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	rawSearch := strings.TrimSpace(search)
	searchPattern := sql.NullString{String: "%" + rawSearch + "%", Valid: rawSearch != ""}
	params := pipelinerunsqlc.ListArtifactsParams{
		ProjectId:     nullableArgument(projectId),
		RepositoryId:  nullableArgument(repositoryId),
		PipelineId:    nullableArgument(pipelineId),
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	}
	count, err := r.q(ctx).CountArtifacts(ctx, pipelinerunsqlc.CountArtifactsParams{
		ProjectId:     params.ProjectId,
		RepositoryId:  params.RepositoryId,
		PipelineId:    params.PipelineId,
		SearchPattern: params.SearchPattern,
	})
	if err != nil {
		return repository.Page[model.Artifact]{}, translate(err)
	}
	rows, err := r.q(ctx).ListArtifacts(ctx, params)
	if err != nil {
		return repository.Page[model.Artifact]{}, translate(err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, listArtifactModel(row))
	}
	return repository.Page[model.Artifact]{Items: items, Total: int(count), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListArtifactsByRun(ctx context.Context, projectId string, runId string) ([]model.Artifact, error) {
	rows, err := r.q(ctx).ListArtifactsByRun(ctx, pipelinerunsqlc.ListArtifactsByRunParams{PipelineRunId: runId, ProjectId: nullableArgument(projectId)})
	if err != nil {
		return nil, translate(err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, runArtifactModel(row))
	}
	return items, nil
}

func (r Repository) Artifact(ctx context.Context, projectId string, id string) (model.Artifact, error) {
	item, err := r.q(ctx).ArtifactById(ctx, pipelinerunsqlc.ArtifactByIdParams{Id: id, ProjectId: nullableArgument(projectId)})
	return artifactModel(item), translate(err)
}

func (r Repository) CreatePipelineRun(ctx context.Context, run model.PipelineRun, binding *model.PipelineRunVersionBinding, stageRuns []model.PipelineStageRun) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.q(txCtx).InsertPipelineRun(txCtx, pipelineRunParams(run)); err != nil {
			return translate(err)
		}
		for _, stageRun := range stageRuns {
			if err := r.q(txCtx).InsertPipelineStageRun(txCtx, pipelineStageRunParams(stageRun)); err != nil {
				return translate(err)
			}
		}
		if binding == nil {
			return nil
		}
		return translate(r.q(txCtx).InsertPipelineRunVersionBinding(txCtx, pipelinerunsqlc.InsertPipelineRunVersionBindingParams{
			PipelineRunId:         run.Id,
			ApplicationId:         binding.ApplicationId,
			ApplicationName:       binding.ApplicationName,
			SourceVersionId:       binding.SourceVersionId,
			SourceVersionLabel:    binding.SourceVersionLabel,
			GeneratedVersionId:    nullString(binding.GeneratedVersionId),
			GeneratedVersionLabel: nullString(binding.GeneratedVersionLabel),
		}))
	})
}

func (r Repository) RepositoryHasActivePipelineRun(ctx context.Context, projectId, repositoryId string) (bool, error) {
	count, err := r.q(ctx).CountActivePipelineRunsByRepository(ctx, pipelinerunsqlc.CountActivePipelineRunsByRepositoryParams{ProjectId: requiredArgument(projectId), RepositoryId: repositoryId, WaitingStatus: status.WorkStatusWaitingToRun, RunningStatus: status.WorkStatusRunning})
	return count > 0, translate(err)
}

func (r Repository) HasCDConfigurationReferences(ctx context.Context, projectId string) (bool, error) {
	count, err := r.q(ctx).CountCDConfigurationReferences(ctx, requiredArgument(projectId))
	return count > 0, translate(err)
}

func (r Repository) PipelineRunVersionBinding(ctx context.Context, projectId, runId string) (model.PipelineRunVersionBinding, error) {
	item, err := r.q(ctx).PipelineRunVersionBindingByRunId(ctx, pipelinerunsqlc.PipelineRunVersionBindingByRunIdParams{PipelineRunId: runId, ProjectId: requiredArgument(projectId)})
	return pipelineRunVersionBindingModel(item), translate(err)
}

func (r Repository) CancelPipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	canceled := false
	err := tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		now := time.Now().UTC()
		rows, err := r.q(txCtx).CancelPipelineRun(txCtx, pipelinerunsqlc.CancelPipelineRunParams{Status: status.WorkStatusCanceled, FinishedAt: sql.NullTime{Time: now, Valid: true}, ErrorMessage: sql.NullString{String: "Cancelled by user", Valid: true}, Id: id, ProjectId: requiredArgument(projectId), WaitingStatus: status.WorkStatusWaitingToRun, RunningStatus: status.WorkStatusRunning})
		if err != nil {
			return translate(err)
		}
		if rows == 0 {
			return nil
		}
		canceled = true
		_, err = r.q(txCtx).CancelRunningPipelineStageRuns(txCtx, pipelinerunsqlc.CancelRunningPipelineStageRunsParams{Status: status.WorkStatusCanceled, FinishedAt: sql.NullTime{Time: now, Valid: true}, ErrorMessage: sql.NullString{String: "Cancelled by user", Valid: true}, PipelineRunId: id, ExpectedStatus: status.WorkStatusRunning, ProjectId: requiredArgument(projectId)})
		return translate(err)
	})
	return canceled, err
}

func (r Repository) CancelRunningPipelineStageRuns(ctx context.Context, projectId, runId string) error {
	_, err := r.q(ctx).CancelRunningPipelineStageRuns(ctx, pipelinerunsqlc.CancelRunningPipelineStageRunsParams{Status: status.WorkStatusCanceled, FinishedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, ErrorMessage: sql.NullString{String: "Cancelled by user", Valid: true}, PipelineRunId: runId, ExpectedStatus: status.WorkStatusRunning, ProjectId: requiredArgument(projectId)})
	return translate(err)
}

func (r Repository) BeginPipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	rows, err := r.q(ctx).BeginPipelineRun(ctx, pipelinerunsqlc.BeginPipelineRunParams{Status: status.WorkStatusRunning, StartedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, Id: id, ProjectId: requiredArgument(projectId), ExpectedStatus: status.WorkStatusWaitingToRun})
	return rows == 1, translate(err)
}

func (r Repository) CompletePipelineRun(ctx context.Context, projectId, id, statusValue, message string) (bool, error) {
	rows, err := r.q(ctx).CompletePipelineRun(ctx, pipelinerunsqlc.CompletePipelineRunParams{Status: statusValue, ErrorMessage: optionalText(message), FinishedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, Id: id, ProjectId: requiredArgument(projectId), ExpectedStatus: status.WorkStatusRunning})
	return rows == 1, translate(err)
}

func (r Repository) BeginPipelineStageRun(ctx context.Context, projectId, id string) (bool, error) {
	rows, err := r.q(ctx).BeginPipelineStageRun(ctx, pipelinerunsqlc.BeginPipelineStageRunParams{Status: status.WorkStatusRunning, StartedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, Id: id, ExpectedStatus: status.WorkStatusWaitingToRun, ProjectId: requiredArgument(projectId), ParentStatus: status.WorkStatusRunning})
	return rows == 1, translate(err)
}

func (r Repository) CompletePipelineStageRun(ctx context.Context, projectId string, stage model.PipelineStageRun) (bool, error) {
	rows, err := r.q(ctx).CompletePipelineStageRun(ctx, pipelinerunsqlc.CompletePipelineStageRunParams{Status: stage.Status, FinishedAt: dbmodel.NullTime(stage.FinishedAt), ExitCode: dbmodel.NullInt64FromIntPtr(stage.ExitCode), ErrorMessage: nullString(stage.ErrorMessage), Id: stage.Id, ExpectedStatus: status.WorkStatusRunning, ProjectId: requiredArgument(projectId)})
	return rows == 1, translate(err)
}

func (r Repository) CreateArtifact(ctx context.Context, artifact model.Artifact) error {
	return translate(r.q(ctx).InsertArtifact(ctx, artifactParams(artifact)))
}

func (r Repository) CommandArtifactByRunStageAndName(ctx context.Context, projectId, runId, stageId, name string) (model.Artifact, error) {
	item, err := r.q(ctx).CommandArtifactByRunStageAndName(ctx, pipelinerunsqlc.CommandArtifactByRunStageAndNameParams{PipelineRunId: runId, PipelineStageId: stageId, Name: name, ProjectId: requiredArgument(projectId)})
	if err != nil {
		return model.Artifact{}, translate(err)
	}
	return model.Artifact{Id: item.Id, Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat)}, nil
}

func (r Repository) CompletePipelineRunVersionBinding(ctx context.Context, projectId, runId, versionId, label string) error {
	return translate(r.q(ctx).CompletePipelineRunVersionBinding(ctx, pipelinerunsqlc.CompletePipelineRunVersionBindingParams{GeneratedVersionId: sql.NullString{String: versionId, Valid: true}, GeneratedVersionLabel: sql.NullString{String: label, Valid: true}, PipelineRunId: runId, ProjectId: requiredArgument(projectId)}))
}

func pipelineRunParams(item model.PipelineRun) pipelinerunsqlc.InsertPipelineRunParams {
	return pipelinerunsqlc.InsertPipelineRunParams{Id: item.Id, ProjectId: nullString(item.ProjectId), RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int64(item.PipelineVersion), Trigger: item.Trigger, RepositoryRef: item.RepositoryRef, VariablesSnapshot: item.VariablesSnapshot, Status: item.Status, RetryOf: nullString(item.RetryOf), StartedAt: dbmodel.NullTime(item.StartedAt), FinishedAt: dbmodel.NullTime(item.FinishedAt), ErrorMessage: nullString(item.ErrorMessage), CreatedAt: timeOrNow(item.CreatedAt)}
}

func pipelineStageRunParams(item model.PipelineStageRun) pipelinerunsqlc.InsertPipelineStageRunParams {
	return pipelinerunsqlc.InsertPipelineStageRunParams{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: dbmodel.NullTime(item.StartedAt), FinishedAt: dbmodel.NullTime(item.FinishedAt), ExitCode: dbmodel.NullInt64FromIntPtr(item.ExitCode), ErrorMessage: nullString(item.ErrorMessage)}
}

func artifactParams(item model.Artifact) pipelinerunsqlc.InsertArtifactParams {
	return pipelinerunsqlc.InsertArtifactParams{Id: item.Id, ProjectId: nullString(item.ProjectId), PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: nullString(item.Location), Value: nullString(item.Value), ValueFormat: nullString(item.ValueFormat), ImageRef: nullString(item.ImageRef), LocalImageSha256: nullString(item.LocalImageSha256), SourceArtifactId: nullString(item.SourceArtifactId), CreatedAt: timeOrNow(item.CreatedAt)}
}

func pipelineRunModel(item pipelinerunsqlc.PipelineRun) model.PipelineRun {
	return model.PipelineRun{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int(item.PipelineVersion), Trigger: item.Trigger, RepositoryRef: item.RepositoryRef, VariablesSnapshot: item.VariablesSnapshot, Status: item.Status, RetryOf: dbmodel.StringPtr(item.RetryOf), StartedAt: dbmodel.TimePtr(item.StartedAt), FinishedAt: dbmodel.TimePtr(item.FinishedAt), ErrorMessage: dbmodel.StringPtr(item.ErrorMessage), CreatedAt: item.CreatedAt}
}

func pipelineStageRunModel(item pipelinerunsqlc.PipelineStageRun) model.PipelineStageRun {
	return model.PipelineStageRun{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: dbmodel.TimePtr(item.StartedAt), FinishedAt: dbmodel.TimePtr(item.FinishedAt), ExitCode: dbmodel.IntPtrFromNullInt64(item.ExitCode), ErrorMessage: dbmodel.StringPtr(item.ErrorMessage)}
}

func pipelineRunVersionBindingModel(item pipelinerunsqlc.PipelineRunVersionBinding) model.PipelineRunVersionBinding {
	return model.PipelineRunVersionBinding{PipelineRunId: item.PipelineRunId, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, SourceVersionId: item.SourceVersionId, SourceVersionLabel: item.SourceVersionLabel, GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionId), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel)}
}

func artifactModel(item pipelinerunsqlc.ArtifactByIdRow) model.Artifact {
	return model.Artifact{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactId), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationId), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionId), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionId), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentId), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func listArtifactModel(item pipelinerunsqlc.ListArtifactsRow) model.Artifact {
	return model.Artifact{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactId), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationId), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionId), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionId), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentId), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func runArtifactModel(item pipelinerunsqlc.ListArtifactsByRunRow) model.Artifact {
	return model.Artifact{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactId), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationId), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionId), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionId), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentId), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func nullableArgument(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func requiredArgument(value string) sql.NullString {
	return sql.NullString{String: strings.TrimSpace(value), Valid: true}
}

func nullableTimeArgument(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullString(value *string) sql.NullString {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func optionalText(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func timeOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func translate(err error) error {
	if err == nil {
		return nil
	}
	return sqlcommon.TranslateError(err)
}
