package environmenthandler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/middleware"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type terminalTestStore struct {
	repository.ProjectReader
	repository.EnvironmentStore
	repository.EnvironmentCredentialStore
	member      atomic.Bool
	mu          sync.Mutex
	environment model.Environment
	credential  model.EnvironmentCredential
}

func (s *terminalTestStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project"}, nil
}

func (s *terminalTestStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return s.member.Load(), nil
}

func (s *terminalTestStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.environment
	if item.SSH != nil {
		ssh := *item.SSH
		item.SSH = &ssh
	}
	return item, nil
}

func (s *terminalTestStore) EnvironmentCredential(context.Context, string) (model.EnvironmentCredential, error) {
	return s.credential, nil
}

func (s *terminalTestStore) RecordHostKey(_ context.Context, environment model.Environment, fingerprint string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.environment.TargetRevision != environment.TargetRevision ||
		(s.environment.SSH.HostKeyFingerprint != "" && s.environment.SSH.HostKeyFingerprint != fingerprint) {
		return false, nil
	}
	s.environment.SSH.HostKeyFingerprint = fingerprint
	return true, nil
}

type terminalTestSession struct {
	reader      *io.PipeReader
	writer      *io.PipeWriter
	done        chan struct{}
	resized     chan [2]int
	once        sync.Once
	exitCode    atomic.Int32
	fingerprint string
}

func newTerminalTestSession() *terminalTestSession {
	reader, writer := io.Pipe()
	return &terminalTestSession{reader: reader, writer: writer, done: make(chan struct{}), resized: make(chan [2]int, 4)}
}

func (s *terminalTestSession) Input() io.WriteCloser      { return s.writer }
func (s *terminalTestSession) Output() io.Reader          { return s.reader }
func (s *terminalTestSession) HostKeyFingerprint() string { return s.fingerprint }
func (s *terminalTestSession) Resize(cols, rows int) error {
	s.resized <- [2]int{cols, rows}
	return nil
}
func (s *terminalTestSession) Wait() (int, error) {
	<-s.done
	return int(s.exitCode.Load()), nil
}
func (s *terminalTestSession) Close() error {
	s.once.Do(func() {
		_ = s.writer.Close()
		_ = s.reader.Close()
		close(s.done)
	})
	return nil
}

type terminalTestRunner struct {
	session      *terminalTestSession
	targets      chan environmentport.Target
	blockConnect *atomic.Bool
}

func (r terminalTestRunner) StartTerminal(ctx context.Context, target environmentport.Target, _, _ int) (environmentport.TerminalSession, error) {
	r.targets <- target
	if r.blockConnect.Load() {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return r.session, nil
}

func terminalHTTPFixture(t *testing.T, origins []string) (*httptest.Server, *environmentsvc.TerminalService, *terminalTestStore, terminalTestRunner, <-chan struct{}) {
	t.Helper()
	revision := int64(1)
	status := model.EnvironmentProbeStatusSucceeded
	store := &terminalTestStore{environment: model.Environment{
		Id: "environment", ProjectId: "project", Code: "project", TargetType: model.EnvironmentTargetTypeSSH,
		WorkspaceRoot:  "/srv/orbit",
		TargetRevision: revision, LastProbeRevision: &revision, LastProbeStatus: &status,
		SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux, Host: "remote-host", Port: 22, Username: "configured-user",
			CredentialId: "credential", CredentialRevision: 1, HostKeyFingerprint: "SHA256:hostkey"},
	}}
	store.member.Store(true)
	const secret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	encrypted, err := security.EncryptString(secret, "managed-key")
	if err != nil {
		t.Fatal(err)
	}
	store.credential = model.EnvironmentCredential{Id: "credential", ProjectId: "project", Revision: 1, EncryptedPrivateKey: encrypted}
	service := environmentsvc.New(store, store, store, secret, nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux, Username: "orbit"})
	terminal := environmentsvc.NewTerminalService(service)
	runner := terminalTestRunner{session: newTerminalTestSession(), targets: make(chan environmentport.Target, 1), blockConnect: &atomic.Bool{}}
	runner.session.fingerprint = "SHA256:hostkey"
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Handler{logger: logger, terminal: terminal, runner: runner, origins: origins}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.LogRequest(logger, middleware.LogRequestConfig{Enabled: true, ResponseBodyLimit: 1024}))
	finished := make(chan struct{})
	engine.GET("/api/environment/terminal", func(c *gin.Context) {
		handler.ConnectTerminal(c)
		close(finished)
	})
	server := httptest.NewServer(engine)
	t.Cleanup(server.Close)
	t.Cleanup(func() { _ = runner.session.Close() })
	return server, terminal, store, runner, finished
}

