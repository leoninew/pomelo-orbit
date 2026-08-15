package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type Service struct {
	store  stores
	logger *slog.Logger
}

type stores struct {
	project     repository.ProjectReader
	pipeline    repository.PipelineStore
	application repository.ApplicationStore
	repository  repository.RepositoryStore
}

func New(project repository.ProjectReader, pipeline repository.PipelineStore, application repository.ApplicationStore, repositories repository.RepositoryStore, logger *slog.Logger) Service {
	return Service{store: stores{project: project, pipeline: pipeline, application: application, repository: repositories}, logger: logger}
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}

func (s stores) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}

func (s stores) Pipeline(ctx context.Context, id string) (model.Pipeline, error) {
	return s.pipeline.Pipeline(ctx, id)
}

func (s stores) PipelineByName(ctx context.Context, projectId string, name string) (model.Pipeline, error) {
	return s.pipeline.PipelineByName(ctx, projectId, name)
}

func (s stores) ListPipelines(ctx context.Context, projectId string, kind string, page int, perPage int, search string) (repository.Page[model.Pipeline], error) {
	return s.pipeline.ListPipelines(ctx, projectId, kind, page, perPage, search)
}

func (s stores) CreatePipeline(ctx context.Context, pipeline model.Pipeline) error {
	return s.pipeline.CreatePipeline(ctx, pipeline)
}

func (s stores) UpdatePipeline(ctx context.Context, pipeline model.Pipeline) error {
	return s.pipeline.UpdatePipeline(ctx, pipeline)
}

func (s stores) DeletePipeline(ctx context.Context, id string) error {
	return s.pipeline.DeletePipeline(ctx, id)
}

func (s stores) ListPipelineStageTemplates(ctx context.Context, projectId string, page, perPage int, search string) (repository.Page[model.PipelineStage], error) {
	return s.pipeline.ListPipelineStageTemplates(ctx, projectId, page, perPage, search)
}

func (s stores) PipelineStageTemplate(ctx context.Context, id string) (model.PipelineStage, error) {
	return s.pipeline.PipelineStageTemplate(ctx, id)
}

func (s stores) PipelineStageTemplateByName(ctx context.Context, projectId, name string) (model.PipelineStage, error) {
	return s.pipeline.PipelineStageTemplateByName(ctx, projectId, name)
}

func (s stores) CreatePipelineStageTemplate(ctx context.Context, stage model.PipelineStage) error {
	return s.pipeline.CreatePipelineStageTemplate(ctx, stage)
}

func (s stores) UpdatePipelineStageTemplate(ctx context.Context, stage model.PipelineStage) error {
	return s.pipeline.UpdatePipelineStageTemplate(ctx, stage)
}

func (s stores) DeletePipelineStageTemplate(ctx context.Context, id string) error {
	return s.pipeline.DeletePipelineStageTemplate(ctx, id)
}

func (s stores) TemplatePipelineStageReferences(ctx context.Context, pipelineId string) ([]model.PipelineStageReference, error) {
	return s.pipeline.TemplatePipelineStageReferences(ctx, pipelineId)
}

func (s stores) ApplicationPipelineStages(ctx context.Context, pipelineId string) ([]model.PipelineStage, error) {
	return s.pipeline.ApplicationPipelineStages(ctx, pipelineId)
}

func (s stores) UpdateTemplatePipelineWithReferences(ctx context.Context, pipeline model.Pipeline, references []model.PipelineStageReference) error {
	return s.pipeline.UpdateTemplatePipelineWithReferences(ctx, pipeline, references)
}

func (s stores) UpdateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return s.pipeline.UpdateApplicationPipelineWithStages(ctx, pipeline, stages)
}

func (s stores) CreateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	return s.pipeline.CreateApplicationPipelineWithStages(ctx, pipeline, stages)
}

func (s stores) LatestPipelineSnapshot(ctx context.Context, pipelineId string) (model.PipelineSnapshot, error) {
	return s.pipeline.LatestPipelineSnapshot(ctx, pipelineId)
}

func (s stores) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return s.pipeline.PipelineSnapshot(ctx, id)
}

func (s stores) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	return s.pipeline.CreatePipelineSnapshot(ctx, snapshot)
}

func (s stores) Application(ctx context.Context, id string) (model.Application, error) {
	return s.application.Application(ctx, id)
}

