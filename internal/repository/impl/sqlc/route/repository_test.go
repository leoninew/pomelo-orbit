package routerepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	databaseinfra "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
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
	const projectId = "01KROUTEREPOSITORYTEST0001"
	if _, err := database.ExecContext(ctx, "INSERT INTO project (id, name, code) VALUES (?, ?, ?)", projectId, "Route Test", "route-test"); err != nil {
		t.Fatal(err)
	}

	repository := NewRepository(database)
	for _, route := range []model.Route{
		{Id: "route-1", ProjectId: stringPtr(projectId), Name: "match alpha", Protocol: "http", Domain: "alpha.example.test", PathPrefix: "/", TargetUrl: "http://alpha.example.test", CertType: "manual", AcmeChallenge: "http"},
		{Id: "route-2", ProjectId: stringPtr(projectId), Name: "match beta", Protocol: "http", Domain: "beta.example.test", PathPrefix: "/", TargetUrl: "http://beta.example.test", CertType: "manual", AcmeChallenge: "http"},
		{Id: "route-3", ProjectId: stringPtr(projectId), Name: "other", Protocol: "http", Domain: "other.example.test", PathPrefix: "/", TargetUrl: "http://other.example.test", CertType: "manual", AcmeChallenge: "http"},
	} {
		if err := repository.CreateRoute(ctx, route); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repository.ListRoutes(ctx, projectId, 1, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 3 || len(all.Items) != 2 || all.Items[0].Id != "route-3" || all.Items[1].Id != "route-2" {
		t.Fatalf("unfiltered routes = %+v, want first page of three routes", all)
	}

	matched, err := repository.ListRoutes(ctx, projectId, 1, 10, "match")
	if err != nil {
		t.Fatal(err)
	}
	if matched.Total != 2 || len(matched.Items) != 2 {
		t.Fatalf("matched routes = %+v, want two matching routes", matched)
	}

	secondPage, err := repository.ListRoutes(ctx, projectId, 2, 1, "match")
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

func TestRouteViewsKeepCustomAndDeletedTargets(t *testing.T) {
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
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`,
		`INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'API', 'api', 'standard', 'project-1')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'unpublished')`,
		`INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('service-1', 'project-1', 'app-1', 'version-1', 'api-main', 'stopped')`,
		`INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('service-2', 'project-1', 'app-1', 'version-1', 'api-worker', 'stopped')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewRepository(database)
	for _, route := range []model.Route{
		{Id: "route-1", ServiceId: stringPtr("service-1"), TargetUrl: "api-main/api/http8080"},
		{Id: "route-2", ServiceId: stringPtr("service-2"), TargetUrl: "api-worker/api/http8080"},
		{Id: "route-3", TargetUrl: "http://example.test:8080"},
	} {
		route.ProjectId, route.Name, route.Protocol = stringPtr("project-1"), route.Id, "http"
		route.Domain, route.PathPrefix, route.CertType, route.AcmeChallenge = route.Id+".test", "/", "manual", "http"
		if err := store.CreateRoute(ctx, route); err != nil {
			t.Fatal(err)
		}
	}
	checkViews := func(deleted bool) {
		t.Helper()
		page, err := store.ListRoutes(ctx, "project-1", 1, 10, "")
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 3 || len(page.Items) != 3 {
			t.Fatalf("routes = %+v, want all three targets", page)
		}
		for _, item := range page.Items {
			view, err := store.RouteView(ctx, "project-1", item.Id)
			if err != nil {
				t.Fatal(err)
			}
			for _, target := range []model.RouteView{item, view} {
				switch target.Id {
				case "route-1":
					if target.ServiceId == nil || *target.ServiceId != "service-1" || target.TargetUrl != "api-main/api/http8080" {
						t.Fatalf("managed target identity = %+v", target)
					}
					if deleted {
						if target.ServiceCode != nil {
							t.Fatalf("deleted service code = %q", *target.ServiceCode)
						}
					} else if target.ServiceCode == nil || *target.ServiceCode != "api-main" {
						t.Fatalf("main service target = %+v", target)
					}
				case "route-2":
					if target.ServiceCode == nil || *target.ServiceCode != "api-worker" {
						t.Fatalf("worker service target = %+v", target)
					}
				case "route-3":
					if target.ServiceId != nil || target.ServiceCode != nil || target.TargetUrl != "http://example.test:8080" {
						t.Fatalf("custom target = %+v", target)
					}
				}
			}
		}
	}
	checkViews(false)
	if _, err := database.ExecContext(ctx, `DELETE FROM service WHERE id = 'service-1'`); err != nil {
		t.Fatal(err)
	}
	checkViews(true)
	if _, err := store.RouteView(ctx, "other-project", "route-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("foreign project route view error = %v, want not found", err)
	}
}
