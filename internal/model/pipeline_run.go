package model

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

type PipelineStageRun struct {
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