func (s stores) Version(ctx context.Context, id string) (model.Version, error) {
	return s.application.Version(ctx, id)
}

func (s stores) VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, versionId)
}

func (s stores) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, id)
}

func (s Service) ListPipelines(ctx context.Context, userId string, projectId string, kind string, page int, perPage int, search string) (repository.Page[pipelinedto.PipelineDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[pipelinedto.PipelineDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[pipelinedto.PipelineDetail]{}, err
	}
	if kind != "" && kind != model.PipelineKindTemplate && kind != model.PipelineKindApplication {
		return repository.Page[pipelinedto.PipelineDetail]{}, apperror.New(apperror.KindValidation, "kind must be template or application")
	}
	pageOfPipelines, err := s.store.ListPipelines(ctx, projectId, kind, page, perPage, search)
	if err != nil {
		return repository.Page[pipelinedto.PipelineDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipelines", err)
	}
	items := make([]pipelinedto.PipelineDetail, 0, len(pageOfPipelines.Items))
	for _, pipeline := range pageOfPipelines.Items {
		detail, err := s.pipelineDetail(ctx, pipeline)
		if err != nil {
			return repository.Page[pipelinedto.PipelineDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[pipelinedto.PipelineDetail]{Items: items, Total: pageOfPipelines.Total, Page: pageOfPipelines.Page, PerPage: pageOfPipelines.PerPage}, nil
}

// CreatePipeline only creates a template. Application pipelines are created
// by InstantiatePipeline so their immutable identity is always complete.
func (s Service) CreatePipeline(ctx context.Context, userId string, input pipelinedto.PipelineCreateInput) (pipelinedto.PipelineDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if strings.TrimSpace(input.Kind) != model.PipelineKindTemplate {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "application pipelines must be instantiated from a template")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "name is required")
	}
	if err := s.ensurePipelineNameAvailable(ctx, projectId, name, ""); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	variables, err := marshalPipelineVariables(pipelinevariable.SanitizeTemplatePipelineVariables(input.VariableDeclarations))
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	pipeline := model.Pipeline{Id: idutil.NewId(), ProjectId: &projectId, Kind: model.PipelineKindTemplate, Name: name, Description: input.Description, VariableDeclarations: variables, Version: 1}
	if err := s.store.CreatePipeline(ctx, pipeline); err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline", err)
	}
	return s.pipelineDetail(ctx, pipeline)
}

func (s Service) PipelineForUser(ctx context.Context, userId string, pipelineId string) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	return s.pipelineDetail(ctx, pipeline)
}

func (s Service) UpdatePipeline(ctx context.Context, userId string, pipelineId string, input pipelinedto.PipelineUpdateInput) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	changed := false
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "name is required")
		}
		if err := s.ensurePipelineNameAvailable(ctx, pipelineProjectId(pipeline), name, pipeline.Id); err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if name != pipeline.Name {
			pipeline.Name = name
			changed = true
		}
	}
	if input.Description != nil && *input.Description != pipeline.Description {
		pipeline.Description = *input.Description
		changed = true
	}
	if input.VariableDeclarations != nil {
		variablesToStore := pipelinevariable.SanitizeTemplatePipelineVariables(*input.VariableDeclarations)
		if pipeline.Kind == model.PipelineKindApplication {
			var err error
			variablesToStore, err = pipelinevariable.NormalizePipelineVariables(*input.VariableDeclarations)
			if err != nil {
				return pipelinedto.PipelineDetail{}, err
			}
		}
		variables, err := marshalPipelineVariables(variablesToStore)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if variables != pipeline.VariableDeclarations {
			pipeline.VariableDeclarations = variables
			changed = true
		}
	}
	if input.ApplicationId != nil {
		bound, err := s.bindApplicationIfUnbound(ctx, &pipeline, *input.ApplicationId)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if bound {
			changed = true
		}
	}
	if err := s.validateStoredPipelineConfiguration(ctx, pipeline); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if changed {
		pipeline.Version++
		if err := s.store.UpdatePipeline(ctx, pipeline); err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline", err)
		}
	}
	return s.pipelineDetail(ctx, pipeline)
}

