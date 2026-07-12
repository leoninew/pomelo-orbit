package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type RepositoryCreateInput struct {
	ProjectId         string
	Name              string
	Code              string
	RepositoryURL     string
	GitCredentialId   *string
	VariableOverrides []map[string]any
	DefaultBranch     string
}

type RepositoryUpdateInput struct {
	Name              *string
	RepositoryURL     *string
	GitCredentialId   *string
	VariableOverrides *[]map[string]any
	DefaultBranch     *string
}

type RepositoryDetail struct {
	Repository           model.Repository
	GitCredentialName    *string
	VariableDeclarations []map[string]any
}

type WebhookCreateInput struct {
	Name         string
	TemplateId   string
	Secret       string
	BranchFilter *string
}

type WebhookUpdateInput struct {
	Name         *string
	TemplateId   *string
	Secret       *string
	BranchFilter *string
	BranchSet    bool
	Enabled      *bool
}

type WebhookReceiveInput struct {
	WebhookId string
	Headers   map[string]string
	Payload   []byte
}

type WebhookReceiveResult struct {
	Status string
	Reason string
	RunId  string
}

const CredentialExportVersion = "1.0"

type CredentialCreateInput struct {
	ProjectId string
	Name      string
	Type      string
	Data      string
}

type CredentialUpdateInput struct {
	Name *string
	Data *string
}

type CredentialExport struct {
	Version string
	Name    string
	Type    string
	Data    string
}

type CredentialDetail struct {
	Credential model.Credential
	Data       string
}

type ArtifactConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type BuildStageCreateInput struct {
	ProjectId   string
	Name        string
	Image       string
	Script      string
	Artifacts   []ArtifactConfig
	Description string
}

type BuildStageUpdateInput struct {
	Name        *string
	Image       *string
	Script      *string
	Artifacts   *[]ArtifactConfig
	Description *string
}

type BuildStageDetail struct {
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
	Stages               []BuildStageDetail
	VariableDeclarations []map[string]any
}

type PipelineSnapshotDetail struct {
	Snapshot          model.PipelineSnapshot
	StagesSnapshot    []model.StageDefinition
	VariablesSnapshot []model.VariableDeclaration
}

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
	StageRuns         []model.StageRun
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
