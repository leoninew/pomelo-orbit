package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

type ArtifactConfig struct {
	Name          string
	Collector     string
	Reference     string
	Command       string
	Format        string
	ComponentName *string
}

type PipelineCreateInput struct {
	ProjectId            string
	Kind                 string
	Name                 string
	Description          string
	VariableDeclarations []map[string]any
}

type PipelineUpdateInput struct {
	Name                 *string
	Description          *string
	VariableDeclarations *[]map[string]any
	// ApplicationId is a first-time bind only: allowed when the pipeline has no
	// application yet; rejected when already bound or when clearing.
	ApplicationId *string
}

type PipelineInstantiateInput struct {
	Name                string
	ApplicationId       *string
	RepositoryId        string
	VersionForkStrategy *string
	FixedVersionId      *string
	ArtifactBindings    []PipelineArtifactBinding
}

type PipelineArtifactBinding struct {
	StageId       string
	ArtifactName  string
	ComponentName string
}

type PipelineStageTemplateCreateInput struct {
	ProjectId   string
	Name        string
	Image       string
	Script      string
	Description string
	Artifacts   []ArtifactConfig
}

type PipelineStageTemplateUpdateInput struct {
	Name        *string
	Image       *string
	Script      *string
	Description *string
	Artifacts   *[]ArtifactConfig
	// The remaining fields make invalid API payloads observable to the service.
	// They are rejected for a reusable template rather than silently discarded.
	DependsOn                *[]string
	SortOrder                *int
	PipelineId               *string
	VersionForkStrategy      *string
	FixedVersionId           *string
	ClearVersionForkStrategy bool
}

type PipelineStageTemplateDetail struct {
	Stage     model.PipelineStage
	Artifacts []ArtifactConfig
}

type PipelineStageImportInput struct {
	SourceTemplateStageId string
	Name                  *string
	Description           *string
	DependsOn             []string
	SortOrder             int
}

type PipelineStageNodeUpdateInput struct {
	Name        *string
	Image       *string
	Script      *string
	DependsOn   *[]string
	SortOrder   *int
	Description *string
}

type PipelineStageNodeDetail struct {
	Node                       model.PipelineStageNode
	Artifacts                  []ArtifactConfig
	DependsOn                  []string
	LatestTemplateStageVersion *int
}

type PipelineStageTemplateUpdatePreview struct {
	Available                     bool
	Node                          PipelineStageNodeDetail
	CurrentTemplate               model.PipelineStage
	ExpectedSourceTemplateVersion int
	TargetTemplateVersion         int
	Differences                   []PipelineStageTemplateFieldDifference
}

type PipelineStageTemplateFieldDifference struct {
	Field   string
	Current string
	Target  string
}

type PipelineStageTemplateApplyUpdateInput struct {
	ExpectedSourceTemplateStageVersion int
	TargetTemplateStageVersion         int
}

type PipelineDetail struct {
	Pipeline             model.Pipeline
	StageNodes           []PipelineStageNodeDetail
	VariableDeclarations []map[string]any
}

type PipelineSnapshotDetail struct {
	Snapshot          model.PipelineSnapshot
	StagesSnapshot    []model.StageDefinition
	VariablesSnapshot []model.VariableDeclaration
}