// bindApplicationIfUnbound sets application identity only when the pipeline has none.
// Rebinding or clearing an existing application is rejected.
func (s Service) bindApplicationIfUnbound(ctx context.Context, pipeline *model.Pipeline, applicationID string) (bool, error) {
	if pipeline.Kind != model.PipelineKindApplication {
		return false, apperror.New(apperror.KindValidation, "only application pipelines can bind an application")
	}
	id := strings.TrimSpace(applicationID)
	if id == "" {
		return false, apperror.New(apperror.KindValidation, "application_id cannot be empty")
	}
	if pipeline.ApplicationId != nil {
		if strings.TrimSpace(*pipeline.ApplicationId) == id {
			return false, nil
		}
		return false, apperror.New(apperror.KindValidation, "application binding cannot be changed once set")
	}
	application, err := s.applicationInProject(ctx, id, pipelineProjectId(*pipeline))
	if err != nil {
		return false, err
	}
	pipeline.ApplicationId = &application.Id
	pipeline.ApplicationName = &application.Name
	return true, nil
}

func (s Service) DeletePipeline(ctx context.Context, userId string, pipelineId string) error {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return err
	}
	if err := s.store.DeletePipeline(ctx, pipeline.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline", err)
	}
	return nil
}

func (s Service) InstantiatePipeline(ctx context.Context, userId string, templateId string, input pipelinedto.PipelineInstantiateInput) (pipelinedto.PipelineDetail, error) {
	template, err := s.loadPipelineForUser(ctx, userId, templateId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if template.Kind != model.PipelineKindTemplate {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "only template pipelines can be instantiated")
	}
	projectId := pipelineProjectId(template)
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "name is required")
	}
	if err := s.ensurePipelineNameAvailable(ctx, projectId, name, ""); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	repo, err := s.repositoryInProject(ctx, strings.TrimSpace(input.RepositoryId), projectId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	var applicationId, applicationName *string
	if input.ApplicationId != nil {
		id := strings.TrimSpace(*input.ApplicationId)
		if id == "" {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "application_id cannot be empty")
		}
		application, err := s.applicationInProject(ctx, id, projectId)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		applicationId, applicationName = &application.Id, &application.Name
	}
	references, err := s.store.TemplatePipelineStageReferences(ctx, template.Id)
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load template pipeline stage references", err)
	}
	if err := validateTemplatePipelineConfiguration(template, references); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	templateIdCopy, templateName, templateVersion := template.Id, template.Name, template.Version
	repositoryId, repositoryName := repo.Id, repo.Name
	pipeline := model.Pipeline{
		Id: idutil.NewId(), ProjectId: template.ProjectId, Kind: model.PipelineKindApplication,
		SourcePipelineId: &templateIdCopy, SourceTemplateName: &templateName, SourceTemplateVersion: &templateVersion,
		ApplicationId: applicationId, ApplicationName: applicationName, RepositoryId: &repositoryId, RepositoryName: &repositoryName,
		Name: name, Description: template.Description, VariableDeclarations: template.VariableDeclarations, Version: 1,
	}
	stages, err := clonePipelineStageReferences(references, pipeline.Id, projectId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if len(input.ArtifactBindings) > 0 && applicationId == nil {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "artifact bindings require an application")
	}
	if err := applyPipelineArtifactBindings(stages, references, input.ArtifactBindings); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if _, err := s.applyVersionForkStrategy(ctx, &pipeline, input.VersionForkStrategy, input.FixedVersionId, false); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if err := s.validatePipelineConfiguration(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if err := s.store.CreateApplicationPipelineWithStages(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to instantiate pipeline", err)
	}
	return s.pipelineDetail(ctx, pipeline)
}

