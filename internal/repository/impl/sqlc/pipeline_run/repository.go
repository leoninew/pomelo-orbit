package pipelinerunrepo

import (
	"context"
	"database/sql"
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

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return Repository{db: db} }
func (r Repository) q(ctx context.Context) *pipelinerunsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *pipelinerunsqlc.Queries { return pipelinerunsqlc.New(dbtx) })
}

func (r Repository) ListPipelineRuns(ctx context.Context, projectID, repositoryID, pipelineID string, from, to *time.Time, page, perPage int) (repository.Page[model.PipelineRun], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	params := pipelinerunsqlc.ListPipelineRunsParams{
		ProjectID:    nullableArgument(projectID),
		RepositoryID: strings.TrimSpace(repositoryID),
		PipelineID:   strings.TrimSpace(pipelineID),
		FromAt:       nullableTimeArgument(from),
		ToAt:         nullableTimeArgument(to),
		Limit:        int64(perPage),
		Offset:       int64((page - 1) * perPage),
	}
	count, err := r.q(ctx).CountPipelineRuns(ctx, pipelinerunsqlc.CountPipelineRunsParams{
		ProjectID:    params.ProjectID,
		RepositoryID: params.RepositoryID,
		PipelineID:   params.PipelineID,
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

func (r Repository) ListPipelineRunsByPipeline(ctx context.Context, pipelineID string, page, perPage int) (repository.Page[model.PipelineRun], error) {
	return r.ListPipelineRuns(ctx, "", "", pipelineID, nil, nil, page, perPage)
}

func (r Repository) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	item, err := r.q(ctx).PipelineRunByID(ctx, id)
	return pipelineRunModel(item), translate(err)
}

func (r Repository) ListPipelineStageRuns(ctx context.Context, runID string) ([]model.PipelineStageRun, error) {
	rows, err := r.q(ctx).ListPipelineStageRuns(ctx, runID)
	if err != nil {
		return nil, translate(err)
	}
	items := make([]model.PipelineStageRun, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineStageRunModel(row))
	}
	return items, nil
}

func (r Repository) PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error) {
	item, err := r.q(ctx).PipelineStageRunByID(ctx, id)
	return pipelineStageRunModel(item), translate(err)
}

func (r Repository) ListArtifacts(ctx context.Context, projectID, repositoryID, pipelineID string, page, perPage int, search string) (repository.Page[model.Artifact], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	rawSearch := strings.TrimSpace(search)
	params := pipelinerunsqlc.ListArtifactsParams{
		ProjectID:     nullableArgument(projectID),
		RepositoryID:  strings.TrimSpace(repositoryID),
		PipelineID:    strings.TrimSpace(pipelineID),
		Search:        rawSearch,
		SearchPattern: "%" + rawSearch + "%",
		Limit:         int64(perPage),
		Offset:        int64((page - 1) * perPage),
	}
	count, err := r.q(ctx).CountArtifacts(ctx, pipelinerunsqlc.CountArtifactsParams{
		ProjectID:     params.ProjectID,
		RepositoryID:  params.RepositoryID,
		PipelineID:    params.PipelineID,
		Search:        params.Search,
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

func (r Repository) ListArtifactsByRun(ctx context.Context, projectID *string, runID string) ([]model.Artifact, error) {
	rows, err := r.q(ctx).ListArtifactsByRun(ctx, pipelinerunsqlc.ListArtifactsByRunParams{PipelineRunID: runID, ProjectID: nullableStringArgument(projectID)})
	if err != nil {
		return nil, translate(err)
	}
	items := make([]model.Artifact, 0, len(rows))
	for _, row := range rows {
		items = append(items, runArtifactModel(row))
	}
	return items, nil
}

func (r Repository) Artifact(ctx context.Context, id string) (model.Artifact, error) {
	item, err := r.q(ctx).ArtifactByID(ctx, id)
	return artifactModel(item), translate(err)
}

func (r Repository) CreatePipelineRun(ctx context.Context, run model.PipelineRun, binding *model.PipelineRunVersionBinding) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.q(txCtx).InsertPipelineRun(txCtx, pipelineRunParams(run)); err != nil {
			return translate(err)
		}
		if binding == nil {
			return nil
		}
		return translate(r.q(txCtx).InsertPipelineRunVersionBinding(txCtx, pipelinerunsqlc.InsertPipelineRunVersionBindingParams{
			PipelineRunID:         run.Id,
			ApplicationID:         binding.ApplicationId,
			ApplicationName:       binding.ApplicationName,
			SourceVersionID:       binding.SourceVersionId,
			SourceVersionLabel:    binding.SourceVersionLabel,
			GeneratedVersionID:    nullString(binding.GeneratedVersionId),
			GeneratedVersionLabel: nullString(binding.GeneratedVersionLabel),
		}))
	})
}

