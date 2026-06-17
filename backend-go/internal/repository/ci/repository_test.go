package ci

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"backend/internal/repository"
	"backend/internal/repository/model"
)

func TestRepositoryPipelineRunRepositorySnapshot(t *testing.T) {
	database := openRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
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

func TestRepositoryStatusUpdates(t *testing.T) {
	database := openRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
	ctx := context.Background()

	if err := store.MarkPipelineRunRunning(ctx, "run-1"); err != nil {
		t.Fatal(err)
	}
	if err := store.CompletePipelineRun(ctx, "run-1", repository.WorkStatusRanToCompletion, ""); err != nil {
		t.Fatal(err)
	}
	run, err := store.PipelineRun(ctx, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != repository.WorkStatusRanToCompletion {
		t.Fatalf("unexpected status: %s", run.Status)
	}
}

func TestCredentialRepositoryCRUD(t *testing.T) {
	database := openCredentialRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
	ctx := context.Background()
	projectId := "project-1"

	credential := model.Credential{Id: "credential-2", ProjectId: &projectId, Name: "Registry", Type: "registry_token", EncryptedData: "encrypted-2"}
	if err := store.CreateCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Credential(ctx, "credential-2")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "Registry" || loaded.EncryptedData != "encrypted-2" {
		t.Fatalf("unexpected loaded credential: %+v", loaded)
	}

	page, err := store.ListCredentials(ctx, projectId, 1, 20, "reg")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Id != "credential-2" {
		t.Fatalf("unexpected credential page: %+v", page)
	}

	byName, err := store.CredentialByName(ctx, projectId, "Registry")
	if err != nil {
		t.Fatal(err)
	}
	if byName.Id != "credential-2" {
		t.Fatalf("unexpected credential by name: %+v", byName)
	}

	loaded.Name = "Registry Updated"
	loaded.EncryptedData = "encrypted-updated"
	if err := store.UpdateCredential(ctx, loaded); err != nil {
		t.Fatal(err)
	}
	updated, err := store.Credential(ctx, loaded.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Registry Updated" || updated.EncryptedData != "encrypted-updated" {
		t.Fatalf("unexpected updated credential: %+v", updated)
	}

	if err := store.DeleteCredential(ctx, loaded.Id); err != nil {
		t.Fatal(err)
	}
	page, err = store.ListCredentials(ctx, projectId, 1, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Id != "credential-1" {
		t.Fatalf("unexpected credential page after delete: %+v", page)
	}
}

func TestCredentialReferencedByRepositories(t *testing.T) {
	database := openCredentialRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
	ctx := context.Background()

	referenced, err := store.CredentialReferencedByRepositories(ctx, "project-1", "credential-1")
	if err != nil {
		t.Fatal(err)
	}
	if !referenced {
		t.Fatal("expected credential to be referenced")
	}
	referenced, err = store.CredentialReferencedByRepositories(ctx, "project-2", "credential-1")
	if err != nil {
		t.Fatal(err)
	}
	if referenced {
		t.Fatal("expected credential not to be referenced by another project")
	}
}

func TestRepositoryListPaginationAndFilters(t *testing.T) {
	database := openRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
	ctx := context.Background()
	projectId := "project-1"
	if _, err := database.Exec(`UPDATE repository SET project_id = ?, created_at = '2024-03-16T00:00:00Z' WHERE id = 'repo-1'`, projectId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO repository (id, project_id, name, code, repository_url, variable_overrides, default_branch, created_at) VALUES ('repo-2', ?, 'Other', 'other', 'https://example.invalid/other.git', '[]', 'main', '2024-03-17T00:00:00Z')`, projectId); err != nil {
		t.Fatal(err)
	}

	page, err := store.ListRepositories(ctx, &projectId, 1, 1, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Page != 1 || page.PerPage != 1 || len(page.Items) != 1 || page.Items[0].Id != "repo-1" {
		t.Fatalf("unexpected repository page: %+v", page)
	}

	page, err = store.ListRepositories(ctx, &projectId, 1, 20, "example.invalid/other")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Code != "other" {
		t.Fatalf("unexpected repository URL search page: %+v", page)
	}
}

func TestPipelineRunAndArtifactListPaginationAndFilters(t *testing.T) {
	database := openRepositoryDB(t)
	defer func() { _ = database.Close() }()
	store := NewRepository(database, "sqlite")
	ctx := context.Background()
	projectId := "project-1"
	if _, err := database.Exec(`UPDATE pipeline_run SET project_id = ?, created_at = '2024-03-16T00:00:00Z' WHERE id = 'run-1'`, projectId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status, created_at) VALUES ('run-2', ?, 'repo-1', 'Repo', 'snapshot-1', 'template-2', 'Template Two', 1, 'manual', 'main', '{}', 'ran_to_completion', '2024-03-18T00:00:00Z')`, projectId); err != nil {
		t.Fatal(err)
	}
	from, to := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 17, 0, 0, 0, 0, time.UTC)
	runs, err := store.ListPipelineRuns(ctx, projectId, "repo-1", "template-1", &from, &to, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if runs.Total != 1 || len(runs.Items) != 1 || runs.Items[0].Id != "run-1" {
		t.Fatalf("unexpected pipeline run page: %+v", runs)
	}

	if _, err := database.Exec(`CREATE TABLE artifact (id TEXT PRIMARY KEY, project_id TEXT, pipeline_run_id TEXT, repository_id TEXT, repository_name TEXT, template_id TEXT, template_name TEXT, stage_name TEXT, type TEXT, name TEXT, path TEXT, created_at DATETIME NOT NULL DEFAULT (datetime('now'))) `); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at) VALUES ('artifact-1', ?, 'run-1', 'repo-1', 'Repo', 'template-1', 'Template', 'build', 'binary', 'app', 'dist/app', '2024-03-16T00:00:00Z')`, projectId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at) VALUES ('artifact-2', ?, 'run-1', 'repo-1', 'Repo', 'template-1', 'Template', 'test', 'binary', 'test-log', 'dist/test.log', '2024-03-16T00:01:00Z')`, projectId); err != nil {
		t.Fatal(err)
	}
	artifacts, err := store.ListArtifacts(ctx, projectId, "repo-1", "template-1", 1, 20, "test.log")
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Total != 1 || len(artifacts.Items) != 1 || artifacts.Items[0].Id != "artifact-2" {
		t.Fatalf("unexpected artifact page: %+v", artifacts)
	}
	runArtifacts, err := store.ListArtifactsByRun(ctx, &projectId, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(runArtifacts) != 2 || runArtifacts[0].Id != "artifact-1" || runArtifacts[1].Id != "artifact-2" {
		t.Fatalf("unexpected run artifacts: %+v", runArtifacts)
	}
}

func openRepositoryDB(t *testing.T) *sqlx.DB {
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

func openCredentialRepositoryDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database := openRepositoryDB(t)
	statements := []string{
		`CREATE TABLE credential (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, encrypted_data TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT (datetime('now')), project_id TEXT)`,
		`INSERT INTO credential (id, project_id, name, type, encrypted_data) VALUES ('credential-1', 'project-1', 'GitHub', 'github_token', 'encrypted-1')`,
		`UPDATE repository SET project_id = 'project-1', git_credential_id = 'credential-1' WHERE id = 'repo-1'`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return database
}