func applyPipelineArtifactBindings(stages []model.PipelineStage, references []model.PipelineStageReference, bindings []pipelinedto.PipelineArtifactBinding) error {
	if len(bindings) == 0 {
		return nil
	}
	if len(stages) != len(references) {
		return apperror.New(apperror.KindInternal, "pipeline stage references could not be materialized")
	}
	stageIndexByReferenceID := make(map[string]int, len(references))
	for index, reference := range references {
		stageIndexByReferenceID[reference.Id] = index
	}
	seen := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		stageID, artifactName := strings.TrimSpace(binding.StageId), strings.TrimSpace(binding.ArtifactName)
		componentName, err := optionalComponentName(stageTemplateStringPointer(binding.ComponentName))
		if err != nil || componentName == nil || stageID == "" || artifactName == "" {
			return apperror.New(apperror.KindValidation, "invalid pipeline artifact binding")
		}
		key := stageID + "\x00" + artifactName
		if _, exists := seen[key]; exists {
			return apperror.New(apperror.KindValidation, "pipeline artifact may only be bound once")
		}
		seen[key] = struct{}{}
		index, exists := stageIndexByReferenceID[stageID]
		if !exists {
			return apperror.New(apperror.KindValidation, "pipeline artifact binding stage does not exist")
		}
		artifacts, err := pipelineStageArtifacts(stages[index])
		if err != nil {
			return err
		}
		found := false
		for artifactIndex := range artifacts {
			if artifacts[artifactIndex].Name == artifactName {
				if artifacts[artifactIndex].Collector != "docker_image" {
					return apperror.New(apperror.KindValidation, "only docker_image artifacts can bind a component")
				}
				artifacts[artifactIndex].ComponentName = componentName
				found = true
				break
			}
		}
		if !found {
			return apperror.New(apperror.KindValidation, "pipeline artifact binding artifact does not exist")
		}
		items := make([]pipelinedto.ArtifactConfig, 0, len(artifacts))
		for _, artifact := range artifacts {
			items = append(items, pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
		}
		data, err := marshalPipelineStageArtifacts(items)
		if err != nil {
			return err
		}
		if data == nil {
			return apperror.New(apperror.KindValidation, "application pipeline stage artifacts are required")
		}
		stages[index].Artifacts = data
	}
	return nil
}