func (r Repository) RepositoryHasRunningPipelineRun(ctx context.Context, repositoryID string) (bool, error) {
	count, err := r.q(ctx).CountRunningPipelineRunsByRepository(ctx, pipelinerunsqlc.CountRunningPipelineRunsByRepositoryParams{RepositoryID: repositoryID, Status: status.WorkStatusRunning})
	return count > 0, translate(err)
}

func (r Repository) PipelineRunVersionBinding(ctx context.Context, runID string) (model.PipelineRunVersionBinding, error) {
	item, err := r.q(ctx).PipelineRunVersionBindingByRunID(ctx, runID)
	return pipelineRunVersionBindingModel(item), translate(err)
}

func (r Repository) CancelPipelineRun(ctx context.Context, id string) error {
	return translate(r.q(ctx).CancelPipelineRun(ctx, pipelinerunsqlc.CancelPipelineRunParams{Status: status.WorkStatusCanceled, FinishedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, ID: id}))
}

func (r Repository) MarkPipelineRunRunning(ctx context.Context, id string) error {
	return translate(r.q(ctx).MarkPipelineRunRunning(ctx, pipelinerunsqlc.MarkPipelineRunRunningParams{Status: status.WorkStatusRunning, StartedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, ID: id}))
}

func (r Repository) CompletePipelineRun(ctx context.Context, id, statusValue, message string) error {
	return translate(r.q(ctx).CompletePipelineRun(ctx, pipelinerunsqlc.CompletePipelineRunParams{Status: statusValue, ErrorMessage: optionalText(message), FinishedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true}, ID: id}))
}

func (r Repository) InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return translate(r.q(ctx).InsertPipelineStageRun(ctx, pipelineStageRunParams(stage)))
}

func (r Repository) UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return translate(r.q(ctx).UpdatePipelineStageRun(ctx, pipelinerunsqlc.UpdatePipelineStageRunParams{Status: stage.Status, StartedAt: dbmodel.NullTime(stage.StartedAt), FinishedAt: dbmodel.NullTime(stage.FinishedAt), ExitCode: dbmodel.NullInt64FromIntPtr(stage.ExitCode), ErrorMessage: nullString(stage.ErrorMessage), ID: stage.Id}))
}

func (r Repository) CreateArtifact(ctx context.Context, artifact model.Artifact) error {
	return translate(r.q(ctx).InsertArtifact(ctx, artifactParams(artifact)))
}

func (r Repository) CommandArtifactByRunStageAndName(ctx context.Context, runID, stageID, name string) (model.Artifact, error) {
	item, err := r.q(ctx).CommandArtifactByRunStageAndName(ctx, pipelinerunsqlc.CommandArtifactByRunStageAndNameParams{PipelineRunID: runID, PipelineStageID: stageID, Name: name})
	if err != nil {
		return model.Artifact{}, translate(err)
	}
	return model.Artifact{Id: item.ID, Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat)}, nil
}

func (r Repository) CompletePipelineRunVersionBinding(ctx context.Context, runID, versionID, label string) error {
	return translate(r.q(ctx).CompletePipelineRunVersionBinding(ctx, pipelinerunsqlc.CompletePipelineRunVersionBindingParams{GeneratedVersionID: sql.NullString{String: versionID, Valid: true}, GeneratedVersionLabel: sql.NullString{String: label, Valid: true}, PipelineRunID: runID}))
}

func pipelineRunParams(item model.PipelineRun) pipelinerunsqlc.InsertPipelineRunParams {
	return pipelinerunsqlc.InsertPipelineRunParams{ID: item.Id, ProjectID: nullString(item.ProjectId), RepositoryID: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotID: item.SnapshotId, PipelineID: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int64(item.PipelineVersion), Trigger: item.Trigger, TriggerRef: item.TriggerRef, VariablesSnapshot: item.VariablesSnapshot, Status: item.Status, RetryOf: nullString(item.RetryOf), StartedAt: dbmodel.NullTime(item.StartedAt), FinishedAt: dbmodel.NullTime(item.FinishedAt), ErrorMessage: nullString(item.ErrorMessage), CreatedAt: timeOrNow(item.CreatedAt)}
}

func pipelineStageRunParams(item model.PipelineStageRun) pipelinerunsqlc.InsertPipelineStageRunParams {
	return pipelinerunsqlc.InsertPipelineStageRunParams{ID: item.Id, PipelineRunID: item.PipelineRunId, StageID: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: dbmodel.NullTime(item.StartedAt), FinishedAt: dbmodel.NullTime(item.FinishedAt), ExitCode: dbmodel.NullInt64FromIntPtr(item.ExitCode), ErrorMessage: nullString(item.ErrorMessage)}
}

