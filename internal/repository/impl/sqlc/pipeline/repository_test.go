package pipelinerepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestListPipelinesBindsNamedFilterAndPaginationParameters(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline (
			id TEXT PRIMARY KEY, project_id TEXT, kind TEXT NOT NULL,
			source_pipeline_id TEXT, source_template_name TEXT, source_template_version INTEGER,
			application_id TEXT, application_name TEXT, repository_id TEXT, repository_name TEXT,
			version_fork_strategy TEXT, fixed_version_id TEXT, fixed_version_label TEXT,
			name TEXT NOT NULL, description TEXT NOT NULL, variable_declarations TEXT NOT NULL,
			version INTEGER NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Now().UTC()
	if _, err := database.Exec(`
		INSERT INTO pipeline (
			id, project_id, kind, name, description, variable_declarations, version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "pipeline-1", "project-1", "template", "Build image", "", "[]", 1, createdAt, createdAt); err != nil {
		t.Fatal(err)
	}

	result, err := NewRepository(database).ListPipelines(context.Background(), "project-1", "template", 1, 20, "image")
	if err != nil {
		t.Fatalf("list pipelines: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Id != "pipeline-1" {
		t.Fatalf("pipelines=%+v", result)
	}
}
