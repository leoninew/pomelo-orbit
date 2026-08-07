package model

import "time"

const (
	PipelineKindTemplate    = "template"
	PipelineKindApplication = "application"

	VersionForkStrategyLatest = "latest"
	VersionForkStrategyFixed  = "fixed"
)

// Pipeline is either a reusable, non-runnable template or an independently
// configured application pipeline materialized from a template.
type Pipeline struct {
	Id                    string    `db:"id"`
	ProjectId             *string   `db:"project_id"`
	Kind                  string    `db:"kind"`
	SourcePipelineId      *string   `db:"source_pipeline_id"`
	SourceTemplateName    *string   `db:"source_template_name"`
	SourceTemplateVersion *int      `db:"source_template_version"`
	ApplicationId         *string   `db:"application_id"`
	ApplicationName       *string   `db:"application_name"`
	RepositoryId          *string   `db:"repository_id"`
	RepositoryName        *string   `db:"repository_name"`
	VersionForkStrategy   *string   `db:"version_fork_strategy"`
	FixedVersionId        *string   `db:"fixed_version_id"`
	FixedVersionLabel     *string   `db:"fixed_version_label"`
	Name                  string    `db:"name"`
	Description           string    `db:"description"`
	VariableDeclarations  string    `db:"variable_declarations"`
	Version               int       `db:"version"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

// PipelineStage is owned by one Pipeline. Its IDs are therefore safe to use
// as the DAG's local identifiers and are never shared across pipelines.
type PipelineStage struct {
	Id          string    `db:"id"`
	PipelineId  string    `db:"pipeline_id"`
	Name        string    `db:"name"`
	Image       string    `db:"image"`
	Script      string    `db:"script"`
	Artifacts   *string   `db:"artifacts"`
	DependsOn   string    `db:"depends_on"`
	SortOrder   int       `db:"sort_order"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// PipelineSnapshot is an immutable executable input for an application
// pipeline. Template pipelines never own snapshots.
type PipelineSnapshot struct {
	Id                    string    `db:"id"`
	ProjectId             *string   `db:"project_id"`
	PipelineId            string    `db:"pipeline_id"`
	PipelineName          string    `db:"pipeline_name"`
	PipelineVersion       int       `db:"pipeline_version"`
	SourcePipelineId      string    `db:"source_pipeline_id"`
	SourceTemplateName    string    `db:"source_template_name"`
	SourceTemplateVersion int       `db:"source_template_version"`
	ApplicationId         *string   `db:"application_id"`
	ApplicationName       *string   `db:"application_name"`
	RepositoryId          string    `db:"repository_id"`
	RepositoryName        string    `db:"repository_name"`
	VersionForkStrategy   *string   `db:"version_fork_strategy"`
	FixedVersionId        *string   `db:"fixed_version_id"`
	FixedVersionLabel     *string   `db:"fixed_version_label"`
	StagesSnapshot        string    `db:"stages_snapshot"`
	VariablesSnapshot     string    `db:"variables_snapshot"`
	CreatedAt             time.Time `db:"created_at"`
}

type StageDefinition struct {
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	Image       string           `json:"image"`
	DependsOn   []string         `json:"depends_on"`
	Script      string           `json:"script"`
	Artifacts   []ArtifactConfig `json:"artifacts"`
	SortOrder   int              `json:"sort_order"`
	Description string           `json:"description"`
}

type ArtifactConfig struct {
	Name          string  `json:"name"`
	Collector     string  `json:"collector"`
	Reference     string  `json:"reference,omitempty"`
	Command       string  `json:"command,omitempty"`
	Format        string  `json:"format,omitempty"`
	ComponentName *string `json:"component_name,omitempty"`
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
