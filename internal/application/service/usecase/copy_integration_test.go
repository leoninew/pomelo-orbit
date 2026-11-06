//go:build !unit

package servicesvc

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
	_ "modernc.org/sqlite"
)

func copyDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`PRAGMA foreign_keys = ON`,
		`INSERT INTO user (id, username, password_hash, email) VALUES ('user', 'copy-user', '', 'copy@example.test')`,
		`INSERT INTO project (id, name, code) VALUES ('source', 'Source', 'source'), ('target', 'Target', 'target')`,
		`INSERT INTO project_member (project_id, user_id) VALUES ('source', 'user'), ('target', 'user')`,
		`INSERT INTO application (id, project_id, name, code, kind) VALUES ('source-app', 'source', 'Postgres', 'postgres', 'standard'), ('existing-app', 'target', 'Existing', 'existing', 'standard')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'source-app', 'v1', 'published'), ('existing-version', 'existing-app', 'existing', 'unpublished')`,
		`INSERT INTO version (id, application_id, label, status, created_from_version_id) VALUES ('version-2', 'source-app', 'v2', 'unpublished', 'version-1')`,
		`INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('existing-service', 'target', 'existing-app', 'existing-version', 'existing', 'stopped')`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if err := environmentrepo.NewRepository(database).CreateEnvironment(ctx, model.Environment{
		Id: "target-environment", ProjectId: "target", Code: "target", TargetType: "ssh",
		WorkspaceRoot: "/srv/orbit", TargetRevision: 2,
		SSH: &model.EnvironmentSSHTarget{Platform: "linux", Host: "example.test", Port: 22, Username: "orbit"},
	}); err != nil {
		t.Fatal(err)
	}
	applications := applicationrepo.NewRepository(database)
	for _, versionId := range []string{"version-1", "version-2"} {
		component := model.VersionComponent{
			Id: versionId + "-component", VersionId: versionId, Name: "db", Image: "postgres:18", PullPolicy: "missing",
			Env:    []model.VersionComponentEnv{{Key: "PGPORT", Value: "${PORT}"}},
			Mounts: []model.VersionComponentMount{{SourceType: "file", Source: "config/db.conf", Target: "/etc/db.conf", Content: "port=5432", Mode: "0644"}},
		}
		if err := applications.CreateVersionComponent(ctx, "source", component); err != nil {
			t.Fatal(err)
		}
	}
	services := servicerepo.NewRepository(database)
	service := model.Service{Id: "source-service", ProjectId: "source", ApplicationId: "source-app", VersionId: "version-1", Code: "postgres-default", Status: "running"}
	component := model.ServiceComponent{Id: "source-component", ServiceId: service.Id, SourceVersionComponentId: "version-1-component", ComponentName: "db", Status: "active", Entrypoint: []string{}, User: stringPtr(""), Env: []model.ServiceComponentEnv{{Key: "PGPORT", Value: stringPtr("${PORT}"), State: model.ServiceComponentOverlayOverride}}}
	if err := services.CreateServiceWithComponents(ctx, "source", service, []model.ServiceComponent{component}); err != nil {
		t.Fatal(err)
	}
	if err := services.ReplaceServiceEnv(ctx, "source", service.Id, []model.ServiceEnv{{Key: "PORT", Value: "5432"}}); err != nil {
		t.Fatal(err)
	}
	if err := services.UpdateServiceDeploymentDirectory(ctx, "source", service.Id, `E:\orbit\postgres`, 7); err != nil {
		t.Fatal(err)
	}
	return database
}

func newDatabaseCopy(database *sql.DB, services repository.ServiceStore) (CopyService, Service) {
	projects, applications := projectrepo.NewRepository(database), applicationrepo.NewRepository(database)
	service := New(projects, applications, services, nil)
	environments := environmentsvc.New(environmentrepo.NewRepository(database), projects, nil, "", nil, nil)
	return NewCopyService(service, applicationsvc.New(projects, applications), environments), service
}

