package pipelinerepo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	pipelinesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/pipeline"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.PipelineStore = Repository{}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return Repository{db: db} }
func (r Repository) q(ctx context.Context) *pipelinesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *pipelinesqlc.Queries { return pipelinesqlc.New(dbtx) })
}

func (r Repository) Pipeline(ctx context.Context, id string) (model.Pipeline, error) {
	item, err := r.q(ctx).PipelineByID(ctx, id)
	return pipelineModel(item), translate(err)
}
func (r Repository) PipelineByName(ctx context.Context, projectID, name string) (model.Pipeline, error) {
	item, err := r.q(ctx).PipelineByName(ctx, pipelinesqlc.PipelineByNameParams{ProjectID: nullString(&projectID), Name: name})
	return pipelineModel(item), translate(err)
}
func (r Repository) ListPipelines(ctx context.Context, projectID, kind string, page, perPage int, search string) (repository.Page[model.Pipeline], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	projectID = strings.TrimSpace(projectID)
	kind = strings.TrimSpace(kind)
	search = strings.TrimSpace(search)
	pattern := "%" + search + "%"
	args := pipelinesqlc.ListPipelinesParams{ProjectID: nullString(&projectID), Kind: kind, Search: search, SearchPattern: pattern, Limit: int64(perPage), Offset: int64((page - 1) * perPage)}
	count, err := r.q(ctx).CountPipelines(ctx, pipelinesqlc.CountPipelinesParams{ProjectID: nullString(&projectID), Kind: kind, Search: search, SearchPattern: pattern})
	if err != nil {
		return repository.Page[model.Pipeline]{}, translate(err)
	}
	rows, err := r.q(ctx).ListPipelines(ctx, args)
	if err != nil {
		return repository.Page[model.Pipeline]{}, translate(err)
	}
	items := make([]model.Pipeline, 0, len(rows))
	for _, item := range rows {
		items = append(items, pipelineModel(item))
	}
	return repository.Page[model.Pipeline]{Items: items, Total: int(count), Page: page, PerPage: perPage}, nil
}
func (r Repository) CreatePipeline(ctx context.Context, item model.Pipeline) error {
	return translate(r.q(ctx).CreatePipeline(ctx, pipelineParams(item)))
}
func (r Repository) UpdatePipeline(ctx context.Context, item model.Pipeline) error {
	return translate(r.q(ctx).UpdatePipeline(ctx, pipelinesqlc.UpdatePipelineParams{Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int64(item.Version), VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionID: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), UpdatedAt: time.Now().UTC(), ID: item.Id}))
}
func (r Repository) DeletePipeline(ctx context.Context, id string) error {
	return translate(r.q(ctx).DeletePipeline(ctx, id))
}
func (r Repository) PipelineStages(ctx context.Context, pipelineID string) ([]model.PipelineStage, error) {
	rows, err := r.q(ctx).PipelineStages(ctx, pipelineID)
	if err != nil {
		return nil, translate(err)
	}
	result := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		result = append(result, stageModel(row))
	}
	return result, nil
}
func (r Repository) PipelineStage(ctx context.Context, id string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageByID(ctx, id)
	return stageModel(row), translate(err)
}
func (r Repository) CreatePipelineStage(ctx context.Context, item model.PipelineStage) error {
	return translate(r.q(ctx).InsertPipelineStage(ctx, stageParams(item)))
}
func (r Repository) UpdatePipelineStage(ctx context.Context, item model.PipelineStage) error {
	return translate(r.q(ctx).UpdatePipelineStage(ctx, pipelinesqlc.UpdatePipelineStageParams{Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: nullString(item.Artifacts), DependsOn: item.DependsOn, SortOrder: int64(item.SortOrder), Description: item.Description, UpdatedAt: time.Now().UTC(), ID: item.Id}))
}
func (r Repository) DeletePipelineStage(ctx context.Context, id string) error {
	return translate(r.q(ctx).DeletePipelineStage(ctx, id))
}
func (r Repository) CreatePipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.CreatePipeline(txCtx, pipeline); err != nil {
			return err
		}
		for _, stage := range stages {
			if err := r.CreatePipelineStage(txCtx, stage); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r Repository) UpdatePipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.UpdatePipeline(txCtx, pipeline); err != nil {
			return err
		}
		q := r.q(txCtx)
		if err := translate(q.DeletePipelineStages(txCtx, pipeline.Id)); err != nil {
			return err
		}
		for _, stage := range stages {
			if err := translate(q.InsertPipelineStage(txCtx, stageParams(stage))); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r Repository) LatestPipelineSnapshot(ctx context.Context, pipelineID string) (model.PipelineSnapshot, error) {
	item, err := r.q(ctx).LatestPipelineSnapshot(ctx, pipelineID)
	return snapshotModel(item), translate(err)
}
func (r Repository) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	item, err := r.q(ctx).PipelineSnapshotByID(ctx, id)
	return snapshotModel(item), translate(err)
}
func (r Repository) CreatePipelineSnapshot(ctx context.Context, item model.PipelineSnapshot) error {
	return translate(r.q(ctx).InsertPipelineSnapshot(ctx, pipelinesqlc.InsertPipelineSnapshotParams{ID: item.Id, ProjectID: nullString(item.ProjectId), PipelineID: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int64(item.PipelineVersion), SourcePipelineID: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int64(item.SourceTemplateVersion), ApplicationID: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), RepositoryID: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionID: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), StagesSnapshot: item.StagesSnapshot, VariablesSnapshot: item.VariablesSnapshot, CreatedAt: timeOrNow(item.CreatedAt)}))
}

