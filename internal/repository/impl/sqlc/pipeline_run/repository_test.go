package pipelinerunrepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
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
			started_at DATETIME, finished_at DATETIME, error_message TEXT, created_at DATETIME NOT NULL
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

	projectID := "project-1"
	runArtifacts, err := repository.ListArtifactsByRun(ctx, &projectID, "run-1")
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
		CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, status TEXT NOT NULL);
		CREATE TABLE pipeline_stage_run (
			id TEXT PRIMARY KEY, pipeline_run_id TEXT NOT NULL, stage_id TEXT NOT NULL,
			stage_name TEXT NOT NULL, status TEXT NOT NULL, started_at DATETIME,
			finished_at DATETIME, exit_code INTEGER, error_message TEXT
		);
		INSERT INTO pipeline_run (id, status) VALUES ('run-1', 'canceled');
		INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status)
		VALUES ('stage-run-1', 'run-1', 'stage-1', 'build', 'waiting_to_run');
	`); err != nil {
		t.Fatal(err)
	}

	repository := NewRepository(database)
	begun, err := repository.BeginPipelineStageRun(context.Background(), "stage-run-1")
	if err != nil {
		t.Fatalf("begin stage run: %v", err)
	}
	if begun {
		t.Fatal("canceled pipeline run must not begin a waiting stage")
	}

	if _, err := database.Exec(`UPDATE pipeline_run SET status = 'running' WHERE id = 'run-1'`); err != nil {
		t.Fatal(err)
	}
	begun, err = repository.BeginPipelineStageRun(context.Background(), "stage-run-1")
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
