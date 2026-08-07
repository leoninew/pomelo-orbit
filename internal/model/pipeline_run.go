package model

import "time"

type PipelineRun struct {
	Id                string     `db:"id"`
	ProjectId         *string    `db:"project_id"`
	RepositoryId      string     `db:"repository_id"`
	RepositoryName    string     `db:"repository_name"`
	SnapshotId        string     `db:"snapshot_id"`
	PipelineId        string     `db:"pipeline_id"`
	PipelineName      string     `db:"pipeline_name"`
	PipelineVersion   int        `db:"pipeline_version"`
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
	Id                    string    `db:"id"`
	ProjectId             *string   `db:"project_id"`
	PipelineRunId         string    `db:"pipeline_run_id"`
	RepositoryId          string    `db:"repository_id"`
	RepositoryName        string    `db:"repository_name"`
	PipelineId            string    `db:"pipeline_id"`
	PipelineName          string    `db:"pipeline_name"`
	PipelineStageId       string    `db:"pipeline_stage_id"`
	StageName             string    `db:"stage_name"`
	Collector             string    `db:"collector"`
	Name                  string    `db:"name"`
	Location              *string   `db:"location"`
	Value                 *string   `db:"value"`
	ValueFormat           *string   `db:"value_format"`
	ImageRef              *string   `db:"image_ref"`
	LocalImageSha256      *string   `db:"local_image_sha256"`
	SourceArtifactId      *string   `db:"source_artifact_id"`
	SourceCommitSha       *string   `db:"source_commit_sha"`
	ApplicationId         *string   `db:"application_id"`
	ApplicationName       *string   `db:"application_name"`
	SourceVersionId       *string   `db:"source_version_id"`
	SourceVersionLabel    *string   `db:"source_version_label"`
	GeneratedVersionId    *string   `db:"generated_version_id"`
	GeneratedVersionLabel *string   `db:"generated_version_label"`
	VersionComponentId    *string   `db:"version_component_id"`
	VersionComponentName  *string   `db:"version_component_name"`
	CreatedAt             time.Time `db:"created_at"`
}

// PipelineRunVersionBinding records the single source Version and optional
// generated Version for a Run. Component-to-artifact lineage lives on the
// generated Version's components.
type PipelineRunVersionBinding struct {
	PipelineRunId         string  `db:"pipeline_run_id"`
	ApplicationId         string  `db:"application_id"`
	ApplicationName       string  `db:"application_name"`
	SourceVersionId       string  `db:"source_version_id"`
	SourceVersionLabel    string  `db:"source_version_label"`
	GeneratedVersionId    *string `db:"generated_version_id"`
	GeneratedVersionLabel *string `db:"generated_version_label"`
}
