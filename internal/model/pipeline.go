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
