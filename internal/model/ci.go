package model

import "time"

type Credential struct {
	Id            string    `db:"id"`
	ProjectId     *string   `db:"project_id"`
	Name          string    `db:"name"`
	Type          string    `db:"type"`
	EncryptedData string    `db:"encrypted_data"`
	CreatedAt     time.Time `db:"created_at"`
}

type PipelineTemplate struct {
	Id                   string    `db:"id"`
	ProjectId            *string   `db:"project_id"`
	Name                 string    `db:"name"`
	Description          string    `db:"description"`
	VariableDeclarations string    `db:"variable_declarations"`
	Version              int       `db:"version"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

type BuildStage struct {
	Id          string    `db:"id"`
	ProjectId   *string   `db:"project_id"`
	Name        string    `db:"name"`
	Image       string    `db:"image"`
	Script      string    `db:"script"`
	Artifacts   *string   `db:"artifacts"`
	Description string    `db:"description"`
	Version     int       `db:"version"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type PipelineTemplateStage struct {
	Id           string `db:"id"`
	TemplateId   string `db:"template_id"`
	StageId      string `db:"stage_id"`
	StageName    string `db:"stage_name"`
	StageVersion int    `db:"stage_version"`
	DependsOn    string `db:"depends_on"`
	SortOrder    int    `db:"sort_order"`
}

type PipelineSnapshot struct {
	Id                string    `db:"id"`
	ProjectId         *string   `db:"project_id"`
	TemplateId        string    `db:"template_id"`
	Version           int       `db:"version"`
	StagesSnapshot    string    `db:"stages_snapshot"`
	VariablesSnapshot string    `db:"variables_snapshot"`
	CreatedAt         time.Time `db:"created_at"`
}

type Repository struct {
	Id                string    `db:"id"`
	ProjectId         *string   `db:"project_id"`
	Name              string    `db:"name"`
	Code              string    `db:"code"`
	RepositoryURL     string    `db:"repository_url"`
	GitCredentialId   *string   `db:"git_credential_id"`
	VariableOverrides string    `db:"variable_overrides"`
	DefaultBranch     string    `db:"default_branch"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type RepositoryWebhook struct {
	Id              string    `db:"id"`
	RepositoryId    string    `db:"repository_id"`
	Name            string    `db:"name"`
	TemplateId      string    `db:"template_id"`
	BranchFilter    *string   `db:"branch_filter"`
	EncryptedSecret string    `db:"encrypted_secret"`
	Enabled         bool      `db:"enabled"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type PipelineRun struct {
	Id                string     `db:"id"`
	ProjectId         *string    `db:"project_id"`
	RepositoryId      string     `db:"repository_id"`
	RepositoryName    string     `db:"repository_name"`
	SnapshotId        string     `db:"snapshot_id"`
	TemplateId        string     `db:"template_id"`
	TemplateName      string     `db:"template_name"`
	TemplateVersion   int        `db:"template_version"`
	Trigger           string     `db:"trigger"`
	TriggerRef        string     `db:"trigger_ref"`
	VariablesSnapshot string     `db:"variables_snapshot"`
	Status            string     `db:"status"`
	RetryOf           *string    `db:"retry_of"`
	StartedAt         *time.Time `db:"started_at"`
	FinishedAt        *time.Time `db:"finished_at"`
	ErrorMessage      *string    `db:"error_message"`
	CreatedAt         time.Time  `db:"created_at"`
}

type StageRun struct {
	Id            string     `db:"id"`
	PipelineRunId string     `db:"pipeline_run_id"`
	StageId       string     `db:"stage_id"`
	StageName     string     `db:"stage_name"`
	Status        string     `db:"status"`
	StartedAt     *time.Time `db:"started_at"`
	FinishedAt    *time.Time `db:"finished_at"`
	ExitCode      *int       `db:"exit_code"`
	ErrorMessage  *string    `db:"error_message"`
}

type Artifact struct {
	Id             string    `db:"id"`
	ProjectId      *string   `db:"project_id"`
	PipelineRunId  string    `db:"pipeline_run_id"`
	RepositoryId   string    `db:"repository_id"`
	RepositoryName string    `db:"repository_name"`
	TemplateId     string    `db:"template_id"`
	TemplateName   string    `db:"template_name"`
	StageName      string    `db:"stage_name"`
	Type           string    `db:"type"`
	Name           string    `db:"name"`
	Path           *string   `db:"path"`
	CreatedAt      time.Time `db:"created_at"`
}

type StageDefinition struct {
	Name      string           `json:"name"`
	Id        string           `json:"id"`
	Image     string           `json:"image"`
	Version   int              `json:"version"`
	DependsOn []string         `json:"depends_on"`
	Script    string           `json:"script"`
	Artifacts []ArtifactConfig `json:"artifacts"`
}

type ArtifactConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type VariableDeclaration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     any    `json:"default"`
	Value       any    `json:"value"`
	Secret      bool   `json:"secret"`
	Source      string `json:"source"`
	Editable    bool   `json:"editable"`
}
