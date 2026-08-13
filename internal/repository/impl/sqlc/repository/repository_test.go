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

	repo := NewRepository(database)
	baseline, err := repo.ListRepositories(ctx, nil, 1, 100, "")
	if err != nil {
		t.Fatal(err)
	}

	projectOne := "project-1"
	projectTwo := "project-2"
	for _, item := range []model.Repository{
		{Id: "repository-1", ProjectId: &projectOne, Name: "Repository One", Code: "repository-one", RepositoryUrl: "https://example.test/one.git", VariableOverrides: "[]", DefaultBranch: "main"},
		{Id: "repository-2", ProjectId: &projectTwo, Name: "Repository Two", Code: "repository-two", RepositoryUrl: "https://example.test/two.git", VariableOverrides: "[]", DefaultBranch: "main"},
	} {
		if err := repo.CreateRepository(ctx, item); err != nil {
			t.Fatal(err)
		}
	}

	filtered, err := repo.ListRepositories(ctx, &projectOne, 1, 1, "One")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Id != "repository-1" {
		t.Fatalf("unexpected filtered page: %+v", filtered)
	}

	all, err := repo.ListRepositories(ctx, nil, 1, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != baseline.Total+2 || len(all.Items) != baseline.Total+2 {
		t.Fatalf("unexpected unfiltered page: %+v", all)
	}
}
