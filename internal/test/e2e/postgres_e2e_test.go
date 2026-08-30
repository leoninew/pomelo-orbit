package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	applicationsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/application"
	deploymentsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/deployment"
	pipelinerunsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/pipeline_run"
	repositoriessqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/repository"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
)

const (
	postgresE2EConfigEnv = "BACKEND_GO_POSTGRES_E2E_CONFIG"
	postgresE2EDsnEnv    = "BACKEND_GO_POSTGRES_E2E_DSN"
	postgresAdminID      = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	postgresProjectID    = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestPostgreSQLMigrationE2E(t *testing.T) {
	cfg := loadPostgreSQLE2EConfig(t)
	if cfg.Database.Driver != config.DatabaseDriverPostgres {
		t.Fatalf("expected postgres config, got %s", cfg.Database.Driver)
	}

	database := openPostgresE2EDatabase(t, cfg)
	defer func() { _ = database.Close() }()

	runMigrationE2E(t, cfg)
	testPostgresRepositoriesAndTransaction(t, database)
	testPostgresOptionalQueryFilters(t, database)
}

func loadPostgreSQLE2EConfig(t *testing.T) config.Config {
	t.Helper()
	configPath := os.Getenv(postgresE2EConfigEnv)
	if configPath != "" {
		configPath, err := filepath.Abs(configPath)
		if err != nil {
			t.Fatal(err)
		}
		environmentConfig, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		return loadE2EConfig(t, environmentConfig)
	}

	dsn := os.Getenv(postgresE2EDsnEnv)
	if dsn == "" {
		t.Skip("set BACKEND_GO_POSTGRES_E2E_CONFIG or BACKEND_GO_POSTGRES_E2E_DSN to run PostgreSQL e2e")
	}
	return loadE2EConfig(t, []byte(fmt.Sprintf(`database:
  driver: postgres
  postgres:
    dsn: %q
`, dsn)))
}

func openPostgresE2EDatabase(t *testing.T, cfg config.Config) *sql.DB {
	t.Helper()
	database, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func testPostgresRepositoriesAndTransaction(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()

	user, err := userrepo.NewRepository(database).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("load seeded admin through sqlc: %v", err)
	}
	if user.Id != postgresAdminID || user.Username != "admin" {
		t.Fatalf("unexpected seeded admin: id=%q username=%q", user.Id, user.Username)
	}

	projects, err := projectrepo.NewRepository(database).ListActiveProjectsByMember(ctx, postgresAdminID)
	if err != nil {
		t.Fatalf("list active projects through sqlc: %v", err)
	}
	if len(projects) != 1 || projects[0].Id != postgresProjectID || !projects[0].IsActive {
		t.Fatalf("unexpected active projects: %+v", projects)
	}

	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE "user" SET status = ? WHERE id = ?`, "disabled", postgresAdminID); err != nil {
		_ = tx.Rollback()
		t.Fatalf("execute transaction with postgres placeholders: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	var status string
	if err := database.QueryRowContext(ctx, `SELECT status FROM "user" WHERE id = ?`, postgresAdminID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "enabled" {
		t.Fatalf("rolled back user status = %q, want enabled", status)
	}
}

func testPostgresOptionalQueryFilters(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()

	repositoryQueries := repositoriessqlc.New(database)
	if _, err := repositoryQueries.CountRepositories(ctx, repositoriessqlc.CountRepositoriesParams{}); err != nil {
		t.Fatalf("count repositories without optional filters: %v", err)
	}
	if _, err := repositoryQueries.CountRepositories(ctx, repositoriessqlc.CountRepositoriesParams{
		ProjectID:     sql.NullString{String: postgresProjectID, Valid: true},
		SearchPattern: sql.NullString{String: "%gateway%", Valid: true},
	}); err != nil {
		t.Fatalf("count repositories with optional text filters: %v", err)
	}
	if _, err := pipelinerunsqlc.New(database).CountPipelineRuns(ctx, pipelinerunsqlc.CountPipelineRunsParams{}); err != nil {
		t.Fatalf("count pipeline runs without optional filters: %v", err)
	}
	if _, err := applicationsqlc.New(database).CountApplications(ctx, applicationsqlc.CountApplicationsParams{}); err != nil {
		t.Fatalf("count applications without optional filters: %v", err)
	}

	deploymentQueries := deploymentsqlc.New(database)
	if _, err := deploymentQueries.CountDeployments(ctx, deploymentsqlc.CountDeploymentsParams{
		ProjectID: sql.NullString{String: postgresProjectID, Valid: true},
	}); err != nil {
		t.Fatalf("count deployments without optional filters: %v", err)
	}

	now := time.Now().UTC()
	if _, err := pipelinerunsqlc.New(database).CountPipelineRuns(ctx, pipelinerunsqlc.CountPipelineRunsParams{
		FromAt: sql.NullTime{Time: now.Add(-time.Hour), Valid: true},
		ToAt:   sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		t.Fatalf("count pipeline runs with optional time filters: %v", err)
	}
	if _, err := deploymentQueries.CountDeployments(ctx, deploymentsqlc.CountDeploymentsParams{
		ProjectID: sql.NullString{String: postgresProjectID, Valid: true},
		DateFrom:  sql.NullTime{Time: now.Add(-time.Hour), Valid: true},
		DateTo:    sql.NullTime{Time: now, Valid: true},
	}); err != nil {
		t.Fatalf("count deployments with optional time filters: %v", err)
	}
}
