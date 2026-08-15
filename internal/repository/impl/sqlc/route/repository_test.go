package routerepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	databaseinfra "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestListRoutesBindsSearchAndPaginationForSQLite(t *testing.T) {
	t.Parallel()

	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := databaseinfra.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	const projectID = "01KRRKK0K3T519ZQZES3M4QA9Z"
	if _, err := database.ExecContext(ctx, "INSERT INTO project (id, name, code) VALUES (?, ?, ?)", projectID, "Route Test", "route-test"); err != nil {
		t.Fatal(err)
	}

	repository := NewRepository(database)
	for _, route := range []model.Route{
		{Id: "route-1", ProjectId: stringPtr(projectID), Name: "match alpha", Protocol: "http", Domain: "alpha.example.test", PathPrefix: "/", TargetUrl: "http://alpha.example.test", CertType: "manual"},
		{Id: "route-2", ProjectId: stringPtr(projectID), Name: "match beta", Protocol: "http", Domain: "beta.example.test", PathPrefix: "/", TargetUrl: "http://beta.example.test", CertType: "manual"},
		{Id: "route-3", ProjectId: stringPtr(projectID), Name: "other", Protocol: "http", Domain: "other.example.test", PathPrefix: "/", TargetUrl: "http://other.example.test", CertType: "manual"},
	} {
		if err := repository.CreateRoute(ctx, route); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repository.ListRoutes(ctx, projectID, 1, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 3 || len(all.Items) != 2 || all.Items[0].Id != "route-3" || all.Items[1].Id != "route-2" {
		t.Fatalf("unfiltered routes = %+v, want first page of three routes", all)
	}

	matched, err := repository.ListRoutes(ctx, projectID, 1, 10, "match")
	if err != nil {
		t.Fatal(err)
	}
	if matched.Total != 2 || len(matched.Items) != 2 {
		t.Fatalf("matched routes = %+v, want two matching routes", matched)
	}

	secondPage, err := repository.ListRoutes(ctx, projectID, 2, 1, "match")
	if err != nil {
		t.Fatal(err)
	}
	if secondPage.Total != 2 || len(secondPage.Items) != 1 || secondPage.Items[0].Id != "route-1" {
		t.Fatalf("second page = %+v, want route-1", secondPage)
	}
}

func stringPtr(value string) *string {
	return &value
}
