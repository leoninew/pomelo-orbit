package clihandler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/middleware"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	"github.com/leoninew/pomelo-orbit/internal/api/http/terminal"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const testPAT = "orbit_mcp_pat_test"

type authStore struct {
	repository.UserStore
	repository.AuthStore
}

func (authStore) UserRoles(context.Context, string) ([]string, error)       { return nil, nil }
func (authStore) UserPermissions(context.Context, string) ([]string, error) { return nil, nil }

func (authStore) UserById(context.Context, string) (model.User, error) {
	return model.User{Id: "actor", Status: "enabled"}, nil
}
func (authStore) MCPAccessTokenByHash(_ context.Context, hash string) (model.MCPAccessToken, error) {
	sum := sha256.Sum256([]byte(testPAT))
	if hash != hex.EncodeToString(sum[:]) {
		return model.MCPAccessToken{}, repository.ErrNotFound
	}
	return model.MCPAccessToken{UserId: "actor"}, nil
}

type applications struct {
	terminal.Application
	input    environmentdto.CommandInput
	calls    int
	released atomic.Bool
}

func (s *applications) List(_ context.Context, actor string) ([]environmentdto.ProjectSummary, error) {
	if actor != "actor" {
		panic("wrong actor")
	}
	return []environmentdto.ProjectSummary{{Code: "production", TargetType: "ssh"}}, nil
}
func (s *applications) Execute(_ context.Context, actor string, input environmentdto.CommandInput) (environmentdto.CommandResult, error) {
	s.calls++
	s.input = input
	code := int32(7)
	return environmentdto.CommandResult{ProjectCode: input.ProjectCode, State: "exited", ExitCode: &code, Stdout: []byte{0, 255}, Stderr: []byte("failure")}, nil
}
func (s *applications) AcquireTerminal(_ context.Context, actor, code string) (environmentsvc.TerminalGrant, func(), error) {
	if actor != "actor" || code != "production" {
		panic("wrong target")
	}
	return environmentsvc.TerminalGrant{ActorId: actor, Target: environmentport.Target{Environment: model.Environment{TargetType: "ssh"}}}, func() { s.released.Store(true) }, nil
}

type failedTerminal struct{}

func (failedTerminal) StartTerminal(context.Context, environmentport.Target, int, int) (environmentport.TerminalSession, error) {
	return nil, errors.New("private SSH authentication cause")
}

func testHandler(t *testing.T) (*gin.Engine, *applications) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := authStore{}
	auth := security.New(slog.Default(), authsvc.New(store, store, jwt.NewTokenService("secret"), slog.Default(), "secret"))
	app := &applications{}
	h := New(auth, app, app, app, failedTerminal{}, slog.Default())
	engine := gin.New()
	engine.Use(middleware.RequestId())
	engine.GET("/api/cli/project", h.List)
	engine.POST("/api/cli/environment/exec", h.Exec)
	engine.GET("/api/cli/environment/terminal", h.Terminal)
	return engine, app
}

func TestCLIExecHTTPContractAndBoundedStrictBody(t *testing.T) {
	engine, app := testHandler(t)
	for _, body := range []string{`{"project_code":"production","argv":["tool","--help"],"stdin":"AP8=","timeout_seconds":0}`, `{"argv":["tool"],"unknown":true}`, `{"stdin":"` + strings.Repeat("a", 128*1024) + `"}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/cli/environment/exec", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+testPAT)
		req.Header.Set("X-Request-Id", "request")
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, req)
		if strings.Contains(body, "project_code") {
			var value map[string]any
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &value) != nil || value["stdout"] != "AP8=" || value["exit_code"] != float64(7) || app.input.TimeoutSeconds == nil || *app.input.TimeoutSeconds != 0 || !bytes.Equal(app.input.Stdin, []byte{0, 255}) {
				t.Fatalf("response=%s input=%+v", response.Body.String(), app.input)
			}
		} else if response.Code != 400 || !strings.Contains(response.Body.String(), `"requestId":"request"`) {
			t.Fatalf("response=%d %s", response.Code, response.Body.String())
		}
	}
	if app.calls != 1 {
		t.Fatalf("invalid bodies executed: %d", app.calls)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/cli/project", nil))
	if response.Code != 401 || response.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatal("missing PAT accepted")
	}
}

func TestCLIPATWebSocketUsesSharedProtocolAndReleasesGrant(t *testing.T) {
	engine, app := testHandler(t)
	server := httptest.NewServer(engine)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/cli/environment/terminal?project_code=production", &websocket.DialOptions{Subprotocols: []string{terminal.Protocol}, HTTPHeader: http.Header{"Authorization": []string{"Bearer " + testPAT}}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	_, data, err := conn.Read(ctx)
	if err != nil || !bytes.Contains(data, []byte(`"type":"connection_failed"`)) || bytes.Contains(data, []byte("private")) {
		t.Fatalf("frame=%s err=%v", data, err)
	}
	_, _, _ = conn.Read(ctx)
	for !app.released.Load() {
		select {
		case <-ctx.Done():
			t.Fatal("grant not released")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
