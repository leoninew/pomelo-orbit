package model

import "time"

const (
	PipelineKindTemplate    = "template"
	PipelineKindApplication = "application"

	PipelineStageKindTemplate    = "template"
	PipelineStageKindApplication = "application"

	PipelineStageNodeTypeTemplateReference = "template_reference"
	PipelineStageNodeTypeApplication       = "application"

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

// PipelineStage is either a project-scoped reusable template or an application
// pipeline's private executable node. Template pipelines use
// PipelineStageReference for their DAG nodes instead of owning stages.
type PipelineStage struct {
	Id                             string    `db:"id"`
	ProjectId                      string    `db:"project_id"`
	Kind                           string    `db:"kind"`
	PipelineId                     *string   `db:"pipeline_id"`
	Name                           string    `db:"name"`
	Image                          string    `db:"image"`
	Script                         string    `db:"script"`
	Description                    string    `db:"description"`
	Version                        *int      `db:"version"`
	SourceTemplateStageId          *string   `db:"source_template_stage_id"`
	SourceTemplateStageName        *string   `db:"source_template_stage_name"`
	SourceTemplateStageVersion     *int      `db:"source_template_stage_version"`
	SourceTemplateStageDescription *string   `db:"source_template_stage_description"`
	Artifacts                      *string   `db:"artifacts"`
	DependsOn                      *string   `db:"depends_on"`
	SortOrder                      *int      `db:"sort_order"`
	CreatedAt                      time.Time `db:"created_at"`
	UpdatedAt                      time.Time `db:"updated_at"`
}

// PipelineStageReference is a copied template-stage definition plus the
// Template Pipeline-local DAG node configuration.
type PipelineStageReference struct {
	Id                             string    `db:"id"`
	PipelineId                     string    `db:"pipeline_id"`
	SourceTemplateStageId          string    `db:"source_template_stage_id"`
	SourceTemplateStageName        string    `db:"source_template_stage_name"`
	SourceTemplateStageVersion     int       `db:"source_template_stage_version"`
	SourceTemplateStageDescription string    `db:"source_template_stage_description"`
	Name                           string    `db:"name"`
	Image                          string    `db:"image"`
	Script                         string    `db:"script"`
	Description                    string    `db:"description"`
	Artifacts                      string    `db:"artifacts"`
	DependsOn                      string    `db:"depends_on"`
	SortOrder                      int       `db:"sort_order"`
	CreatedAt                      time.Time `db:"created_at"`
	UpdatedAt                      time.Time `db:"updated_at"`
}

// PipelineStageNode is the editor-facing union of a template reference and
// an application stage. It is not a separately persisted resource.
type PipelineStageNode struct {
	Id                             string
	NodeType                       string
	PipelineId                     string
	Name                           string
	Image                          string
	Script                         string
	Description                    string
	DependsOn                      string
	SortOrder                      int
	SourceTemplateStageId          string
	SourceTemplateStageName        string
	SourceTemplateStageVersion     int
	SourceTemplateStageDescription string
	Artifacts                      *string
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

// PipelineSnapshot is an immutable structural and historical snapshot for an
// application pipeline. Run variable values are stored on PipelineRun.
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
	Id                         string           `json:"id"`
	Name                       string           `json:"name"`
	Image                      string           `json:"image"`
	DependsOn                  []string         `json:"depends_on"`
	Script                     string           `json:"script"`
	Artifacts                  []ArtifactConfig `json:"artifacts"`
	SortOrder                  int              `json:"sort_order"`
	Description                string           `json:"description"`
	SourceTemplateStageId      string           `json:"source_template_stage_id"`
	SourceTemplateStageName    string           `json:"source_template_stage_name"`
	SourceTemplateStageVersion int              `json:"source_template_stage_version"`
}

type ArtifactConfig struct {
	Name          string  `json:"name"`
	Collector     string  `json:"collector"`
	Reference     string  `json:"reference,omitempty"`
	Command       string  `json:"command,omitempty"`
	Format        string  `json:"format,omitempty"`
	ComponentName *string `json:"component_name,omitempty"`
}

type StageVariableDefault struct {
	StageId   string `json:"stage_id"`
	StageName string `json:"stage_name"`
	Default   any    `json:"default"`
}

type VariableDeclaration struct {
	Name          string                 `json:"name"`
	StageId       string                 `json:"stage_id,omitempty"`
	StageName     string                 `json:"stage_name,omitempty"`
	Description   string                 `json:"description"`
	Default       any                    `json:"default"`
	Value         any                    `json:"value"`
	Secret        bool                   `json:"secret"`
	Source        string                 `json:"source"`
	Editable      bool                   `json:"editable"`
	StageDefaults []StageVariableDefault `json:"stage_defaults,omitempty"`
}
