package pipelinerepo

import (
	"context"
	"database/sql"
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

func (r Repository) Pipeline(ctx context.Context, projectId string, id string) (model.Pipeline, error) {
	item, err := r.q(ctx).PipelineById(ctx, pipelinesqlc.PipelineByIdParams{Id: id, ProjectId: nullString(&projectId)})
	return pipelineModel(item), translate(err)
}
func (r Repository) PipelineByName(ctx context.Context, projectId, kind, name string) (model.Pipeline, error) {
	item, err := r.q(ctx).PipelineByName(ctx, pipelinesqlc.PipelineByNameParams{ProjectId: nullString(&projectId), Kind: kind, Name: name})
	return pipelineModel(item), translate(err)
}
func (r Repository) ListPipelines(ctx context.Context, projectId, kind string, page, perPage int, search string) (repository.Page[model.Pipeline], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	searchPattern := sql.NullString{String: "%" + search + "%", Valid: search != ""}
	kindValue := sql.NullString{String: kind, Valid: kind != ""}
	projectIdArg := sql.NullString{String: projectId, Valid: true}
	args := pipelinesqlc.ListPipelinesParams{ProjectId: projectIdArg, Kind: kindValue, SearchPattern: searchPattern, Limit: int32(perPage), Offset: int32((page - 1) * perPage)}
	count, err := r.q(ctx).CountPipelines(ctx, pipelinesqlc.CountPipelinesParams{ProjectId: projectIdArg, Kind: kindValue, SearchPattern: searchPattern})
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
func (r Repository) UpdatePipeline(ctx context.Context, projectId string, item model.Pipeline) error {
	return translate(r.q(ctx).UpdatePipeline(ctx, pipelinesqlc.UpdatePipelineParams{Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int64(item.Version), ApplicationId: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionId: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), UpdatedAt: time.Now().UTC(), Id: item.Id, ProjectId: nullString(&projectId)}))
}
func (r Repository) DeletePipeline(ctx context.Context, projectId string, id string) error {
	return translate(r.q(ctx).DeletePipeline(ctx, pipelinesqlc.DeletePipelineParams{Id: id, ProjectId: nullString(&projectId)}))
}
func (r Repository) ListPipelineStageTemplates(ctx context.Context, projectId string, page, perPage int, search string) (repository.Page[model.PipelineStage], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	searchPattern := sql.NullString{String: "%" + search + "%", Valid: search != ""}
	args := pipelinesqlc.ListPipelineStageTemplatesParams{SearchPattern: searchPattern, Offset: int32((page - 1) * perPage), Limit: int32(perPage)}
	count, err := r.q(ctx).CountPipelineStageTemplates(ctx, pipelinesqlc.CountPipelineStageTemplatesParams{SearchPattern: searchPattern})
	if err != nil {
		return repository.Page[model.PipelineStage]{}, translate(err)
	}
	rows, err := r.q(ctx).ListPipelineStageTemplates(ctx, args)
	if err != nil {
		return repository.Page[model.PipelineStage]{}, translate(err)
	}
	items := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		items = append(items, stageTemplateListModel(row))
	}
	return repository.Page[model.PipelineStage]{Items: items, Total: int(count), Page: page, PerPage: perPage}, nil
}
func (r Repository) PipelineStageTemplate(ctx context.Context, projectId string, id string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageTemplateById(ctx, id)
	return stageTemplateByIdModel(row), translate(err)
}
func (r Repository) PipelineStageTemplateByName(ctx context.Context, projectId, name string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageTemplateByName(ctx, name)
	return stageTemplateByNameModel(row), translate(err)
}
func (r Repository) CreatePipelineStageTemplate(ctx context.Context, item model.PipelineStage) error {
	return translate(r.q(ctx).InsertPipelineStageTemplate(ctx, templateStageParams(item)))
}
func (r Repository) UpdatePipelineStageTemplate(ctx context.Context, projectId string, item model.PipelineStage) error {
	return translate(r.q(ctx).UpdatePipelineStageTemplate(ctx, pipelinesqlc.UpdatePipelineStageTemplateParams{Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: nullString(item.Artifacts), Version: nullInt(item.Version), UpdatedAt: time.Now().UTC(), Id: item.Id}))
}
func (r Repository) DeletePipelineStageTemplate(ctx context.Context, projectId string, id string) error {
	return translate(r.q(ctx).DeletePipelineStageTemplate(ctx, id))
}
func (r Repository) TemplatePipelineStageReferences(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStageReference, error) {
	rows, err := r.q(ctx).TemplatePipelineStageReferences(ctx, pipelineId)
	if err != nil {
		return nil, translate(err)
	}
	result := make([]model.PipelineStageReference, 0, len(rows))
	for _, row := range rows {
		result = append(result, referenceModel(row))
	}
	return result, nil
}
func (r Repository) ApplicationPipelineStages(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStage, error) {
	rows, err := r.q(ctx).ApplicationPipelineStages(ctx, pipelinesqlc.ApplicationPipelineStagesParams{PipelineId: nullString(&pipelineId), ProjectId: nullString(&projectId)})
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
	if err := r.CreatePipeline(ctx, pipeline); err != nil {
		return err
	}
	for _, stage := range stages {
		if err := translate(r.q(ctx).InsertApplicationPipelineStage(ctx, applicationStageParams(stage))); err != nil {
			return err
		}
	}
	return nil
}
func (r Repository) UpdateApplicationPipelineWithStages(ctx context.Context, projectId string, pipeline model.Pipeline, stages []model.PipelineStage) error {
	if err := r.UpdatePipeline(ctx, projectId, pipeline); err != nil {
		return err
	}
	q := r.q(ctx)
	if err := translate(q.DeleteApplicationPipelineStages(ctx, pipelinesqlc.DeleteApplicationPipelineStagesParams{PipelineId: nullString(&pipeline.Id), ProjectId: nullString(&projectId)})); err != nil {
		return err
	}
	for _, stage := range stages {
		if err := translate(q.InsertApplicationPipelineStage(ctx, applicationStageParams(stage))); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) UpdateTemplatePipelineWithReferences(ctx context.Context, projectId string, pipeline model.Pipeline, references []model.PipelineStageReference) error {
	if err := r.UpdatePipeline(ctx, projectId, pipeline); err != nil {
		return err
	}
	q := r.q(ctx)
	if err := translate(q.DeleteTemplatePipelineStageReferences(ctx, pipeline.Id)); err != nil {
		return err
	}
	for _, reference := range references {
		if err := translate(q.InsertTemplatePipelineStageReference(ctx, referenceParams(reference))); err != nil {
			return err
		}
	}
	return nil
}
func (r Repository) LatestPipelineSnapshot(ctx context.Context, projectId string, pipelineId string) (model.PipelineSnapshot, error) {
	item, err := r.q(ctx).LatestPipelineSnapshot(ctx, pipelinesqlc.LatestPipelineSnapshotParams{PipelineId: pipelineId, ProjectId: nullString(&projectId)})
	return snapshotModel(item), translate(err)
}
func (r Repository) PipelineSnapshot(ctx context.Context, projectId string, id string) (model.PipelineSnapshot, error) {
	item, err := r.q(ctx).PipelineSnapshotById(ctx, pipelinesqlc.PipelineSnapshotByIdParams{Id: id, ProjectId: nullString(&projectId)})
	return snapshotModel(item), translate(err)
}
func (r Repository) CreatePipelineSnapshot(ctx context.Context, item model.PipelineSnapshot) error {
	return translate(r.q(ctx).InsertPipelineSnapshot(ctx, pipelinesqlc.InsertPipelineSnapshotParams{Id: item.Id, ProjectId: nullString(item.ProjectId), PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int64(item.PipelineVersion), SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int64(item.SourceTemplateVersion), ApplicationId: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionId: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), StagesSnapshot: item.StagesSnapshot, VariablesSnapshot: item.VariablesSnapshot, CreatedAt: timeOrNow(item.CreatedAt)}))
}