func pipelineParams(item model.Pipeline) pipelinesqlc.CreatePipelineParams {
	return pipelinesqlc.CreatePipelineParams{ID: item.Id, ProjectID: nullString(item.ProjectId), Kind: item.Kind, SourcePipelineID: nullString(item.SourcePipelineId), SourceTemplateName: nullString(item.SourceTemplateName), SourceTemplateVersion: nullInt(item.SourceTemplateVersion), ApplicationID: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), RepositoryID: nullString(item.RepositoryId), RepositoryName: nullString(item.RepositoryName), VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionID: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int64(item.Version), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func stageParams(item model.PipelineStage) pipelinesqlc.InsertPipelineStageParams {
	return pipelinesqlc.InsertPipelineStageParams{ID: item.Id, PipelineID: item.PipelineId, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: nullString(item.Artifacts), DependsOn: item.DependsOn, SortOrder: int64(item.SortOrder), Description: item.Description, CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func pipelineModel(item pipelinesqlc.Pipeline) model.Pipeline {
	result := model.Pipeline{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), Kind: item.Kind, SourcePipelineId: dbmodel.StringPtr(item.SourcePipelineID), SourceTemplateName: dbmodel.StringPtr(item.SourceTemplateName), ApplicationId: dbmodel.StringPtr(item.ApplicationID), ApplicationName: dbmodel.StringPtr(item.ApplicationName), RepositoryId: dbmodel.StringPtr(item.RepositoryID), RepositoryName: dbmodel.StringPtr(item.RepositoryName), VersionForkStrategy: dbmodel.StringPtr(item.VersionForkStrategy), FixedVersionId: dbmodel.StringPtr(item.FixedVersionID), FixedVersionLabel: dbmodel.StringPtr(item.FixedVersionLabel), Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int(item.Version), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.SourceTemplateVersion.Valid {
		value := int(item.SourceTemplateVersion.Int64)
		result.SourceTemplateVersion = &value
	}
	return result
}
func stageModel(item pipelinesqlc.PipelineStage) model.PipelineStage {
	return model.PipelineStage{Id: item.ID, PipelineId: item.PipelineID, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: dbmodel.StringPtr(item.Artifacts), DependsOn: item.DependsOn, SortOrder: int(item.SortOrder), Description: item.Description, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func snapshotModel(item pipelinesqlc.PipelineSnapshot) model.PipelineSnapshot {
	return model.PipelineSnapshot{Id: item.ID, ProjectId: dbmodel.StringPtr(item.ProjectID), PipelineId: item.PipelineID, PipelineName: item.PipelineName, PipelineVersion: int(item.PipelineVersion), SourcePipelineId: item.SourcePipelineID, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int(item.SourceTemplateVersion), ApplicationId: dbmodel.StringPtr(item.ApplicationID), ApplicationName: dbmodel.StringPtr(item.ApplicationName), RepositoryId: item.RepositoryID, RepositoryName: item.RepositoryName, VersionForkStrategy: dbmodel.StringPtr(item.VersionForkStrategy), FixedVersionId: dbmodel.StringPtr(item.FixedVersionID), FixedVersionLabel: dbmodel.StringPtr(item.FixedVersionLabel), StagesSnapshot: item.StagesSnapshot, VariablesSnapshot: item.VariablesSnapshot, CreatedAt: item.CreatedAt}
}
func nullString(value *string) sql.NullString {
	if value == nil || strings.TrimSpace(*value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}
func nullInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
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