func dialTestTerminal(t *testing.T, ctx context.Context, server *httptest.Server, svc *environmentsvc.TerminalService, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ticket, err := svc.IssueTicket(ctx, "actor", "project")
	if err != nil {
		t.Fatal(err)
	}
	return websocket.Dial(ctx, server.URL+"/api/environment/terminal?project_id=project", &websocket.DialOptions{
		Subprotocols: []string{terminalProtocol, "ticket." + ticket}, HTTPHeader: http.Header{"Origin": []string{origin}},
	})
}

func readTerminalControl(t *testing.T, ctx context.Context, conn *websocket.Conn) terminalControl {
	t.Helper()
	messageType, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var control terminalControl
	if messageType != websocket.MessageText || json.Unmarshal(data, &control) != nil {
		t.Fatalf("expected terminal control, got type %v", messageType)
	}
	return control
}

func awaitTerminalClosed(t *testing.T, finished <-chan struct{}) {
	t.Helper()
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("terminal handler did not release its connection")
	}
}

func TestTerminalWebSocketTransfersInputOutputResizeAndExit(t *testing.T) {
	server, svc, _, runner, finished := terminalHTTPFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	if conn.Subprotocol() != terminalProtocol || readTerminalControl(t, ctx, conn).Type != "ready" {
		t.Fatal("terminal protocol or SSH readiness was not established")
	}
	target := <-runner.targets
	if target.Environment.SSH.Username != "configured-user" || target.Environment.SSH.Host != "remote-host" {
		t.Fatal("WebSocket changed the configured SSH identity")
	}
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("echo input\n")); err != nil {
		t.Fatal(err)
	}
	messageType, output, err := conn.Read(ctx)
	if err != nil || messageType != websocket.MessageBinary || string(output) != "echo input\n" {
		t.Fatalf("terminal output type=%v err=%v", messageType, err)
	}
	if err := conn.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":100,"rows":40}`)); err != nil {
		t.Fatal(err)
	}
	select {
	case size := <-runner.session.resized:
		if size != [2]int{100, 40} {
			t.Fatalf("resize = %v", size)
		}
	case <-ctx.Done():
		t.Fatal("resize was not forwarded")
	}
	runner.session.exitCode.Store(7)
	_ = runner.session.writer.Close()
	runner.session.once.Do(func() { close(runner.session.done) })
	control := readTerminalControl(t, ctx, conn)
	if control.Type != "exit" || control.Code != 7 {
		t.Fatalf("exit control = %+v", control)
	}
	_, _, err = conn.Read(ctx)
	if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
		t.Fatalf("close error = %v", err)
	}
	awaitTerminalClosed(t, finished)
}

func TestTerminalRejectsForeignOriginBeforeStartingSSH(t *testing.T) {
	server, svc, _, runner, finished := terminalHTTPFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, response, err := dialTestTerminal(t, ctx, server, svc, "https://foreign.example")
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign origin error=%v", err)
	}
	select {
	case <-runner.targets:
		t.Fatal("SSH started for a foreign origin")
	default:
	}
	awaitTerminalClosed(t, finished)
}

func TestTerminalAllowsConfiguredOriginAndClosesWithClient(t *testing.T) {
	server, svc, _, _, finished := terminalHTTPFixture(t, []string{"https://orbit.example"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, "https://orbit.example")
	if err != nil {
		t.Fatal(err)
	}
	if readTerminalControl(t, ctx, conn).Type != "ready" {
		t.Fatal("terminal was not ready")
	}
	_ = conn.CloseNow()
	awaitTerminalClosed(t, finished)
}

func TestTerminalStopsBeforeForwardingInputAfterMembershipRemoval(t *testing.T) {
	server, svc, store, _, finished := terminalHTTPFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	_ = readTerminalControl(t, ctx, conn)
	store.member.Store(false)
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("blocked command")); err != nil {
		t.Fatal(err)
	}
	if readTerminalControl(t, ctx, conn).Type != "access_or_target_changed" {
		t.Fatal("revoked input was forwarded")
	}
	_, _, _ = conn.Read(ctx)
	awaitTerminalClosed(t, finished)
}

func TestTerminalTicketRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/api/environment/terminal/ticket", Handler{}.IssueTerminalTicket)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/environment/terminal/ticket?project_id=project", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestTerminalDisconnectCancelsPendingSSHConnection(t *testing.T) {
	server, svc, _, runner, finished := terminalHTTPFixture(t, nil)
	runner.blockConnect.Store(true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-runner.targets:
	case <-ctx.Done():
		t.Fatal("SSH did not start connecting")
	}
	_ = conn.CloseNow()
	awaitTerminalClosed(t, finished)
}

func TestTerminalClosesAfterTargetChangesWithoutClientInput(t *testing.T) {
	server, svc, store, _, finished := terminalHTTPFixture(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	_ = readTerminalControl(t, ctx, conn)
	store.mu.Lock()
	store.environment.TargetRevision++
	store.mu.Unlock()
	if readTerminalControl(t, ctx, conn).Type != "access_or_target_changed" {
		t.Fatal("target change did not stop an idle terminal")
	}
	_, _, _ = conn.Read(ctx)
	awaitTerminalClosed(t, finished)
}

func TestLocalTerminalConnectsAndSurvivesDockerProbeFailure(t *testing.T) {
	server, svc, store, runner, finished := terminalHTTPFixture(t, nil)
	store.environment.TargetType, store.environment.SSH = model.EnvironmentTargetTypeLocal, nil
	store.environment.LastProbeRevision, store.environment.LastProbeStatus = nil, nil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	if readTerminalControl(t, ctx, conn).Type != "ready" {
		t.Fatal("local terminal was not ready without a probe")
	}
	if target := <-runner.targets; !target.Environment.IsLocal() || target.PrivateKey != nil {
		t.Fatal("local terminal used an SSH target or private key")
	}
	store.mu.Lock()
	failed := model.EnvironmentProbeStatusFailed
	store.environment.LastProbeStatus = &failed
	store.mu.Unlock()
	if err := conn.Write(ctx, websocket.MessageBinary, []byte("local-input")); err != nil {
		t.Fatal(err)
	}
	messageType, data, err := conn.Read(ctx)
	if err != nil || messageType != websocket.MessageBinary || string(data) != "local-input" {
		t.Fatalf("local terminal after probe failure: type=%v data=%q err=%v", messageType, data, err)
	}
	_ = conn.CloseNow()
	awaitTerminalClosed(t, finished)
}

func TestTerminalPinsFirstHostKeyWithoutProbeBeforeReady(t *testing.T) {
	server, svc, store, _, finished := terminalHTTPFixture(t, nil)
	store.environment.SSH.HostKeyFingerprint = ""
	store.environment.LastProbeRevision, store.environment.LastProbeStatus = nil, nil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if readTerminalControl(t, ctx, conn).Type != "ready" {
		t.Fatal("SSH terminal was not ready without Docker probe")
	}
	store.mu.Lock()
	pin, status := store.environment.SSH.HostKeyFingerprint, store.environment.LastProbeStatus
	store.mu.Unlock()
	if pin != "SHA256:hostkey" || status != nil {
		t.Fatal("terminal readiness did not pin the host key independently of Probe")
	}
	_ = conn.CloseNow()
	awaitTerminalClosed(t, finished)
}

func TestTerminalDoesNotSendReadyOrInputWhenHostKeyConfirmationFails(t *testing.T) {
	server, svc, _, runner, finished := terminalHTTPFixture(t, nil)
	runner.session.fingerprint = "SHA256:different"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := dialTestTerminal(t, ctx, server, svc, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	if readTerminalControl(t, ctx, conn).Type != "connection_failed" {
		t.Fatal("terminal became ready before host key confirmation")
	}
	_, _, _ = conn.Read(ctx)
	awaitTerminalClosed(t, finished)
}
