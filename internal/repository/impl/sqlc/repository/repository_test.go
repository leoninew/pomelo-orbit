package vcsrepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func openRepositoryTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, "sqlite"); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestListRepositoriesUsesNamedSQLiteParameters(t *testing.T) {
	database := openRepositoryTestDatabase(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)
	for _, item := range []model.Repository{
		{Id: "repository-1", Name: "Repository One", Code: "repository-one", RepositoryUrl: "https://example.test/one.git", VariableOverrides: "[]", DefaultBranch: "main"},
		{Id: "repository-2", Name: "Repository Two", Code: "repository-two", RepositoryUrl: "https://example.test/two.git", VariableOverrides: "[]", DefaultBranch: "main"},
	} {
		if err := repo.CreateRepository(ctx, item); err != nil {
			t.Fatal(err)
		}
	}

	filtered, err := repo.ListRepositories(ctx, 1, 1, "One")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Id != "repository-1" {
		t.Fatalf("unexpected filtered page: %+v", filtered)
	}

	all, err := repo.ListRepositories(ctx, 1, 100, "Repository")
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("unexpected global page: %+v", all)
	}
}

func TestRepositoryCodeIsGloballyUnique(t *testing.T) {
	database := openRepositoryTestDatabase(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)
	if err := repo.CreateRepository(ctx, model.Repository{Id: "repository-1", Name: "First", Code: "shared-code", RepositoryUrl: "https://example.test/first.git", VariableOverrides: "[]", DefaultBranch: "main"}); err != nil {
		t.Fatal(err)
	}
	item, err := repo.RepositoryByCode(ctx, "shared-code")
	if err != nil || item.Id != "repository-1" {
		t.Fatalf("repository by code = %+v, %v", item, err)
	}
	err = repo.CreateRepository(ctx, model.Repository{
		Id: "repository-2", Name: "Duplicate", Code: "shared-code",
		RepositoryUrl: "https://example.test/duplicate.git", VariableOverrides: "[]", DefaultBranch: "main",
	})
	if err == nil {
		t.Fatal("duplicate code must be rejected globally")
	}
}

func TestRepositoryHasBoundPipelinesAcrossProjects(t *testing.T) {
	database := openRepositoryTestDatabase(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	for _, project := range []struct {
		id   string
		code string
	}{
		{id: "project-1", code: "project-one"},
		{id: "project-2", code: "project-two"},
	} {
		if _, err := database.ExecContext(ctx, "INSERT INTO project (id, name, code) VALUES (?, ?, ?)", project.id, project.code, project.code); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO pipeline (id, project_id, kind, repository_id, name)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`, "pipeline-1", "project-1", "application", "repository-1", "Build one", "pipeline-2", "project-2", "application", "repository-1", "Build two"); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(database)
	bound, err := repo.RepositoryHasBoundPipelines(ctx, "repository-1")
	if err != nil {
		t.Fatalf("check bound pipelines: %v", err)
	}
	if !bound {
		t.Fatal("expected repository to have a bound pipeline")
	}

	bound, err = repo.RepositoryHasBoundPipelines(ctx, "repository-2")
	if err != nil {
		t.Fatalf("check unrelated repository: %v", err)
	}
	if bound {
		t.Fatal("unrelated repository must not be reported as bound")
	}

	if _, err := database.ExecContext(ctx, "DELETE FROM pipeline WHERE id = ?", "pipeline-1"); err != nil {
		t.Fatal(err)
	}
	bound, err = repo.RepositoryHasBoundPipelines(ctx, "repository-1")
	if err != nil {
		t.Fatalf("check other project: %v", err)
	}
	if !bound {
		t.Fatal("another project binding must still be counted")
	}
}
