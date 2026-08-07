package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

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
	Name                     *string
	Description              *string
	VariableDeclarations     *[]map[string]any
	VersionForkStrategy      *string
	FixedVersionId           *string
	ClearVersionForkStrategy bool
}

type PipelineInstantiateInput struct {
	Name          string
	ApplicationId *string
	RepositoryId  string
}

type PipelineStageCreateInput struct {
	Name                string
	Image               string
	Script              string
	Artifacts           []ArtifactConfig
	DependsOn           []string
	SortOrder           int
	Description         string
	VersionForkStrategy *string
	FixedVersionId      *string
}

type PipelineStageUpdateInput struct {
	Name                     *string
	Image                    *string
	Script                   *string
	Artifacts                *[]ArtifactConfig
	DependsOn                *[]string
	SortOrder                *int
	Description              *string
	VersionForkStrategy      *string
	FixedVersionId           *string
	ClearVersionForkStrategy bool
}

type PipelineStageDetail struct {
	Stage     model.PipelineStage
	Artifacts []ArtifactConfig
	DependsOn []string
}

type PipelineDetail struct {
	Pipeline             model.Pipeline
	Stages               []PipelineStageDetail
	VariableDeclarations []map[string]any
}

type PipelineSnapshotDetail struct {
	Snapshot          model.PipelineSnapshot
	StagesSnapshot    []model.StageDefinition
	VariablesSnapshot []model.VariableDeclaration
}
