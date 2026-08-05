package model

import "time"

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

type PipelineStage struct {
	Id                  string               `db:"id"`
	ProjectId           *string              `db:"project_id"`
	Name                string               `db:"name"`
	Image               string               `db:"image"`
	Script              string               `db:"script"`
	Artifacts           *string              `db:"artifacts"`
	BuildVersionBinding *BuildVersionBinding `db:"-"`
	Description         string               `db:"description"`
	Version             int                  `db:"version"`
	CreatedAt           time.Time            `db:"created_at"`
	UpdatedAt           time.Time            `db:"updated_at"`
}

// BuildVersionBinding describes the optional application-version fork performed by a build stage.
type BuildVersionBinding struct {
	ApplicationId   string  `json:"application_id"`
	ApplicationName string  `json:"application_name"`
	ComponentName   string  `json:"component_name"`
	ForkStrategy    string  `json:"fork_strategy"`
	FixedVersionId  *string `json:"fixed_version_id,omitempty"`
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

type StageDefinition struct {
	Name                string               `json:"name"`
	Id                  string               `json:"id"`
	Image               string               `json:"image"`
	Version             int                  `json:"version"`
	DependsOn           []string             `json:"depends_on"`
	Script              string               `json:"script"`
	Artifacts           []ArtifactConfig     `json:"artifacts"`
	BuildVersionBinding *BuildVersionBinding `json:"build_version_binding,omitempty"`
}

type ArtifactConfig struct {
	Name      string `json:"name"`
	Collector string `json:"collector"`
	Reference string `json:"reference,omitempty"`
	Command   string `json:"command,omitempty"`
	Format    string `json:"format,omitempty"`
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
