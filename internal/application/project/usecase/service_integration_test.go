package projectsvc

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	"golang.org/x/crypto/ssh"
	_ "modernc.org/sqlite"

	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	credentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/credential"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

const (
	projectTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	projectTestFernetKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

func TestProjectServiceCreateUpdateMembersAndDeprecate(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.Create(ctx, projectTestUserId, testProjectCreateInput(t, "Second Project", "second"))
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "second" || !created.IsActive {
		t.Fatalf("unexpected created project: %+v", created)
	}
	loaded, err := service.LoadForUser(ctx, created.Id, projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(ctx, loaded, projectdto.SaveInput{Name: "Second Project Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Second Project Updated" || updated.Code != "second" {
		t.Fatalf("unexpected updated project: %+v", updated)
	}
	members, err := service.Members(ctx, updated.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Username != "admin" {
		t.Fatalf("unexpected project members: %+v", members)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO user (id, username, password_hash, status, oauth_provider, oauth_provider_id, email, auth_source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "member-user", "member", "hash", "enabled", "", "", "member@example.test", "password"); err != nil {
		t.Fatal(err)
	}
	members, err = service.AddMember(ctx, updated.Id, "member-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("unexpected members after add: %+v", members)
	}
	members, err = service.RemoveMember(ctx, updated.Id, "member-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("unexpected members after remove: %+v", members)
	}
	if err := service.Deprecate(ctx, updated, projectTestUserId); err == nil || apperror.StatusCode(err) != 400 {
		t.Fatalf("expected active environment deprecation validation error, got %v", err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE environment SET state = ? WHERE project_id = ?`, "disabled", updated.Id); err != nil {
		t.Fatal(err)
	}
	if err := service.Deprecate(ctx, updated, projectTestUserId); err != nil {
		t.Fatal(err)
	}
	deprecated, err := service.LoadForUser(ctx, updated.Id, projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	if deprecated.IsActive {
		t.Fatalf("expected deprecated project, got %+v", deprecated)
	}
}

func TestProjectServiceRejectsDuplicateCodeAndLastActiveDeprecation(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	if _, err := service.Create(ctx, projectTestUserId, testProjectCreateInput(t, "Duplicate", "default")); err == nil || apperror.StatusCode(err) != 409 {
		t.Fatalf("expected duplicate code conflict, got %v", err)
	}
	defaultProject, err := service.LoadForUser(ctx, "01KRRKK0K3T519ZQZES3M4QA9Z", projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Deprecate(ctx, defaultProject, projectTestUserId); err == nil || apperror.StatusCode(err) != 400 {
		t.Fatalf("expected last active project deprecation validation error, got %v", err)
	}
}

func TestProjectCreateRollsBackBootstrapThroughRequestTransaction(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	input := testProjectCreateInput(t, "Rollback", "rollback")
	input.Environment.Platform = "unsupported"
	input.Environment.DeploymentSSHKeyName = "rollback-deploy-key"

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(databasetx.Middleware(database))
	router.POST("/api/project", func(c *gin.Context) {
		if _, err := service.Create(c.Request.Context(), projectTestUserId, input); err != nil {
			transportresponse.WriteError(c, err)
			return
		}
		c.Status(http.StatusCreated)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/project", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM project WHERE code = ?`, "rollback", 0)
	assertCount(t, database, `SELECT COUNT(*) FROM credential WHERE name = ?`, "rollback-deploy-key", 0)
	assertCount(t, database, `SELECT COUNT(*) FROM environment WHERE code = ?`, "rollback", 0)
}
func newProjectIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	projectStore := projectrepo.NewRepository(database)
	credentialService := credentialsvc.New(projectStore, credentialrepo.NewRepository(database), projectTestFernetKey)
	environmentService := environmentsvc.New(environmentrepo.NewRepository(database), projectStore, credentialService, credentialService, nil)
	return New(projectStore, userrepo.NewRepository(database), environmentrepo.NewRepository(database), credentialService, environmentService), database
}

func testProjectCreateInput(t *testing.T, name string, code string) projectdto.CreateInput {
	t.Helper()
	return projectdto.CreateInput{
		Name: name,
		Code: code,
		Environment: projectdto.EnvironmentCreateInput{
			State:                   "active",
			Platform:                "linux",
			Host:                    "192.0.2.10",
			Port:                    22,
			Username:                "deploy",
			WorkspaceRoot:           "/srv/pomelo-orbit",
			DeploymentSSHKeyName:    "project-deploy-key",
			DeploymentSSHPrivateKey: testDeploymentSSHPrivateKey(t),
			HostKeyFingerprint:      "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		},
	}
}

func testDeploymentSSHPrivateKey(t *testing.T) string {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "project-service-test")
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(block))
}

func assertCount(t *testing.T, database *sql.DB, query string, value string, want int) {
	t.Helper()
	var got int
	if err := database.QueryRow(query, value).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("query %q with %q returned %d, want %d", query, value, got, want)
	}
}