func artifactParams(item model.Artifact) pipelinerunsqlc.InsertArtifactParams {
	return pipelinerunsqlc.InsertArtifactParams{ID: item.Id, ProjectID: nullString(item.ProjectId), PipelineRunID: item.PipelineRunId, RepositoryID: item.RepositoryId, RepositoryName: item.RepositoryName, PipelineID: item.PipelineId, PipelineName: item.PipelineName, PipelineStageID: item.PipelineStageId, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: nullString(item.Location), Value: nullString(item.Value), ValueFormat: nullString(item.ValueFormat), ImageRef: nullString(item.ImageRef), LocalImageSha256: nullString(item.LocalImageSha256), SourceArtifactID: nullString(item.SourceArtifactId), CreatedAt: timeOrNow(item.CreatedAt)}
}

func pipelineRunModel(item pipelinerunsqlc.PipelineRun) model.PipelineRun {
	return model.PipelineRun{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), RepositoryId: item.RepositoryID, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotID, PipelineId: item.PipelineID, PipelineName: item.PipelineName, PipelineVersion: int(item.PipelineVersion), Trigger: item.Trigger, TriggerRef: item.TriggerRef, VariablesSnapshot: item.VariablesSnapshot, Status: item.Status, RetryOf: dbmodel.StringPtr(item.RetryOf), StartedAt: dbmodel.TimePtr(item.StartedAt), FinishedAt: dbmodel.TimePtr(item.FinishedAt), ErrorMessage: dbmodel.StringPtr(item.ErrorMessage), CreatedAt: item.CreatedAt}
}

func pipelineStageRunModel(item pipelinerunsqlc.PipelineStageRun) model.PipelineStageRun {
	return model.PipelineStageRun{Id: item.ID, PipelineRunId: item.PipelineRunID, StageId: item.StageID, StageName: item.StageName, Status: item.Status, StartedAt: dbmodel.TimePtr(item.StartedAt), FinishedAt: dbmodel.TimePtr(item.FinishedAt), ExitCode: dbmodel.IntPtrFromNullInt64(item.ExitCode), ErrorMessage: dbmodel.StringPtr(item.ErrorMessage)}
}

func pipelineRunVersionBindingModel(item pipelinerunsqlc.PipelineRunVersionBinding) model.PipelineRunVersionBinding {
	return model.PipelineRunVersionBinding{PipelineRunId: item.PipelineRunID, ApplicationId: item.ApplicationID, ApplicationName: item.ApplicationName, SourceVersionId: item.SourceVersionID, SourceVersionLabel: item.SourceVersionLabel, GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionID), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel)}
}

func artifactModel(item pipelinerunsqlc.ArtifactByIDRow) model.Artifact {
	return model.Artifact{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), PipelineRunId: item.PipelineRunID, RepositoryId: item.RepositoryID, RepositoryName: item.RepositoryName, PipelineId: item.PipelineID, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageID, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactID), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationID), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionID), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionID), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentID), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func listArtifactModel(item pipelinerunsqlc.ListArtifactsRow) model.Artifact {
	return model.Artifact{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), PipelineRunId: item.PipelineRunID, RepositoryId: item.RepositoryID, RepositoryName: item.RepositoryName, PipelineId: item.PipelineID, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageID, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactID), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationID), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionID), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionID), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentID), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func runArtifactModel(item pipelinerunsqlc.ListArtifactsByRunRow) model.Artifact {
	return model.Artifact{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), PipelineRunId: item.PipelineRunID, RepositoryId: item.RepositoryID, RepositoryName: item.RepositoryName, PipelineId: item.PipelineID, PipelineName: item.PipelineName, PipelineStageId: item.PipelineStageID, StageName: item.StageName, Collector: item.Collector, Name: item.Name, Location: dbmodel.StringPtr(item.Location), Value: dbmodel.StringPtr(item.Value), ValueFormat: dbmodel.StringPtr(item.ValueFormat), ImageRef: dbmodel.StringPtr(item.ImageRef), LocalImageSha256: dbmodel.StringPtr(item.LocalImageSha256), SourceArtifactId: dbmodel.StringPtr(item.SourceArtifactID), SourceCommitSha: dbmodel.StringPtr(item.SourceCommitSha), ApplicationId: dbmodel.StringPtr(item.ApplicationID), ApplicationName: dbmodel.StringPtr(item.ApplicationName), SourceVersionId: dbmodel.StringPtr(item.SourceVersionID), SourceVersionLabel: dbmodel.StringPtr(item.SourceVersionLabel), GeneratedVersionId: dbmodel.StringPtr(item.GeneratedVersionID), GeneratedVersionLabel: dbmodel.StringPtr(item.GeneratedVersionLabel), VersionComponentId: dbmodel.StringPtr(item.VersionComponentID), VersionComponentName: dbmodel.StringPtr(item.VersionComponentName), CreatedAt: item.CreatedAt}
}

func nullableArgument(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nullableStringArgument(value *string) any {
	if value == nil {
		return nil
	}
	return nullableArgument(*value)
}

func nullableTimeArgument(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
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
