package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type ArtifactConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type PipelineStageCreateInput struct {
	ProjectId   string
	Name        string
	Image       string
	Script      string
	Artifacts   []ArtifactConfig
	Description string
}

type PipelineStageUpdateInput struct {
	Name        *string
	Image       *string
	Script      *string
	Artifacts   *[]ArtifactConfig
	Description *string
}

type PipelineStageDetail struct {
	Id          string
	Name        string
	Image       string
	Script      string
	Artifacts   []ArtifactConfig
	Description string
	Version     int
	CreatedAt   string
	UpdatedAt   string
}

type StageOrchestration struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type PipelineTemplateCreateInput struct {
	ProjectId            string
	Name                 string
	Description          string
	VariableDeclarations []map[string]any
}

type PipelineTemplateUpdateInput struct {
	Name                 *string
	Description          *string
	Orchestration        *[]StageOrchestration
	VariableDeclarations *[]map[string]any
}

type PipelineTemplateResolveInput struct {
	ProjectId            string
	Orchestration        []StageOrchestration
	VariableDeclarations []map[string]any
}

type PipelineTemplateDetail struct {
	Template             model.PipelineTemplate
	Orchestration        []StageOrchestration
	Stages               []PipelineStageDetail
	VariableDeclarations []map[string]any
}

type PipelineSnapshotDetail struct {
	Snapshot          model.PipelineSnapshot
	StagesSnapshot    []model.StageDefinition
	VariablesSnapshot []model.VariableDeclaration
}
