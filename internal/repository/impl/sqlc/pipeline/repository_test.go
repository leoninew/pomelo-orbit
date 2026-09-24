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
	`, "pipeline-1", nil, "template", "Build image", "", "[]", 1, createdAt, createdAt); err != nil {
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

func TestListPipelinesScopesGlobalTemplatesAndApplicationPipelines(t *testing.T) {
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
	for _, values := range [][]any{
		{"template-1", nil, "template", "Shared build", "", "[]"},
		{"application-1", "project-1", "application", "Project one build", "", "[]"},
		{"application-2", "project-2", "application", "Project two build", "", "[]"},
	} {
		if _, err := database.Exec(`
			INSERT INTO pipeline (id, project_id, kind, name, description, variable_declarations, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
		`, values[0], values[1], values[2], values[3], values[4], values[5], createdAt, createdAt); err != nil {
			t.Fatal(err)
		}
	}

	store := NewRepository(database)
	for _, projectId := range []string{"project-1", "project-2"} {
		result, err := store.ListPipelines(context.Background(), projectId, "template", 1, 20, "")
		if err != nil {
			t.Fatalf("list global templates for %s: %v", projectId, err)
		}
		if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Id != "template-1" || result.Items[0].ProjectId != nil {
			t.Fatalf("global templates for %s = %+v", projectId, result)
		}
	}

	for _, testCase := range []struct {
		projectId  string
		pipelineId string
	}{
		{projectId: "project-1", pipelineId: "application-1"},
		{projectId: "project-2", pipelineId: "application-2"},
	} {
		result, err := store.ListPipelines(context.Background(), testCase.projectId, "application", 1, 20, "")
		if err != nil {
			t.Fatalf("list application pipelines for %s: %v", testCase.projectId, err)
		}
		if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Id != testCase.pipelineId {
			t.Fatalf("application pipelines for %s = %+v", testCase.projectId, result)
		}
	}
}

func TestListPipelineStageTemplatesScopesGlobally(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if _, err := database.Exec(`
		CREATE TABLE pipeline_stage (
			id TEXT PRIMARY KEY, project_id TEXT, kind TEXT NOT NULL, pipeline_id TEXT,
			name TEXT NOT NULL, image TEXT NOT NULL, script TEXT NOT NULL,
			description TEXT NOT NULL, version INTEGER, source_template_stage_id TEXT,
			source_template_stage_name TEXT, source_template_stage_version INTEGER,
			source_template_stage_description TEXT, artifacts TEXT, depends_on TEXT,
			sort_order INTEGER, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
	createdAt := time.Now().UTC()
	if _, err := database.Exec(`
		INSERT INTO pipeline_stage (
			id, project_id, kind, name, image, script, description, version, artifacts, created_at, updated_at
		) VALUES (?, ?, 'template', ?, ?, ?, ?, ?, ?, ?, ?)
	`, "stage-template", nil, "Shared build", "docker:27", "docker build .", "", 1, "[]", createdAt, createdAt); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
		INSERT INTO pipeline_stage (
			id, project_id, kind, pipeline_id, name, image, script, description, artifacts, depends_on, sort_order, created_at, updated_at
		) VALUES (?, ?, 'application', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "stage-application", "project-1", "application-1", "Private build", "docker:27", "docker build .", "", "[]", "[]", 0, createdAt, createdAt); err != nil {
		t.Fatal(err)
	}

	store := NewRepository(database)
	result, err := store.ListPipelineStageTemplates(context.Background(), "project-2", 1, 20, "")
	if err != nil {
		t.Fatalf("list global stage templates: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Id != "stage-template" || result.Items[0].ProjectId != "" {
		t.Fatalf("global stage templates = %+v", result)
	}

	stage, err := store.PipelineStageTemplate(context.Background(), "project-2", "stage-template")
	if err != nil {
		t.Fatalf("load global stage template: %v", err)
	}
	if stage.Id != "stage-template" || stage.ProjectId != "" {
		t.Fatalf("global stage template = %+v", stage)
	}
}
