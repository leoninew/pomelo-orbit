package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type PipelineRunTriggerInput struct {
	TriggerRef string
	Variables  map[string]string
}

type PipelineRunVariablePreviewInput struct {
	TriggerRef string
	Variables  map[string]string
}

type PipelineRunVariablePreview struct {
	TriggerRef           string
	VariableDeclarations []model.VariableDeclaration
}

type PipelineRunDispatchInput struct {
	PipelineRunId string
}

type ExecutePipelineRunInput struct {
	PipelineRunId string
	Variables     map[string]any
}

type PipelineRunListInput struct {
	ProjectId    string
	RepositoryId string
	PipelineId   string
	DateFrom     string
	DateTo       string
	Page         int
	PerPage      int
}

type PipelineRunDetail struct {
	Run               model.PipelineRun
	VariablesSnapshot []model.VariableDeclaration
	PipelineStageRuns []model.PipelineStageRun
	VersionBinding    *model.PipelineRunVersionBinding
}

type PipelineStageLog struct {
	Logs       string
	Offset     int
	IsComplete bool
}

type ArtifactListInput struct {
	ProjectId    string
	RepositoryId string
	PipelineId   string
	Search       string
	Page         int
	PerPage      int
}
