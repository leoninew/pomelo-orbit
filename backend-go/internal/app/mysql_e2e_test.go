package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/repository"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
	transporthttp "backend/internal/transport/http"
)

func TestMySQLE2E(t *testing.T) {
	configPath := os.Getenv("BACKEND_GO_E2E_CONFIG")
	if configPath == "" {
		t.Skip("set BACKEND_GO_E2E_CONFIG to run MySQL e2e test")
	}
	moduleRoot := filepath.Join("..", "..")
	if err := os.Chdir(moduleRoot); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Driver != config.DatabaseDriverMySQL {
		t.Fatalf("expected mysql config, got %s", cfg.Database.Driver)
	}

	database, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	migrator := db.NewMigrator(database, cfg.Database.Driver)
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
	statuses, err := migrator.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 6 {
		t.Fatalf("expected 6 migrations, got %d", len(statuses))
	}
	for _, migration := range statuses {
		if !migration.Applied {
			t.Fatalf("expected migration %s applied", migration.Filename)
		}
	}

	var userCount int
	if err := database.Get(&userCount, "SELECT COUNT(*) FROM user WHERE username = ?", "admin"); err != nil {
		t.Fatal(err)
	}
	if userCount != 1 {
		t.Fatalf("expected admin user, got %d", userCount)
	}

	taskRepo := taskrepo.NewRepository(database, cfg.Database.Driver)
	taskId := repository.NewId()
	ctx := context.Background()
	if err := taskRepo.Enqueue(ctx, taskId, status.TaskTypeCIPipelineRunExecute, `{"pipeline_run_id":"run-1"}`, 1); err != nil {
		t.Fatal(err)
	}
	queued, err := taskRepo.FindById(ctx, taskId)
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != status.TaskPending {
		t.Fatalf("unexpected task status: %s", queued.Status)
	}

	cfg.Turnstile.Enabled = false
	store := repository.NewStore(database, cfg.Database.Driver)
	server := transporthttp.New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), store, taskRepo, cfg.Worker.MaxAttempts)
	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(loginRecorder.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	if token.AccessToken == "" || token.TokenType != "bearer" {
		t.Fatalf("unexpected token response: %+v", token)
	}
}
