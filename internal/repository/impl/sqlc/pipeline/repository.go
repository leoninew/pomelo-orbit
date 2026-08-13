package pipelinerepo

import (
	"context"
	"database/sql"
	"strings"
	"time"

	pipelinesqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/pipeline"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
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
	searchPattern := sql.NullString{String: "%" + search + "%", Valid: search != ""}
	kindValue := sql.NullString{String: kind, Valid: kind != ""}
	args := pipelinesqlc.ListPipelinesParams{ProjectID: projectID, Kind: kindValue, SearchPattern: searchPattern, Limit: int64(perPage), Offset: int64((page - 1) * perPage)}
	count, err := r.q(ctx).CountPipelines(ctx, pipelinesqlc.CountPipelinesParams{ProjectID: projectID, Kind: kindValue, SearchPattern: searchPattern})
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
	return translate(r.q(ctx).UpdatePipeline(ctx, pipelinesqlc.UpdatePipelineParams{Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int64(item.Version), ApplicationID: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionID: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), UpdatedAt: time.Now().UTC(), ID: item.Id}))
}
func (r Repository) DeletePipeline(ctx context.Context, id string) error {
	return translate(r.q(ctx).DeletePipeline(ctx, id))
}
func (r Repository) ListPipelineStageTemplates(ctx context.Context, projectID string, page, perPage int, search string) (repository.Page[model.PipelineStage], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	projectID, search = strings.TrimSpace(projectID), strings.TrimSpace(search)
	searchPattern := sql.NullString{String: "%" + search + "%", Valid: search != ""}
	args := pipelinesqlc.ListPipelineStageTemplatesParams{ProjectID: projectID, SearchPattern: searchPattern, Offset: int64((page - 1) * perPage), Limit: int64(perPage)}
	count, err := r.q(ctx).CountPipelineStageTemplates(ctx, pipelinesqlc.CountPipelineStageTemplatesParams{ProjectID: projectID, SearchPattern: searchPattern})
	if err != nil {
		return repository.Page[model.PipelineStage]{}, translate(err)
	}
	rows, err := r.q(ctx).ListPipelineStageTemplates(ctx, args)
	if err != nil {
		return repository.Page[model.PipelineStage]{}, translate(err)
	}
	items := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		items = append(items, stageModel(row))
	}
	return repository.Page[model.PipelineStage]{Items: items, Total: int(count), Page: page, PerPage: perPage}, nil
}
func (r Repository) PipelineStageTemplate(ctx context.Context, id string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageTemplateByID(ctx, id)
	return stageModel(row), translate(err)
}
func (r Repository) PipelineStageTemplateByName(ctx context.Context, projectID, name string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageTemplateByName(ctx, pipelinesqlc.PipelineStageTemplateByNameParams{ProjectID: projectID, Name: name})
	return stageModel(row), translate(err)
}
func (r Repository) CreatePipelineStageTemplate(ctx context.Context, item model.PipelineStage) error {
	return translate(r.q(ctx).InsertPipelineStageTemplate(ctx, templateStageParams(item)))
}
func (r Repository) UpdatePipelineStageTemplate(ctx context.Context, item model.PipelineStage) error {
	return translate(r.q(ctx).UpdatePipelineStageTemplate(ctx, pipelinesqlc.UpdatePipelineStageTemplateParams{Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: nullString(item.Artifacts), Version: nullInt(item.Version), UpdatedAt: time.Now().UTC(), ID: item.Id}))
}
func (r Repository) DeletePipelineStageTemplate(ctx context.Context, id string) error {
	return translate(r.q(ctx).DeletePipelineStageTemplate(ctx, id))
}
func (r Repository) TemplatePipelineStageReferences(ctx context.Context, pipelineID string) ([]model.PipelineStageReference, error) {
	rows, err := r.q(ctx).TemplatePipelineStageReferences(ctx, pipelineID)
	if err != nil {
		return nil, translate(err)
	}
	result := make([]model.PipelineStageReference, 0, len(rows))
	for _, row := range rows {
		result = append(result, referenceModel(row))
	}
	return result, nil
}
func (r Repository) ApplicationPipelineStages(ctx context.Context, pipelineID string) ([]model.PipelineStage, error) {
	rows, err := r.q(ctx).ApplicationPipelineStages(ctx, nullString(&pipelineID))
	if err != nil {
		return nil, translate(err)
	}
	result := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		result = append(result, stageModel(row))
	}
	return result, nil
}
func (r Repository) CreateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.CreatePipeline(txCtx, pipeline); err != nil {
			return err
		}
		for _, stage := range stages {
			if err := translate(r.q(txCtx).InsertApplicationPipelineStage(txCtx, applicationStageParams(stage))); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r Repository) UpdateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.UpdatePipeline(txCtx, pipeline); err != nil {
			return err
		}
		q := r.q(txCtx)
		if err := translate(q.DeleteApplicationPipelineStages(txCtx, nullString(&pipeline.Id))); err != nil {
			return err
		}
		for _, stage := range stages {
			if err := translate(q.InsertApplicationPipelineStage(txCtx, applicationStageParams(stage))); err != nil {
				return err
			}
		}
		return nil
	})
}
func (r Repository) UpdateTemplatePipelineWithReferences(ctx context.Context, pipeline model.Pipeline, references []model.PipelineStageReference) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.UpdatePipeline(txCtx, pipeline); err != nil {
			return err
		}
		q := r.q(txCtx)
		if err := translate(q.DeleteTemplatePipelineStageReferences(txCtx, pipeline.Id)); err != nil {
			return err
		}
		for _, reference := range references {
			if err := translate(q.InsertTemplatePipelineStageReference(txCtx, referenceParams(reference))); err != nil {
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
func templateStageParams(item model.PipelineStage) pipelinesqlc.InsertPipelineStageTemplateParams {
	return pipelinesqlc.InsertPipelineStageTemplateParams{ID: item.Id, ProjectID: item.ProjectId, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Version: nullInt(item.Version), Artifacts: nullString(item.Artifacts), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func applicationStageParams(item model.PipelineStage) pipelinesqlc.InsertApplicationPipelineStageParams {
	return pipelinesqlc.InsertApplicationPipelineStageParams{ID: item.Id, ProjectID: item.ProjectId, PipelineID: nullString(item.PipelineId), Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, SourceTemplateStageID: nullString(item.SourceTemplateStageId), SourceTemplateStageName: nullString(item.SourceTemplateStageName), SourceTemplateStageVersion: nullInt(item.SourceTemplateStageVersion), SourceTemplateStageDescription: nullString(item.SourceTemplateStageDescription), Artifacts: nullString(item.Artifacts), DependsOn: nullString(item.DependsOn), SortOrder: nullInt(item.SortOrder), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func referenceParams(item model.PipelineStageReference) pipelinesqlc.InsertTemplatePipelineStageReferenceParams {
	return pipelinesqlc.InsertTemplatePipelineStageReferenceParams{ID: item.Id, PipelineID: item.PipelineId, SourceTemplateStageID: item.SourceTemplateStageId, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageVersion: int64(item.SourceTemplateStageVersion), SourceTemplateStageDescription: item.SourceTemplateStageDescription, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: item.Artifacts, DependsOn: item.DependsOn, SortOrder: int64(item.SortOrder), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
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
	return model.PipelineStage{Id: item.ID, ProjectId: item.ProjectID, Kind: item.Kind, PipelineId: dbmodel.StringPtr(item.PipelineID), Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Version: dbmodel.IntPtrFromNullInt64(item.Version), SourceTemplateStageId: dbmodel.StringPtr(item.SourceTemplateStageID), SourceTemplateStageName: dbmodel.StringPtr(item.SourceTemplateStageName), SourceTemplateStageVersion: dbmodel.IntPtrFromNullInt64(item.SourceTemplateStageVersion), SourceTemplateStageDescription: dbmodel.StringPtr(item.SourceTemplateStageDescription), Artifacts: dbmodel.StringPtr(item.Artifacts), DependsOn: dbmodel.StringPtr(item.DependsOn), SortOrder: dbmodel.IntPtrFromNullInt64(item.SortOrder), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func referenceModel(item pipelinesqlc.PipelineStageReference) model.PipelineStageReference {
	return model.PipelineStageReference{Id: item.ID, PipelineId: item.PipelineID, SourceTemplateStageId: item.SourceTemplateStageID, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageVersion: int(item.SourceTemplateStageVersion), SourceTemplateStageDescription: item.SourceTemplateStageDescription, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: item.Artifacts, DependsOn: item.DependsOn, SortOrder: int(item.SortOrder), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
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