func TestCopyServicePersistsIndependentConfiguration(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest", true: "all"}[all], func(t *testing.T) {
			database := copyDatabase(t)
			services := servicerepo.NewRepository(database)
			copier, service := newDatabaseCopy(database, services)
			ctx := context.Background()
			source, err := service.ServiceDefinitionForUser(ctx, "user", "source", "source-service")
			if err != nil {
				t.Fatal(err)
			}
			var copied model.Service
			err = tx.NewTransactionRunner(database).RunInTransaction(ctx, func(ctx context.Context) error {
				var err error
				copied, err = copier.Copy(ctx, "user", "source", "source-service", servicedto.ServiceCopyInput{
					TargetProjectId: "target", CopyAllVersions: all, DeploymentDirectory: "/custom/postgres", EnvironmentTargetRevision: 2,
				})
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			target, err := service.ServiceDefinitionForUser(ctx, "user", "target", copied.Id)
			if err != nil {
				t.Fatal(err)
			}
			applications := applicationrepo.NewRepository(database)
			versions, err := applications.ListVersions(ctx, "target", copied.ApplicationId)
			if err != nil {
				t.Fatal(err)
			}
			wantCount, wantLabel := 1, "v2"
			if all {
				wantCount, wantLabel = 2, "v1"
			}
			version, err := applications.Version(ctx, "target", copied.VersionId)
			if err != nil {
				t.Fatal(err)
			}
			if len(versions) != wantCount || versions[0].Label != "v2" || version.Label != wantLabel {
				t.Fatal("wrong copied version scope, ordering or service binding")
			}
			if all {
				if versions[0].CreatedFromVersionId == nil || *versions[0].CreatedFromVersionId != versions[1].Id {
					t.Fatal("version lineage was not remapped")
				}
			} else if versions[0].CreatedFromVersionId != nil {
				t.Fatal("latest copy retained excluded parent")
			}
			declarations, err := applications.VersionComponentsByVersion(ctx, "target", copied.VersionId)
			if err != nil {
				t.Fatal(err)
			}
			if len(target.Components) != 1 || target.Components[0].SourceVersionComponentId != declarations[0].Id || declarations[0].Env[0].Value != "${PORT}" || declarations[0].Mounts[0].Content != "port=5432" {
				t.Fatal("component configuration or ownership changed")
			}
			if !reflect.DeepEqual(source.Env, target.Env) || !reflect.DeepEqual(source.Components[0].Env, target.Components[0].Env) || target.Components[0].Entrypoint == nil || target.Components[0].User == nil || *target.Components[0].User != "" {
				t.Fatal("environment or explicit overlays changed")
			}
			if target.Service.Status != "stopped" || target.Service.DirectoryTargetRevision != 2 || target.Service.DeploymentDirectory != "/custom/postgres" || target.Service.RuntimeDirectory != "" {
				t.Fatal("target service runtime state is incorrect")
			}
			after, err := service.ServiceDefinitionForUser(ctx, "user", "source", "source-service")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(source, after) {
				t.Fatal("source configuration changed")
			}
		})
	}
}

type copyFailingEnvStore struct{ repository.ServiceStore }

func (copyFailingEnvStore) ReplaceServiceEnv(context.Context, string, string, []model.ServiceEnv) error {
	return errors.New("injected environment write failure")
}

func TestCopyServiceRollsBackAllResourcesAndRejectsUnauthorizedCopies(t *testing.T) {
	for _, test := range []struct {
		name, statement string
		kind            apperror.Kind
		failEnv         bool
	}{
		{name: "environment write failure", kind: apperror.KindInternal, failEnv: true},
		{name: "source membership", statement: `DELETE FROM project_member WHERE project_id = 'source'`, kind: apperror.KindForbidden},
		{name: "target membership", statement: `DELETE FROM project_member WHERE project_id = 'target'`, kind: apperror.KindForbidden},
		{name: "service code conflict", statement: `UPDATE service SET code = 'postgres-default' WHERE id = 'existing-service'`, kind: apperror.KindConflict},
		{name: "application name conflict", statement: `UPDATE application SET name = 'Postgres' WHERE id = 'existing-app'`, kind: apperror.KindConflict},
		{name: "application code conflict", statement: `UPDATE application SET code = 'postgres' WHERE id = 'existing-app'`, kind: apperror.KindConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := copyDatabase(t)
			if test.statement != "" {
				if _, err := database.Exec(test.statement); err != nil {
					t.Fatal(err)
				}
			}
			var services repository.ServiceStore = servicerepo.NewRepository(database)
			if test.failEnv {
				services = copyFailingEnvStore{ServiceStore: services}
			}
			copier, service := newDatabaseCopy(database, services)
			ctx := context.Background()
			before, err := service.ServiceDefinitionForUser(ctx, "user", "target", "existing-service")
			if test.kind != apperror.KindForbidden && err != nil {
				t.Fatal(err)
			}
			err = tx.NewTransactionRunner(database).RunInTransaction(ctx, func(ctx context.Context) error {
				_, err := copier.Copy(ctx, "user", "source", "source-service", servicedto.ServiceCopyInput{
					TargetProjectId: "target", CopyAllVersions: true, DeploymentDirectory: "/custom/postgres", EnvironmentTargetRevision: 2,
				})
				return err
			})
			if !apperror.IsKind(err, test.kind) {
				t.Fatalf("expected %v, got %v", test.kind, err)
			}
			for table, want := range map[string]int{"application": 2, "version": 3, "version_component": 2, "service": 2, "service_component": 1, "service_env": 1} {
				var count int
				if err := database.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != want {
					t.Fatalf("%s count = %d, want %d", table, count, want)
				}
			}
			if test.kind != apperror.KindForbidden {
				after, err := service.ServiceDefinitionForUser(ctx, "user", "target", "existing-service")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatal("existing target service changed")
				}
			}
		})
	}
}