func (s Service) PipelineSnapshotForUser(ctx context.Context, userId string, snapshotId string) (pipelinedto.PipelineSnapshotDetail, error) {
	snapshot, err := s.store.PipelineSnapshot(ctx, strings.TrimSpace(snapshotId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinedto.PipelineSnapshotDetail{}, apperror.New(apperror.KindNotFound, "Pipeline snapshot "+snapshotId+" not found")
		}
		return pipelinedto.PipelineSnapshotDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	projectId, err := requiredProjectID(snapshot.ProjectId, "Pipeline snapshot")
	if err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	stages, err := snapshotStages(snapshot.StagesSnapshot)
	if err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	variables, err := snapshotVariables(snapshot.VariablesSnapshot)
	if err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	return pipelinedto.PipelineSnapshotDetail{Snapshot: snapshot, StagesSnapshot: stages, VariablesSnapshot: variables}, nil
}

func (s Service) loadPipelineForUser(ctx context.Context, userId string, pipelineId string) (model.Pipeline, error) {
	pipelineId = strings.TrimSpace(pipelineId)
	pipeline, err := s.store.Pipeline(ctx, pipelineId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Pipeline{}, apperror.New(apperror.KindNotFound, "Pipeline "+pipelineId+" not found")
		}
		return model.Pipeline{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline", err)
	}
	projectId, err := requiredProjectID(pipeline.ProjectId, "Pipeline")
	if err != nil {
		return model.Pipeline{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Pipeline{}, err
	}
	return pipeline, nil
}

func (s Service) pipelineDetail(ctx context.Context, pipeline model.Pipeline) (pipelinedto.PipelineDetail, error) {
	nodes, stageViews, err := s.pipelineStageNodes(ctx, pipeline)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	variables, err := pipelinevariable.PipelineVariables(pipeline.VariableDeclarations)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		return pipelinedto.PipelineDetail{Pipeline: pipeline, StageNodes: nodes, VariableDeclarations: pipelinevariable.ResolveTemplatePipelineVariables(stageViews, variables)}, nil
	}
	managedVariables, err := pipelinevariable.ResolvePipelineVariables(stageViews, variables)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	return pipelinedto.PipelineDetail{Pipeline: pipeline, StageNodes: nodes, VariableDeclarations: managedVariables}, nil
}

// applyVersionForkStrategy changes the version selection captured while an
// application pipeline is instantiated.
func (s Service) applyVersionForkStrategy(ctx context.Context, pipeline *model.Pipeline, strategyInput *string, fixedVersionInput *string, clear bool) (bool, error) {
	if strategyInput == nil && fixedVersionInput == nil && !clear {
		return false, nil
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		return false, apperror.New(apperror.KindValidation, "template pipelines cannot configure a version strategy")
	}
	if clear {
		if strategyInput != nil || fixedVersionInput != nil {
			return false, apperror.New(apperror.KindValidation, "cannot set and clear version strategy together")
		}
		if pipeline.VersionForkStrategy == nil && pipeline.FixedVersionId == nil {
			return false, nil
		}
		pipeline.VersionForkStrategy, pipeline.FixedVersionId, pipeline.FixedVersionLabel = nil, nil, nil
		return true, nil
	}

	strategy := pipeline.VersionForkStrategy
	if strategyInput != nil {
		value := strings.TrimSpace(*strategyInput)
		strategy = &value
	}
	fixed := pipeline.FixedVersionId
	if fixedVersionInput != nil {
		value := strings.TrimSpace(*fixedVersionInput)
		if value == "" {
			fixed = nil
		} else {
			fixed = &value
		}
	}
	changed := !stringPointerEqual(pipeline.VersionForkStrategy, strategy) || !stringPointerEqual(pipeline.FixedVersionId, fixed)
	pipeline.VersionForkStrategy, pipeline.FixedVersionId = strategy, fixed
	if err := s.populateFixedVersionLabel(ctx, pipeline); err != nil {
		return false, err
	}
	return changed, nil
}

func stringPointerEqual(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func (s Service) ensurePipelineNameAvailable(ctx context.Context, projectId string, name string, currentID string) error {
	existing, err := s.store.PipelineByName(ctx, projectId, name)
	if err == nil && existing.Id != currentID {
		return apperror.New(apperror.KindConflict, "Pipeline '"+name+"' already exists")
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline name", err)
	}
	return nil
}

func (s Service) applicationInProject(ctx context.Context, applicationId string, projectId string) (model.Application, error) {
	app, err := s.store.Application(ctx, applicationId)
	if errors.Is(err, repository.ErrNotFound) || app.ProjectId == nil || *app.ProjectId != projectId {
		return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
	}
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return app, nil
}

func (s Service) repositoryInProject(ctx context.Context, repositoryId string, projectId string) (model.Repository, error) {
	repo, err := s.store.Repository(ctx, repositoryId)
	if errors.Is(err, repository.ErrNotFound) || repo.ProjectId == nil || *repo.ProjectId != projectId {
		return model.Repository{}, apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
	}
	if err != nil {
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return repo, nil
}

func (s Service) populateFixedVersionLabel(ctx context.Context, pipeline *model.Pipeline) error {
	if pipeline.VersionForkStrategy == nil || *pipeline.VersionForkStrategy != model.VersionForkStrategyFixed || pipeline.FixedVersionId == nil {
		pipeline.FixedVersionLabel = nil
		return nil
	}
	version, err := s.store.Version(ctx, *pipeline.FixedVersionId)
	if errors.Is(err, repository.ErrNotFound) {
		return apperror.New(apperror.KindNotFound, "Fixed version not found")
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load fixed version", err)
	}
	if pipeline.ApplicationId == nil || version.ApplicationId != *pipeline.ApplicationId {
		return apperror.New(apperror.KindValidation, "fixed_version_id must belong to pipeline application")
	}
	label := version.Label
	pipeline.FixedVersionLabel = &label
	return nil
}

func pipelineProjectId(pipeline model.Pipeline) string {
	if pipeline.ProjectId == nil {
		return ""
	}
	return *pipeline.ProjectId
}

func marshalPipelineVariables(variables []map[string]any) (string, error) {
	if variables == nil {
		variables = []map[string]any{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variable_declarations", err)
	}
	return string(data), nil
}

func marshalPipelineStageArtifacts(input []pipelinedto.ArtifactConfig) (*string, error) {
	artifacts := make([]model.ArtifactConfig, 0, len(input))
	for _, artifact := range input {
		componentName, err := optionalComponentName(artifact.ComponentName)
		if err != nil {
			return nil, err
		}
		artifact = pipelinedto.ArtifactConfig{Name: strings.TrimSpace(artifact.Name), Collector: strings.TrimSpace(artifact.Collector), Reference: strings.TrimSpace(artifact.Reference), Command: strings.TrimSpace(artifact.Command), Format: strings.TrimSpace(artifact.Format), ComponentName: componentName}
		if err := validateArtifact(artifact); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, model.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
	}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		if _, exists := seen[artifact.Name]; exists {
			return nil, apperror.New(apperror.KindValidation, "Pipeline stage artifact names must be unique")
		}
		seen[artifact.Name] = struct{}{}
	}
	if len(artifacts) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(artifacts)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
	}
	value := string(data)
	return &value, nil
}

func validateArtifact(artifact pipelinedto.ArtifactConfig) error {
	if artifact.Name == "" || artifact.Collector == "" {
		return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
	}
	switch artifact.Collector {
	case "file":
		if artifact.Reference == "" || artifact.Command != "" || artifact.Format != "" || artifact.ComponentName != nil {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
		}
	case "command":
		if artifact.Reference != "" || artifact.Command == "" || (artifact.Format != "text" && artifact.Format != "git_object_id") || artifact.ComponentName != nil {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
		}
	case "docker_image":
		if artifact.Reference == "" || artifact.Command != "" || artifact.Format != "" {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
		}
	default:
		return apperror.New(apperror.KindValidation, "Unknown artifact collector")
	}
	return nil
}

func optionalComponentName(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	name := strings.TrimSpace(*value)
	if name == "" {
		return nil, nil
	}
	for index, runeValue := range name {
		if (runeValue < 'a' || runeValue > 'z') && (runeValue < '0' || runeValue > '9') && runeValue != '-' || (index == 0 && (runeValue < 'a' || runeValue > 'z')) {
			return nil, apperror.New(apperror.KindValidation, "Invalid component_name")
		}
	}
	return &name, nil
}

func marshalDependsOn(dependsOn []string) (string, error) {
	if dependsOn == nil {
		dependsOn = []string{}
	}
	data, err := json.Marshal(dependsOn)
	if err != nil {
		return "", apperror.New(apperror.KindValidation, "Invalid pipeline stage dependencies")
	}
	return string(data), nil
}

func pipelineStageArtifacts(stage model.PipelineStage) ([]model.ArtifactConfig, error) {
	if stage.Artifacts == nil {
		return nil, apperror.New(apperror.KindValidation, "application pipeline stage artifacts are required")
	}
	if err := validateJSONArray(*stage.Artifacts, "artifacts"); err != nil {
		return nil, err
	}
	var artifacts []model.ArtifactConfig
	if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
	}
	seen := map[string]struct{}{}
	for _, artifact := range artifacts {
		input := pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName}
		if err := validateArtifact(input); err != nil {
			return nil, err
		}
		if _, exists := seen[artifact.Name]; exists {
			return nil, apperror.New(apperror.KindValidation, "Pipeline stage artifact names must be unique")
		}
		seen[artifact.Name] = struct{}{}
	}
	return artifacts, nil
}

