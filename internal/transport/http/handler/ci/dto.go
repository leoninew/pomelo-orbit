package cihandler

import (
	"encoding/json"

	transportresponse "backend/internal/transport/http/response"
)

type PipelineTemplateResp struct {
	Id                   string                    `json:"id"`
	Name                 string                    `json:"name"`
	Description          string                    `json:"description"`
	Orchestration        []StageOrchestrationResp  `json:"orchestration"`
	Stages               []BuildStageResp          `json:"stages"`
	VariableDeclarations []VariableDeclarationResp `json:"variable_declarations"`
	Version              int                       `json:"version"`
	CreatedAt            string                    `json:"created_at"`
	UpdatedAt            string                    `json:"updated_at"`
}

type StageOrchestrationResp struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type StageOrchestrationReq struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type ArtifactConfigResp struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type ArtifactConfigReq struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type VariableDeclarationResp struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     any    `json:"default"`
	Value       any    `json:"value"`
	Secret      bool   `json:"secret"`
	Source      string `json:"source"`
	Editable    bool   `json:"editable"`
}

type VariableDeclarationReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     any    `json:"default"`
	Value       any    `json:"value"`
	Secret      bool   `json:"secret"`
	Source      string `json:"source"`
	Editable    bool   `json:"editable"`
}

type TemplateVariableResolveResp = transportresponse.ListResp[VariableDeclarationResp]

type BuildStageResp struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Script      string               `json:"script"`
	Artifacts   []ArtifactConfigResp `json:"artifacts"`
	Description string               `json:"description"`
	Version     int                  `json:"version"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type PipelineTemplateCreateReq struct {
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	VariableDeclarations []VariableDeclarationReq `json:"variable_declarations"`
}

type TemplateVariableResolveReq struct {
	Orchestration        []StageOrchestrationReq  `json:"orchestration"`
	VariableDeclarations []VariableDeclarationReq `json:"variable_declarations"`
}

type PipelineRunTriggerReq struct {
	TemplateId string            `json:"template_id"`
	TriggerRef string            `json:"trigger_ref"`
	Variables  map[string]string `json:"variables"`
}

type PipelineRunResp struct {
	Id                string                    `json:"id"`
	ProjectId         *string                   `json:"project_id,omitempty"`
	RepositoryId      string                    `json:"repository_id"`
	RepositoryName    string                    `json:"repository_name"`
	SnapshotId        string                    `json:"snapshot_id"`
	TemplateId        string                    `json:"template_id"`
	TemplateName      string                    `json:"template_name"`
	TemplateVersion   int                       `json:"template_version"`
	Trigger           string                    `json:"trigger"`
	TriggerRef        string                    `json:"trigger_ref"`
	VariablesSnapshot []VariableDeclarationResp `json:"variables_snapshot"`
	Status            string                    `json:"status"`
	RetryOf           *string                   `json:"retry_of"`
	StartedAt         *string                   `json:"started_at"`
	FinishedAt        *string                   `json:"finished_at"`
	ErrorMessage      *string                   `json:"error_message"`
	CreatedAt         string                    `json:"created_at"`
	StageRuns         []StageRunResp            `json:"stage_runs"`
}

type StageRunResp struct {
	Id            string  `json:"id"`
	PipelineRunId string  `json:"pipeline_run_id"`
	StageId       string  `json:"stage_id"`
	StageName     string  `json:"stage_name"`
	Status        string  `json:"status"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	ExitCode      *int    `json:"exit_code"`
	ErrorMessage  *string `json:"error_message"`
}