func (r Repository) HasCDConfigurationReferences(ctx context.Context, projectId string) (bool, error) {
	count, err := r.q(ctx).CountCDConfigurationReferences(ctx, pipelinesqlc.CountCDConfigurationReferencesParams{ProjectId: nullString(&projectId)})
	return count > 0, translate(err)
}

func pipelineParams(item model.Pipeline) pipelinesqlc.CreatePipelineParams {
	return pipelinesqlc.CreatePipelineParams{Id: item.Id, ProjectId: nullString(item.ProjectId), Kind: item.Kind, SourcePipelineId: nullString(item.SourcePipelineId), SourceTemplateName: nullString(item.SourceTemplateName), SourceTemplateVersion: nullInt(item.SourceTemplateVersion), ApplicationId: nullString(item.ApplicationId), ApplicationName: nullString(item.ApplicationName), RepositoryId: nullString(item.RepositoryId), RepositoryName: nullString(item.RepositoryName), VersionForkStrategy: nullString(item.VersionForkStrategy), FixedVersionId: nullString(item.FixedVersionId), FixedVersionLabel: nullString(item.FixedVersionLabel), Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int64(item.Version), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func templateStageParams(item model.PipelineStage) pipelinesqlc.InsertPipelineStageTemplateParams {
	return pipelinesqlc.InsertPipelineStageTemplateParams{Id: item.Id, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Version: nullInt(item.Version), Artifacts: nullString(item.Artifacts), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func applicationStageParams(item model.PipelineStage) pipelinesqlc.InsertApplicationPipelineStageParams {
	return pipelinesqlc.InsertApplicationPipelineStageParams{Id: item.Id, ProjectId: nullString(&item.ProjectId), PipelineId: nullString(item.PipelineId), Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, SourceTemplateStageId: nullString(item.SourceTemplateStageId), SourceTemplateStageName: nullString(item.SourceTemplateStageName), SourceTemplateStageVersion: nullInt(item.SourceTemplateStageVersion), SourceTemplateStageDescription: nullString(item.SourceTemplateStageDescription), Artifacts: nullString(item.Artifacts), DependsOn: nullString(item.DependsOn), SortOrder: nullInt(item.SortOrder), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func referenceParams(item model.PipelineStageReference) pipelinesqlc.InsertTemplatePipelineStageReferenceParams {
	return pipelinesqlc.InsertTemplatePipelineStageReferenceParams{Id: item.Id, PipelineId: item.PipelineId, SourceTemplateStageId: item.SourceTemplateStageId, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageVersion: int64(item.SourceTemplateStageVersion), SourceTemplateStageDescription: item.SourceTemplateStageDescription, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: item.Artifacts, DependsOn: item.DependsOn, SortOrder: int64(item.SortOrder), CreatedAt: timeOrNow(item.CreatedAt), UpdatedAt: timeOrNow(item.UpdatedAt)}
}
func pipelineModel(item pipelinesqlc.Pipeline) model.Pipeline {
	result := model.Pipeline{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), Kind: item.Kind, SourcePipelineId: dbmodel.StringPtr(item.SourcePipelineId), SourceTemplateName: dbmodel.StringPtr(item.SourceTemplateName), ApplicationId: dbmodel.StringPtr(item.ApplicationId), ApplicationName: dbmodel.StringPtr(item.ApplicationName), RepositoryId: dbmodel.StringPtr(item.RepositoryId), RepositoryName: dbmodel.StringPtr(item.RepositoryName), VersionForkStrategy: dbmodel.StringPtr(item.VersionForkStrategy), FixedVersionId: dbmodel.StringPtr(item.FixedVersionId), FixedVersionLabel: dbmodel.StringPtr(item.FixedVersionLabel), Name: item.Name, Description: item.Description, VariableDeclarations: item.VariableDeclarations, Version: int(item.Version), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.SourceTemplateVersion.Valid {
		value := int(item.SourceTemplateVersion.Int64)
		result.SourceTemplateVersion = &value
	}
	return result
}
func stageTemplateListModel(item pipelinesqlc.ListPipelineStageTemplatesRow) model.PipelineStage {
	return stageModelFromValues(item.Id, item.ProjectId, item.Kind, item.PipelineId, item.Name, item.Image, item.Script, item.Description, item.Version, item.SourceTemplateStageId, item.SourceTemplateStageName, item.SourceTemplateStageVersion, item.SourceTemplateStageDescription, item.Artifacts, item.DependsOn, item.SortOrder, item.CreatedAt, item.UpdatedAt)
}
func stageTemplateByIdModel(item pipelinesqlc.PipelineStageTemplateByIdRow) model.PipelineStage {
	return stageModelFromValues(item.Id, item.ProjectId, item.Kind, item.PipelineId, item.Name, item.Image, item.Script, item.Description, item.Version, item.SourceTemplateStageId, item.SourceTemplateStageName, item.SourceTemplateStageVersion, item.SourceTemplateStageDescription, item.Artifacts, item.DependsOn, item.SortOrder, item.CreatedAt, item.UpdatedAt)
}
func stageTemplateByNameModel(item pipelinesqlc.PipelineStageTemplateByNameRow) model.PipelineStage {
	return stageModelFromValues(item.Id, item.ProjectId, item.Kind, item.PipelineId, item.Name, item.Image, item.Script, item.Description, item.Version, item.SourceTemplateStageId, item.SourceTemplateStageName, item.SourceTemplateStageVersion, item.SourceTemplateStageDescription, item.Artifacts, item.DependsOn, item.SortOrder, item.CreatedAt, item.UpdatedAt)
}
func stageModel(item pipelinesqlc.PipelineStage) model.PipelineStage {
	return stageModelFromValues(item.Id, item.ProjectId.String, item.Kind, item.PipelineId, item.Name, item.Image, item.Script, item.Description, item.Version, item.SourceTemplateStageId, item.SourceTemplateStageName, item.SourceTemplateStageVersion, item.SourceTemplateStageDescription, item.Artifacts, item.DependsOn, item.SortOrder, item.CreatedAt, item.UpdatedAt)
}
func stageModelFromValues(id, projectId, kind string, pipelineId sql.NullString, name, image, script, description string, version sql.NullInt64, sourceTemplateStageId, sourceTemplateStageName sql.NullString, sourceTemplateStageVersion sql.NullInt64, sourceTemplateStageDescription, artifacts, dependsOn sql.NullString, sortOrder sql.NullInt64, createdAt, updatedAt time.Time) model.PipelineStage {
	return model.PipelineStage{Id: id, ProjectId: projectId, Kind: kind, PipelineId: dbmodel.StringPtr(pipelineId), Name: name, Image: image, Script: script, Description: description, Version: dbmodel.IntPtrFromNullInt64(version), SourceTemplateStageId: dbmodel.StringPtr(sourceTemplateStageId), SourceTemplateStageName: dbmodel.StringPtr(sourceTemplateStageName), SourceTemplateStageVersion: dbmodel.IntPtrFromNullInt64(sourceTemplateStageVersion), SourceTemplateStageDescription: dbmodel.StringPtr(sourceTemplateStageDescription), Artifacts: dbmodel.StringPtr(artifacts), DependsOn: dbmodel.StringPtr(dependsOn), SortOrder: dbmodel.IntPtrFromNullInt64(sortOrder), CreatedAt: createdAt, UpdatedAt: updatedAt}
}
func referenceModel(item pipelinesqlc.PipelineStageReference) model.PipelineStageReference {
	return model.PipelineStageReference{Id: item.Id, PipelineId: item.PipelineId, SourceTemplateStageId: item.SourceTemplateStageId, SourceTemplateStageName: item.SourceTemplateStageName, SourceTemplateStageVersion: int(item.SourceTemplateStageVersion), SourceTemplateStageDescription: item.SourceTemplateStageDescription, Name: item.Name, Image: item.Image, Script: item.Script, Description: item.Description, Artifacts: item.Artifacts, DependsOn: item.DependsOn, SortOrder: int(item.SortOrder), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func snapshotModel(item pipelinesqlc.PipelineSnapshot) model.PipelineSnapshot {
	return model.PipelineSnapshot{Id: item.Id, ProjectId: dbmodel.StringPtr(item.ProjectId), PipelineId: item.PipelineId, PipelineName: item.PipelineName, PipelineVersion: int(item.PipelineVersion), SourcePipelineId: item.SourcePipelineId, SourceTemplateName: item.SourceTemplateName, SourceTemplateVersion: int(item.SourceTemplateVersion), ApplicationId: dbmodel.StringPtr(item.ApplicationId), ApplicationName: dbmodel.StringPtr(item.ApplicationName), RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, VersionForkStrategy: dbmodel.StringPtr(item.VersionForkStrategy), FixedVersionId: dbmodel.StringPtr(item.FixedVersionId), FixedVersionLabel: dbmodel.StringPtr(item.FixedVersionLabel), StagesSnapshot: item.StagesSnapshot, VariablesSnapshot: item.VariablesSnapshot, CreatedAt: item.CreatedAt}
}
func nullString(value *string) sql.NullString {
	if value == nil || *value == "" {
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
