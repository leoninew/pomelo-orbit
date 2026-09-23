package pipelinerunrepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	_ "modernc.org/sqlite"
)

func TestListQueriesBindNamedPaginationParameters(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline_run (
			id TEXT PRIMARY KEY, project_id TEXT, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL,
			snapshot_id TEXT NOT NULL, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL,
			pipeline_version INTEGER NOT NULL, trigger TEXT NOT NULL, repository_ref TEXT NOT NULL,
			variables_snapshot TEXT NOT NULL, status TEXT NOT NULL, retry_of TEXT,
			environment_id TEXT, environment_target_type TEXT, environment_target_revision INTEGER,
			ssh_credential_id TEXT, ssh_credential_revision INTEGER, started_at DATETIME, finished_at DATETIME, error_message TEXT, created_at DATETIME NOT NULL
		);
		CREATE TABLE artifact (
			id TEXT PRIMARY KEY, project_id TEXT, pipeline_run_id TEXT NOT NULL, repository_id TEXT NOT NULL,
			repository_name TEXT NOT NULL, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL,
			pipeline_stage_id TEXT NOT NULL, stage_name TEXT NOT NULL, collector TEXT NOT NULL, name TEXT NOT NULL,
			location TEXT, value TEXT, value_format TEXT, image_ref TEXT, local_image_sha256 TEXT,
			source_artifact_id TEXT, created_at DATETIME NOT NULL
		);
		CREATE TABLE pipeline_run_version_binding (
			pipeline_run_id TEXT PRIMARY KEY, application_id TEXT NOT NULL, application_name TEXT NOT NULL,
			source_version_id TEXT NOT NULL, source_version_label TEXT NOT NULL,
			generated_version_id TEXT, generated_version_label TEXT
		);
		CREATE TABLE version_component (id TEXT PRIMARY KEY, name TEXT NOT NULL, artifact_id TEXT);
	`); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	createdAt := time.Now().UTC()
	if _, err := database.ExecContext(ctx, `
		INSERT INTO pipeline_run (
			id, project_id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name,
			pipeline_version, trigger, repository_ref, variables_snapshot, status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "run-1", "project-1", "repository-1", "Repository", "snapshot-1", "pipeline-1", "Pipeline", 1, "manual", "main", "{}", "waiting_to_run", createdAt); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO artifact (
			id, project_id, pipeline_run_id, repository_id, repository_name, pipeline_id, pipeline_name,
			pipeline_stage_id, stage_name, collector, name, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "artifact-1", "project-1", "run-1", "repository-1", "Repository", "pipeline-1", "Pipeline", "stage-1", "Build", "docker_image", "image", createdAt); err != nil {
		t.Fatal(err)
	}

	repository := NewRepository(database)
	runs, err := repository.ListPipelineRuns(ctx, "project-1", "", "", nil, nil, 1, 20)
	if err != nil {
		t.Fatalf("list pipeline runs: %v", err)
	}
	if len(runs.Items) != 1 || runs.Items[0].Id != "run-1" {
		t.Fatalf("pipeline runs=%+v", runs.Items)
	}
	from := createdAt.Add(-time.Minute)
	to := createdAt.Add(time.Minute)
	filteredRuns, err := repository.ListPipelineRuns(ctx, "project-1", "repository-1", "pipeline-1", &from, &to, 1, 1)
	if err != nil {
		t.Fatalf("list filtered pipeline runs: %v", err)
	}
	if filteredRuns.Total != 1 || len(filteredRuns.Items) != 1 || filteredRuns.Items[0].Id != "run-1" {
		t.Fatalf("filtered pipeline runs=%+v", filteredRuns)
	}

	artifacts, err := repository.ListArtifacts(ctx, "project-1", "", "", 1, 20, "")
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts.Items) != 1 || artifacts.Items[0].Id != "artifact-1" {
		t.Fatalf("artifacts=%+v", artifacts.Items)
	}

	projectId := "project-1"
	runArtifacts, err := repository.ListArtifactsByRun(ctx, projectId, "run-1")
	if err != nil {
		t.Fatalf("list artifacts by run: %v", err)
	}
	if len(runArtifacts) != 1 || runArtifacts[0].Id != "artifact-1" {
		t.Fatalf("run artifacts=%+v", runArtifacts)
	}
}

