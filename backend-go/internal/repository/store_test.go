package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func TestNewId(t *testing.T) {
	id := NewId()
	if len(id) != 26 {
		t.Fatalf("expected 26-char id, got %q", id)
	}
	if id == NewId() {
		t.Fatal("expected unique ids")
	}
}

func TestStorePipelineRunRepositorySnapshot(t *testing.T) {
	database := openStoreDB(t)
	defer func() { _ = database.Close() }()
	store := NewStore(database, "sqlite")
	ctx := context.Background()

	run, err := store.PipelineRun(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.RepositoryId != "repo-1" || run.SnapshotId != "snapshot-1" {
		t.Fatalf("unexpected run: %+v", run)
	}
	repo, err := store.Repository(ctx, "repo-1")
	if err != nil {
		t.Fatal(err)
	}
	if repo.Code != "demo" {
		t.Fatalf("unexpected repo: %+v", repo)
	}
	snapshot, err := store.PipelineSnapshot(ctx, "snapshot-1")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.StagesSnapshot != "[]" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestStoreStatusUpdates(t *testing.T) {
	database := openStoreDB(t)
	defer func() { _ = database.Close() }()
	store := NewStore(database, "sqlite")
	ctx := context.Background()

	if err := store.MarkPipelineRunRunning(ctx, "run-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.CompletePipelineRun(ctx, "run-1", WorkStatusRanToCompletion, ""); err != nil {
		t.Fatal(err)
	}
	run, err := store.PipelineRun(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != WorkStatusRanToCompletion {
		t.Fatalf("unexpected status: %s", run.Status)
	}
}

func openStoreDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	schema := []string{
		`CREATE TABLE pipeline_run (id TEXT PRIMARY KEY, project_id TEXT, repository_id TEXT, repository_name TEXT, snapshot_id TEXT, template_id TEXT, template_name TEXT, template_version INTEGER, trigger TEXT, trigger_ref TEXT, variables_snapshot TEXT, status TEXT, retry_of TEXT, started_at DATETIME, finished_at DATETIME, error_message TEXT, created_at DATETIME NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE repository (id TEXT PRIMARY KEY, project_id TEXT, name TEXT, code TEXT, repository_url TEXT, git_credential_id TEXT, variable_overrides TEXT, default_branch TEXT, created_at DATETIME NOT NULL DEFAULT (datetime('now')), updated_at DATETIME NOT NULL DEFAULT (datetime('now')))`,
		`CREATE TABLE pipeline_snapshot (id TEXT PRIMARY KEY, project_id TEXT, template_id TEXT, version INTEGER, stages_snapshot TEXT, variables_snapshot TEXT, created_at DATETIME NOT NULL DEFAULT (datetime('now')))`,
		`INSERT INTO repository (id, name, code, repository_url, variable_overrides, default_branch) VALUES ('repo-1', 'Repo', 'demo', 'https://example.invalid/repo.git', '{}', 'main')`,
		`INSERT INTO pipeline_snapshot (id, template_id, version, stages_snapshot, variables_snapshot) VALUES ('snapshot-1', 'template-1', 1, '[]', '[]')`,
		`INSERT INTO pipeline_run (id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status) VALUES ('run-1', 'repo-1', 'Repo', 'snapshot-1', 'template-1', 'Template', 1, 'manual', '', '{}', 'waiting_to_run')`,
	}
	for _, statement := range schema {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return database
}
