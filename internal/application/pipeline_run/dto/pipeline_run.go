package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type PipelineRunTriggerInput struct {
	RepositoryId string
	TemplateId   string
	TriggerRef   string
	Variables    map[string]string
}

type PipelineRunDispatchInput struct {
	PipelineRunID string
}

type ExecutePipelineRunInput struct {
	PipelineRunID string
	Variables     map[string]any
}

type PipelineRunListInput struct {
	ProjectId    string
	RepositoryId string
	TemplateId   string
	DateFrom     string
	DateTo       string
	Page         int
	PerPage      int
}

type PipelineRunDetail struct {
	Run               model.PipelineRun
	VariablesSnapshot []model.VariableDeclaration
	PipelineStageRuns []model.PipelineStageRun
}

type PipelineStageLog struct {
	Logs       string
	Offset     int
	IsComplete bool
}

type ArtifactListInput struct {
	ProjectId    string
	RepositoryId string
	TemplateId   string
	Search       string
	Page         int
	PerPage      int
}