func TestBeginPipelineStageRunRequiresRunningParentRun(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, status TEXT NOT NULL);
		CREATE TABLE pipeline_stage_run (
			id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, stage_id TEXT NOT NULL,
			stage_name TEXT NOT NULL, status TEXT NOT NULL, started_at DATETIME,
			finished_at DATETIME, exit_code INTEGER, error_message TEXT
		);
		INSERT INTO pipeline_run (id, project_id, status) VALUES ('run-1', 'project-1', 'canceled');
		INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status)
		VALUES ('stage-run-1', 'run-1', 'stage-1', 'build', 'waiting_to_run');
	`); err != nil {
		t.Fatal(err)
	}

	repository := NewRepository(database)
	begun, err := repository.BeginPipelineStageRun(context.Background(), "project-1", "stage-run-1")
	if err != nil {
		t.Fatalf("begin stage run: %v", err)
	}
	if begun {
		t.Fatal("canceled pipeline run must not begin a waiting stage")
	}

	if _, err := database.Exec(`UPDATE pipeline_run SET status = 'running' WHERE id = 'run-1'`); err != nil {
		t.Fatal(err)
	}
	begun, err = repository.BeginPipelineStageRun(context.Background(), "project-1", "stage-run-1")
	if err != nil {
		t.Fatalf("begin stage run for running parent: %v", err)
	}
	if !begun {
		t.Fatal("running pipeline run must begin a waiting stage")
	}

	var stageStatus string
	if err := database.QueryRow(`SELECT status FROM pipeline_stage_run WHERE id = 'stage-run-1'`).Scan(&stageStatus); err != nil {
		t.Fatal(err)
	}
	if stageStatus != status.WorkStatusRunning {
		t.Fatalf("stage status = %q, want %q", stageStatus, status.WorkStatusRunning)
	}
}

func TestDeletePipelineRunDeletesRelatedRecords(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, project_id TEXT NOT NULL);
		CREATE TABLE pipeline_stage_run (id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL);
		CREATE TABLE pipeline_run_version_binding (pipeline_run_id TEXT PRIMARY KEY);
		CREATE TABLE artifact (id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL);
		INSERT INTO pipeline_run (id, project_id) VALUES ('run-1', 'project-1');
		INSERT INTO pipeline_stage_run (id, pipeline_run_id) VALUES ('stage-run-1', 'run-1');
		INSERT INTO pipeline_run_version_binding (pipeline_run_id) VALUES ('run-1');
		INSERT INTO artifact (id, pipeline_run_id) VALUES ('artifact-1', 'run-1');
	`); err != nil {
		t.Fatal(err)
	}

	repoStore := NewRepository(database)
	if err := repoStore.DeletePipelineRun(context.Background(), "project-2", "run-1"); err != nil {
		t.Fatalf("DeletePipelineRun for another project returned error: %v", err)
	}
	var runCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM pipeline_run").Scan(&runCount); err != nil {
		t.Fatal(err)
	}
	if runCount != 1 {
		t.Fatalf("cross-project delete removed %d runs", 1-runCount)
	}
	if err := repoStore.DeletePipelineRun(context.Background(), "project-1", "run-1"); err != nil {
		t.Fatalf("DeletePipelineRun returned error: %v", err)
	}
	for _, table := range []string{"pipeline_run", "pipeline_stage_run", "pipeline_run_version_binding", "artifact"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0", table, count)
		}
	}
}