type ArtifactResp struct {
	Id             string  `json:"id"`
	PipelineRunId  string  `json:"pipeline_run_id"`
	RepositoryId   string  `json:"repository_id"`
	RepositoryName string  `json:"repository_name"`
	TemplateId     string  `json:"template_id"`
	TemplateName   string  `json:"template_name"`
	StageName      string  `json:"stage_name"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	Path           *string `json:"path"`
	CreatedAt      string  `json:"created_at"`
}

type PipelineRunArtifactListResp = transportresponse.ListResp[ArtifactResp]

type PipelineStageLogResp struct {
	Logs       string `json:"logs"`
	Offset     int    `json:"offset"`
	IsComplete bool   `json:"is_complete"`
}

type RepositoryResp struct {
	Id                   string                    `json:"id"`
	ProjectId            *string                   `json:"project_id,omitempty"`
	Name                 string                    `json:"name"`
	Code                 string                    `json:"code"`
	RepositoryURL        string                    `json:"repository_url"`
	HasCredential        bool                      `json:"has_credential"`
	GitCredentialId      *string                   `json:"git_credential_id"`
	GitCredentialName    *string                   `json:"git_credential_name,omitempty"`
	VariableDeclarations []VariableDeclarationResp `json:"variable_declarations,omitempty"`
	DefaultBranch        string                    `json:"default_branch"`
	CreatedAt            string                    `json:"created_at"`
	UpdatedAt            string                    `json:"updated_at"`
}

type RepositoryCreateReq struct {
	Name              string                   `json:"name"`
	Code              string                   `json:"code"`
	RepositoryURL     string                   `json:"repository_url"`
	GitCredentialId   *string                  `json:"git_credential_id"`
	VariableOverrides []VariableDeclarationReq `json:"variable_overrides"`
	DefaultBranch     string                   `json:"default_branch"`
}

type RepositoryUpdateReq struct {
	Name              *string                   `json:"name"`
	RepositoryURL     *string                   `json:"repository_url"`
	GitCredentialId   *string                   `json:"git_credential_id"`
	VariableOverrides *[]VariableDeclarationReq `json:"variable_overrides"`
	DefaultBranch     *string                   `json:"default_branch"`
}

type RepositoryWebhookResp struct {
	Id           string  `json:"id"`
	RepositoryId string  `json:"repository_id"`
	Name         string  `json:"name"`
	TemplateId   string  `json:"template_id"`
	BranchFilter *string `json:"branch_filter"`
	Secret       string  `json:"secret"`
	Enabled      bool    `json:"enabled"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type RepositoryWebhookListResp = transportresponse.ListResp[RepositoryWebhookResp]

type RepositoryWebhookReceiveResp struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	RunId  string `json:"run_id,omitempty"`
}

type RepositoryWebhookCreateReq struct {
	Name         string  `json:"name"`
	TemplateId   string  `json:"template_id"`
	Secret       string  `json:"secret"`
	BranchFilter *string `json:"branch_filter"`
}

type RepositoryWebhookUpdateReq struct {
	Name         *string `json:"name"`
	TemplateId   *string `json:"template_id"`
	Secret       *string `json:"secret"`
	BranchFilter *string `json:"branch_filter"`
	BranchSet    bool
	Enabled      *bool `json:"enabled"`
}

func (r *RepositoryWebhookUpdateReq) UnmarshalJSON(data []byte) error {
	type alias RepositoryWebhookUpdateReq
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*r = RepositoryWebhookUpdateReq(decoded)
	_, r.BranchSet = raw["branch_filter"]
	return nil
}

type BuildStageCreateReq struct {
	Name        string              `json:"name"`
	Image       string              `json:"image"`
	Script      string              `json:"script"`
	Artifacts   []ArtifactConfigReq `json:"artifacts"`
	Description string              `json:"description"`
}

type CredentialResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type CredentialDetailResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Data      string `json:"data"`
	CreatedAt string `json:"created_at"`
}

type CredentialCreateReq struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}

type CredentialUpdateReq struct {
	Name *string `json:"name"`
	Data *string `json:"data"`
}

type CredentialExportResp struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

type CredentialImportReq struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

type PipelineSnapshotResp struct {
	Id                string                    `json:"id"`
	TemplateId        string                    `json:"template_id"`
	Version           int                       `json:"version"`
	StagesSnapshot    []SnapshotStageResp       `json:"stages_snapshot"`
	VariablesSnapshot []VariableDeclarationResp `json:"variables_snapshot"`
	CreatedAt         string                    `json:"created_at"`
}

type SnapshotStageResp struct {
	Name      string               `json:"name"`
	Id        string               `json:"id"`
	Image     string               `json:"image"`
	Version   int                  `json:"version"`
	DependsOn []string             `json:"depends_on"`
	Script    string               `json:"script"`
	Artifacts []ArtifactConfigResp `json:"artifacts"`
}
