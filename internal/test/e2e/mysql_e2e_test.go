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

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	"gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

func TestMySQLE2E(t *testing.T) {
	configPath := os.Getenv("BACKEND_GO_E2E_CONFIG")
	if configPath == "" {
		t.Skip("set BACKEND_GO_E2E_CONFIG to run MySQL e2e test")
	}
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join("..", "..")
	defaultConfig, err := os.ReadFile(filepath.Join(moduleRoot, config.DefaultConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	envConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	configDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configDir, filepath.Dir(config.DefaultConfigFile)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, config.DefaultConfigFile), defaultConfig, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POMELO_ORBIT_APP__ENV", "e2e")
	if err := os.WriteFile(filepath.Join(configDir, config.EnvConfigFile("e2e")), envConfig, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(configDir); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load()
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

	if err := db.MigrateUp(database, cfg.Database.Driver); err != nil {
		t.Fatal(err)
	}
	version, err := db.ReadMigrationVersion(database, cfg.Database.Driver)
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 32 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}

	var userCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM user WHERE username = ?", "admin").Scan(&userCount); err != nil {
		t.Fatal(err)
	}
	if userCount != 1 {
		t.Fatalf("expected admin user, got %d", userCount)
	}

	taskRepo := taskrepo.NewRepository(database)
	taskId := idutil.NewId()
	ctx := context.Background()
	if err := taskRepo.Enqueue(ctx, taskId, status.TaskTypePipelineRunExecute, `{"pipeline_run_id":"run-1"}`, 1); err != nil {
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
	server := bootstrap.NewHTTPServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), database, taskRepo)
	csrfRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(csrfRecorder, httptest.NewRequest(http.MethodGet, "/api/auth/csrf-token", nil))
	if csrfRecorder.Code != http.StatusOK {
		t.Fatalf("expected csrf status 200, got %d: %s", csrfRecorder.Code, csrfRecorder.Body.String())
	}
	var csrfResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(csrfRecorder.Body).Decode(&csrfResp); err != nil {
		t.Fatal(err)
	}
	if csrfResp.Token == "" {
		t.Fatal("expected non-empty csrf token")
	}
	loginPayload, err := json.Marshal(map[string]string{
		"username":   "admin",
		"password":   "admin",
		"csrf_token": csrfResp.Token,
	})
	if err != nil {
		t.Fatal(err)
	}
	loginBody := bytes.NewBuffer(loginPayload)
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