func TestPipelineRunDerivedQueriesRequireMatchingProject(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, status TEXT NOT NULL);
		CREATE TABLE pipeline_stage_run (
			id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, stage_id TEXT NOT NULL,
			stage_name TEXT NOT NULL, status TEXT NOT NULL, started_at DATETIME,
			finished_at DATETIME, exit_code INTEGER, error_message TEXT
		);
		CREATE TABLE pipeline_run_version_binding (
			pipeline_run_id TEXT PRIMARY KEY, application_id TEXT NOT NULL, application_name TEXT NOT NULL,
			source_version_id TEXT NOT NULL, source_version_label TEXT NOT NULL,
			generated_version_id TEXT, generated_version_label TEXT
		);
		CREATE TABLE artifact (
			id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, pipeline_stage_id TEXT NOT NULL,
			name TEXT NOT NULL, collector TEXT NOT NULL, value TEXT, value_format TEXT
		);
		INSERT INTO pipeline_run (id, project_id, status) VALUES ('run-1', 'project-1', 'running');
		INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status)
		VALUES ('stage-run-1', 'run-1', 'stage-1', 'Build', 'waiting_to_run');
		INSERT INTO pipeline_run_version_binding (pipeline_run_id, application_id, application_name, source_version_id, source_version_label)
		VALUES ('run-1', 'application-1', 'Application', 'version-1', 'v1');
		INSERT INTO artifact (id, pipeline_run_id, pipeline_stage_id, name, collector, value, value_format)
		VALUES ('artifact-1', 'run-1', 'stage-1', 'commit', 'command', '0123456789012345678901234567890123456789', 'git_object_id');
	`); err != nil {
		t.Fatal(err)
	}

	repoStore := NewRepository(database)
	stageRuns, err := repoStore.ListPipelineStageRuns(context.Background(), "project-1", "run-1")
	if err != nil {
		t.Fatalf("ListPipelineStageRuns: %v", err)
	}
	if len(stageRuns) != 1 || stageRuns[0].Id != "stage-run-1" {
		t.Fatalf("stage runs=%+v", stageRuns)
	}
	stageRuns, err = repoStore.ListPipelineStageRuns(context.Background(), "project-2", "run-1")
	if err != nil {
		t.Fatalf("ListPipelineStageRuns for another project: %v", err)
	}
	if len(stageRuns) != 0 {
		t.Fatalf("cross-project stage runs=%+v", stageRuns)
	}
	if _, err := repoStore.PipelineRunVersionBinding(context.Background(), "project-2", "run-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-project version binding error=%v, want not found", err)
	}
	if _, err := repoStore.CommandArtifactByRunStageAndName(context.Background(), "project-2", "run-1", "stage-1", "commit"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-project command artifact error=%v, want not found", err)
	}
}

func TestPipelineRunTargetSnapshotRoundTrip(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	_, err = database.Exec(`CREATE TABLE pipeline_run (
		id TEXT PRIMARY KEY, project_id TEXT, repository_id TEXT NOT NULL, repository_name TEXT NOT NULL,
		snapshot_id TEXT NOT NULL, pipeline_id TEXT NOT NULL, pipeline_name TEXT NOT NULL,
		pipeline_version INTEGER NOT NULL, trigger TEXT NOT NULL, repository_ref TEXT NOT NULL,
		variables_snapshot TEXT NOT NULL, status TEXT NOT NULL, retry_of TEXT,
		environment_id TEXT, environment_target_type TEXT, environment_target_revision INTEGER,
		ssh_credential_id TEXT, ssh_credential_revision INTEGER,
		started_at DATETIME, finished_at DATETIME, error_message TEXT, created_at DATETIME NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}
	projectId, environmentId, targetType, credentialId := "project-1", "environment-1", model.EnvironmentTargetTypeSSH, "credential-1"
	targetRevision, credentialRevision := int64(3), int64(5)
	run := model.PipelineRun{Id: "run-1", ProjectId: &projectId, RepositoryId: "repo-1", RepositoryName: "repo",
		SnapshotId: "snapshot-1", PipelineId: "pipeline-1", PipelineName: "pipeline", PipelineVersion: 1,
		Trigger: "manual", RepositoryRef: "main", VariablesSnapshot: `[]`, Status: status.WorkStatusWaitingToRun,
		EnvironmentId: &environmentId, EnvironmentTargetType: &targetType, EnvironmentTargetRevision: &targetRevision,
		SSHCredentialId: &credentialId, SSHCredentialRevision: &credentialRevision}
	store := NewRepository(database)
	if err := store.CreatePipelineRun(context.Background(), run, nil, nil); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.PipelineRun(context.Background(), projectId, run.Id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.EnvironmentId == nil || *loaded.EnvironmentId != environmentId ||
		loaded.EnvironmentTargetType == nil || *loaded.EnvironmentTargetType != targetType ||
		loaded.EnvironmentTargetRevision == nil || *loaded.EnvironmentTargetRevision != targetRevision ||
		loaded.SSHCredentialId == nil || *loaded.SSHCredentialId != credentialId ||
		loaded.SSHCredentialRevision == nil || *loaded.SSHCredentialRevision != credentialRevision {
		t.Fatalf("persisted run target = %+v", loaded)
	}
	listed, err := store.ListPipelineRuns(context.Background(), projectId, "", "", nil, nil, 1, 10)
	if err != nil || len(listed.Items) != 1 || listed.Items[0].SSHCredentialRevision == nil ||
		*listed.Items[0].SSHCredentialRevision != credentialRevision {
		t.Fatalf("listed run target = %+v, err = %v", listed, err)
	}
}