func stageDependsOn(stage model.PipelineStage) ([]string, error) {
	if stage.DependsOn == nil {
		return nil, apperror.New(apperror.KindValidation, "application pipeline stage dependencies are required")
	}
	if err := validateJSONArray(*stage.DependsOn, "depends_on"); err != nil {
		return nil, err
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(*stage.DependsOn), &dependsOn); err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage dependencies")
	}
	return dependsOn, nil
}

func snapshotStages(value string) ([]model.StageDefinition, error) {
	if strings.TrimSpace(value) == "" {
		return []model.StageDefinition{}, nil
	}
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	return stages, nil
}

func snapshotVariables(value string) ([]model.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []model.VariableDeclaration{}, nil
	}
	var variables []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot variables")
	}
	return variables, nil
}

func (s Service) validatePipelineConfiguration(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error {
	definitions, err := pipelineStageDefinitions(stages)
	if err != nil {
		return err
	}
	if err := validatePipelineDAG(definitions); err != nil {
		return apperror.New(apperror.KindValidation, err.Error())
	}
	mappings := componentMappings(definitions)
	if pipeline.Kind == model.PipelineKindTemplate {
		if pipeline.ApplicationId != nil || pipeline.RepositoryId != nil || pipeline.SourcePipelineId != nil || pipeline.VersionForkStrategy != nil || pipeline.FixedVersionId != nil {
			return apperror.New(apperror.KindValidation, "template pipeline cannot have application bindings")
		}
		if len(mappings) != 0 {
			return apperror.New(apperror.KindValidation, "template artifact component_name must be empty")
		}
		return nil
	}
	if pipeline.Kind != model.PipelineKindApplication || pipeline.RepositoryId == nil || pipeline.RepositoryName == nil || pipeline.SourcePipelineId == nil || pipeline.SourceTemplateName == nil || pipeline.SourceTemplateVersion == nil {
		return apperror.New(apperror.KindValidation, "application pipeline requires template and repository bindings")
	}
	if (pipeline.ApplicationId == nil) != (pipeline.ApplicationName == nil) {
		return apperror.New(apperror.KindValidation, "application pipeline application binding is incomplete")
	}
	if len(mappings) == 0 {
		if pipeline.VersionForkStrategy != nil || pipeline.FixedVersionId != nil {
			return apperror.New(apperror.KindValidation, "version strategy requires a component-bound image artifact")
		}
		return nil
	}
	if pipeline.ApplicationId == nil {
		return apperror.New(apperror.KindValidation, "component-bound image artifacts require an application binding")
	}
	if pipeline.VersionForkStrategy == nil {
		return apperror.New(apperror.KindValidation, "component-bound image artifacts require a version strategy")
	}
	switch *pipeline.VersionForkStrategy {
	case model.VersionForkStrategyLatest:
		if pipeline.FixedVersionId != nil {
			return apperror.New(apperror.KindValidation, "latest version strategy cannot set fixed_version_id")
		}
	case model.VersionForkStrategyFixed:
		if pipeline.FixedVersionId == nil || strings.TrimSpace(*pipeline.FixedVersionId) == "" {
			return apperror.New(apperror.KindValidation, "fixed version strategy requires fixed_version_id")
		}
		if err := s.populateFixedVersionLabel(ctx, &pipeline); err != nil {
			return err
		}
	default:
		return apperror.New(apperror.KindValidation, "version_fork_strategy must be latest or fixed")
	}
	seen := map[string]struct{}{}
	for _, mapping := range mappings {
		if _, exists := seen[mapping.ComponentName]; exists {
			return apperror.New(apperror.KindValidation, "component_name may only be mapped by one image artifact")
		}
		seen[mapping.ComponentName] = struct{}{}
		if _, err := sourceCommitArtifactForStage(mapping.Stage, definitions); err != nil {
			return apperror.New(apperror.KindValidation, fmt.Sprintf("stage %s artifact %s: %v", mapping.Stage.Name, mapping.Artifact.Name, err))
		}
	}
	if pipeline.FixedVersionId != nil {
		components, err := s.store.VersionComponentsByVersion(ctx, *pipeline.FixedVersionId)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to load fixed version components", err)
		}
		for componentName := range seen {
			if !versionHasComponent(components, componentName) {
				return apperror.New(apperror.KindValidation, "fixed version does not contain component "+componentName)
			}
		}
	}
	return nil
}

