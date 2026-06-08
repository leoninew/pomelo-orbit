package orbit

import "time"

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

type Repository struct {
	Id                string  `db:"id"`
	ProjectId         *string `db:"project_id"`
	Name              string  `db:"name"`
	Code              string  `db:"code"`
	RepositoryURL     string  `db:"repository_url"`
	GitCredentialId   *string `db:"git_credential_id"`
	VariableOverrides string  `db:"variable_overrides"`
	DefaultBranch     string  `db:"default_branch"`
}

type PipelineSnapshot struct {
	Id                string  `db:"id"`
	ProjectId         *string `db:"project_id"`
	TemplateId        string  `db:"template_id"`
	Version           int     `db:"version"`
	StagesSnapshot    string  `db:"stages_snapshot"`
	VariablesSnapshot string  `db:"variables_snapshot"`
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

type Credential struct {
	Id            string  `db:"id"`
	ProjectId     *string `db:"project_id"`
	Name          string  `db:"name"`
	Type          string  `db:"type"`
	EncryptedData string  `db:"encrypted_data"`
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

type Application struct {
	Id              string  `db:"id"`
	ProjectId       *string `db:"project_id"`
	Name            string  `db:"name"`
	Code            string  `db:"code"`
	ImagePullPolicy string  `db:"image_pull_policy"`
	Status          string  `db:"status"`
	RouteManaged    bool    `db:"route_managed"`
}

type Deployment struct {
	Id            string     `db:"id"`
	ProjectId     *string    `db:"project_id"`
	ApplicationId *string    `db:"application_id"`
	OperationType string     `db:"operation_type"`
	Status        string     `db:"status"`
	StartedAt     time.Time  `db:"started_at"`
	FinishedAt    *time.Time `db:"finished_at"`
	ErrorMessage  *string    `db:"error_message"`
}

type ApplicationConfigFile struct {
	Id            string `db:"id"`
	ApplicationId string `db:"application_id"`
	Path          string `db:"path"`
	Content       string `db:"content"`
}

type ApplicationServiceConfig struct {
	Id            string  `db:"id"`
	ApplicationId string  `db:"application_id"`
	ServiceName   string  `db:"service_name"`
	Image         *string `db:"image"`
	Environment   *string `db:"environment"`
	Volumes       *string `db:"volumes"`
}

type ApplicationRoute struct {
	Id            string `db:"id"`
	ApplicationId string `db:"application_id"`
	ServiceName   string `db:"service_name"`
	Domain        string `db:"domain"`
	Port          int    `db:"port"`
}