type componentMapping struct {
	Stage         model.StageDefinition
	Artifact      model.ArtifactConfig
	ComponentName string
}

func componentMappings(stages []model.StageDefinition) []componentMapping {
	result := make([]componentMapping, 0)
	for _, stage := range stages {
		for _, artifact := range stage.Artifacts {
			if artifact.Collector == "docker_image" && artifact.ComponentName != nil {
				result = append(result, componentMapping{Stage: stage, Artifact: artifact, ComponentName: *artifact.ComponentName})
			}
		}
	}
	return result
}

func componentMappingsForStages(stages []model.PipelineStage) ([]componentMapping, error) {
	definitions, err := pipelineStageDefinitions(stages)
	if err != nil {
		return nil, err
	}
	return componentMappings(definitions), nil
}

func pipelineStageDefinitions(stages []model.PipelineStage) ([]model.StageDefinition, error) {
	definitions := make([]model.StageDefinition, 0, len(stages))
	for _, stage := range stages {
		if err := validateApplicationPipelineStage(stage); err != nil {
			return nil, err
		}
		artifacts, err := pipelineStageArtifacts(stage)
		if err != nil {
			return nil, err
		}
		dependsOn, err := stageDependsOn(stage)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, model.StageDefinition{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Artifacts: artifacts, DependsOn: dependsOn, SortOrder: *stage.SortOrder, Description: stage.Description, SourceTemplateStageId: *stage.SourceTemplateStageId, SourceTemplateStageName: *stage.SourceTemplateStageName, SourceTemplateStageVersion: *stage.SourceTemplateStageVersion})
	}
	return definitions, nil
}

func validatePipelineStageTemplate(stage model.PipelineStage) error {
	if strings.TrimSpace(stage.Id) == "" || strings.TrimSpace(stage.ProjectId) == "" || stage.Kind != model.PipelineStageKindTemplate || strings.TrimSpace(stage.Name) == "" || strings.TrimSpace(stage.Image) == "" || stage.Version == nil || *stage.Version <= 0 {
		return apperror.New(apperror.KindValidation, "invalid pipeline stage template")
	}
	if stage.PipelineId != nil || stage.SourceTemplateStageId != nil || stage.SourceTemplateStageName != nil || stage.SourceTemplateStageVersion != nil || stage.SourceTemplateStageDescription != nil || stage.DependsOn != nil || stage.SortOrder != nil {
		return apperror.New(apperror.KindValidation, "pipeline stage templates cannot contain pipeline configuration")
	}
	_, err := templateStageArtifacts(stage)
	return err
}

func validateApplicationPipelineStage(stage model.PipelineStage) error {
	if strings.TrimSpace(stage.Id) == "" || strings.TrimSpace(stage.ProjectId) == "" || stage.Kind != model.PipelineStageKindApplication || stage.PipelineId == nil || strings.TrimSpace(*stage.PipelineId) == "" || strings.TrimSpace(stage.Name) == "" || strings.TrimSpace(stage.Image) == "" || stage.Version != nil || stage.Artifacts == nil || stage.DependsOn == nil || stage.SortOrder == nil || *stage.SortOrder < 0 {
		return apperror.New(apperror.KindValidation, "invalid application pipeline stage")
	}
	if stage.SourceTemplateStageId == nil || strings.TrimSpace(*stage.SourceTemplateStageId) == "" || stage.SourceTemplateStageName == nil || strings.TrimSpace(*stage.SourceTemplateStageName) == "" || stage.SourceTemplateStageVersion == nil || *stage.SourceTemplateStageVersion <= 0 || stage.SourceTemplateStageDescription == nil {
		return apperror.New(apperror.KindValidation, "application pipeline stage source snapshot is required")
	}
	return nil
}

func validateJSONArray(value, field string) error {
	var items []json.RawMessage
	if strings.TrimSpace(value) == "" || json.Unmarshal([]byte(value), &items) != nil || items == nil {
		return apperror.New(apperror.KindValidation, field+" must be a JSON array")
	}
	return nil
}

func validatePipelineDAG(stages []model.StageDefinition) error {
	byID := make(map[string]model.StageDefinition, len(stages))
	names := map[string]struct{}{}
	for _, stage := range stages {
		if stage.Id == "" || stage.Name == "" {
			return errors.New("pipeline stage id and name are required")
		}
		if _, exists := byID[stage.Id]; exists {
			return fmt.Errorf("duplicate pipeline stage %s", stage.Id)
		}
		if _, exists := names[stage.Name]; exists {
			return fmt.Errorf("duplicate pipeline stage name %s", stage.Name)
		}
		byID[stage.Id], names[stage.Name] = stage, struct{}{}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return errors.New("pipeline stage dependencies contain a cycle")
		}
		if visited[id] {
			return nil
		}
		stage, exists := byID[id]
		if !exists {
			return fmt.Errorf("pipeline stage dependency %s does not exist", id)
		}
		visiting[id] = true
		for _, dependency := range stage.DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[id], visited[id] = false, true
		return nil
	}
	for _, stage := range stages {
		if err := visit(stage.Id); err != nil {
			return err
		}
	}
	return nil
}

func sourceCommitArtifactForStage(stage model.StageDefinition, stages []model.StageDefinition) (model.ArtifactConfig, error) {
	byID := make(map[string]model.StageDefinition, len(stages))
	for _, item := range stages {
		byID[item.Id] = item
	}
	seen := map[string]bool{}
	candidates := make([]model.ArtifactConfig, 0, 1)
	var visit func(string) error
	visit = func(id string) error {
		if seen[id] {
			return nil
		}
		seen[id] = true
		current, exists := byID[id]
		if !exists {
			return fmt.Errorf("dependency %s does not exist", id)
		}
		for _, artifact := range current.Artifacts {
			if artifact.Collector == "command" && artifact.Format == "git_object_id" {
				candidates = append(candidates, artifact)
			}
		}
		for _, dependency := range current.DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	for _, dependency := range stage.DependsOn {
		if err := visit(dependency); err != nil {
			return model.ArtifactConfig{}, err
		}
	}
	if len(candidates) != 1 {
		return model.ArtifactConfig{}, fmt.Errorf("requires exactly one transitive git_object_id artifact, found %d", len(candidates))
	}
	return candidates[0], nil
}

func versionHasComponent(components []model.VersionComponent, name string) bool {
	for _, component := range components {
		if component.Name == name {
			return true
		}
	}
	return false
}
