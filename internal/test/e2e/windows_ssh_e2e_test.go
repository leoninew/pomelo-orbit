package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	"github.com/leoninew/pomelo-orbit/internal/bootstrap"
	"github.com/leoninew/pomelo-orbit/internal/config"
	traefikclient "github.com/leoninew/pomelo-orbit/internal/infrastructure/external/traefik"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	"github.com/leoninew/pomelo-orbit/internal/model"
	taskrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/task"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const windowsSSHE2EEnabledEnv = "POMELO_ORBIT_WINDOWS_SSH_E2E"
const windowsSSHE2ESecret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

type windowsSSHTarget struct {
	host               string
	port               int
	username           string
	workspaceRoot      string
	privateKey         string
	hostKeyFingerprint string
}

type e2eProjectResponse struct {
	Id string `json:"id"`
}

type e2eEnvironmentResponse struct {
	ProjectId           string                   `json:"project_id"`
	State               string                   `json:"state"`
	TargetType          string                   `json:"target_type"`
	SSH                 *e2eEnvironmentSSHTarget `json:"ssh"`
	TargetRevision      int64                    `json:"target_revision,string"`
	LastProbeRevision   *int64                   `json:"last_probe_revision,string"`
	LastProbeStatus     *string                  `json:"last_probe_status"`
	LastProbeDiagnostic *string                  `json:"last_probe_diagnostic"`
}

type e2eEnvironmentSSHTarget struct {
	Platform           string `json:"platform"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Username           string `json:"username"`
	WorkspaceRoot      string `json:"workspace_root"`
	HostKeyFingerprint string `json:"host_key_fingerprint"`
}

func TestWindowsSSHEnvironmentHTTPIntegration(t *testing.T) {
	if os.Getenv(windowsSSHE2EEnabledEnv) != "1" {
		t.Skip("set POMELO_ORBIT_WINDOWS_SSH_E2E=1 to run the local Windows OpenSSH integration test")
	}
	if runtime.GOOS != "windows" {
		t.Skip("Windows OpenSSH integration test requires Windows")
	}

	target := loadWindowsSSHTarget(t)
	ctx := context.Background()
	cfg := config.Config{
		Server: config.ServerConfig{ApiPathPrefixes: []string{"/api"}},
		Database: config.DatabaseConfig{
			Driver: config.DatabaseDriverSQLite,
			SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo-orbit-e2e.db")},
		},
		Workspace: config.WorkspaceConfig{Root: t.TempDir()},
		Logging:   config.LoggingConfig{DeploymentRoot: t.TempDir()},
		Worker:    config.WorkerConfig{MaxAttempts: 1},
		Jwt:       config.JwtConfig{SecretKey: windowsSSHE2ESecret},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := bootstrap.New(cfg, logger).Migrate(); err != nil {
		t.Fatalf("migrate temporary control plane: %v", err)
	}
	database, err := bootstrap.OpenDatabase(cfg)
	if err != nil {
		t.Fatalf("open temporary control plane: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	admin, err := userrepo.NewRepository(database).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("load seeded admin: %v", err)
	}
	token, err := jwt.NewTokenService(cfg.Jwt.SecretKey).Sign(admin.Id, admin.Username)
	if err != nil {
		t.Fatalf("sign admin token: %v", err)
	}
	handler := bootstrap.NewHTTPServer(cfg, logger, database, taskrepo.NewRepository(database)).Handler()

	code := "windows-ssh-e2e-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	project := createE2EWindowsSSHProject(t, handler, token, code)
	updateE2EWindowsSSHEnvironment(t, handler, token, project.Id, target)
	assertEnvironmentCredentialEncrypted(t, database, project.Id)

	environment := getE2EEnvironment(t, handler, token, project.Id)
	if environment.ProjectId != project.Id || environment.State != model.EnvironmentStateActive || environment.TargetType != model.EnvironmentTargetTypeSSH || environment.SSH == nil || environment.SSH.Platform != model.EnvironmentPlatformWindows {
		t.Fatalf("unexpected created environment: %+v", environment)
	}
	if environment.SSH.Host != target.host || environment.SSH.Port != target.port || environment.SSH.Username != target.username || environment.SSH.WorkspaceRoot != target.workspaceRoot {
		t.Fatalf("environment target = %+v, want %+v", environment, target)
	}
	if environment.TargetRevision != 2 || environment.LastProbeStatus != nil || environment.LastProbeRevision != nil {
		t.Fatalf("unexpected initial environment probe state: %+v", environment)
	}
	if environment.SSH.HostKeyFingerprint != "" {
		t.Fatalf("expected empty host key fingerprint before probe, got %q", environment.SSH.HostKeyFingerprint)
	}
	probed := probeE2EEnvironment(t, handler, token, project.Id)
	if probed.LastProbeStatus == nil || *probed.LastProbeStatus != model.EnvironmentProbeStatusSucceeded ||
		probed.LastProbeRevision == nil || *probed.LastProbeRevision != probed.TargetRevision ||
		probed.LastProbeDiagnostic == nil || strings.TrimSpace(*probed.LastProbeDiagnostic) == "" {
		t.Fatalf("unexpected probe result: %+v", probed)
	}
	if probed.SSH == nil || probed.SSH.HostKeyFingerprint != target.hostKeyFingerprint {
		t.Fatalf("probe fingerprint = %#v, want %q", probed.SSH, target.hostKeyFingerprint)
	}
}

func TestWindowsSSHRuntimeQueryIntegration(t *testing.T) {
	if os.Getenv(windowsSSHE2EEnabledEnv) != "1" {
		t.Skip("set POMELO_ORBIT_WINDOWS_SSH_E2E=1 to run the local Windows OpenSSH integration test")
	}
	if runtime.GOOS != "windows" {
		t.Skip("Windows OpenSSH integration test requires Windows")
	}

	targetConfig := loadWindowsSSHTarget(t)
	target := windowsSSHRuntimeTarget("environment-query-e2e", "", targetConfig)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := sshrunner.NewRuntime().QueryAtEnvironmentRoot(ctx, target, "docker", "version", "--format", "{{json .Server}}")
	if err != nil {
		t.Fatalf("query remote Docker JSON: %v", err)
	}
	var server struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal([]byte(output), &server); err != nil {
		t.Fatalf("decode remote Docker JSON: %v; output=%q", err, output)
	}
	if strings.TrimSpace(server.Version) == "" {
		t.Fatalf("remote Docker JSON is missing Version: %s", output)
	}
}

func TestWindowsSSHTraefikRouterQueryIntegration(t *testing.T) {
	if os.Getenv(windowsSSHE2EEnabledEnv) != "1" {
		t.Skip("set POMELO_ORBIT_WINDOWS_SSH_E2E=1 to run the local Windows OpenSSH integration test")
	}
	if runtime.GOOS != "windows" {
		t.Skip("Windows OpenSSH integration test requires Windows")
	}

	targetConfig := loadWindowsSSHTarget(t)
	target := windowsSSHRuntimeTarget("environment-traefik-e2e", "project-traefik-e2e", targetConfig)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	manager := traefikclient.NewRouteManager(windowsSSHTargetResolver{target: target}, sshrunner.NewRuntime())
	routers, err := manager.ListRouters(ctx, target.Environment.ProjectId, model.GatewayConfig{RestApiUrl: windowsSSHTraefikRestApiUrl()})
	if err != nil {
		t.Fatalf("query Traefik routers through Windows SSH: %v", err)
	}
	if len(routers) == 0 {
		t.Fatal("Traefik router query returned no routers")
	}
	for _, router := range routers {
		if strings.TrimSpace(router.Name) == "" || strings.TrimSpace(router.Provider) == "" || strings.TrimSpace(router.Status) == "" {
			t.Fatalf("Traefik router lacks required summary fields: %+v", router)
		}
	}
}

func TestWindowsSSHRuntimeComposeIntegration(t *testing.T) {
	if os.Getenv(windowsSSHE2EEnabledEnv) != "1" {
		t.Skip("set POMELO_ORBIT_WINDOWS_SSH_E2E=1 to run the local Windows OpenSSH integration test")
	}
	if runtime.GOOS != "windows" {
		t.Skip("Windows OpenSSH integration test requires Windows")
	}

	targetConfig := loadWindowsSSHTarget(t)
	testId := strconv.FormatInt(time.Now().UnixNano(), 36)
	serviceCode := "windows-ssh-runtime-e2e-" + testId
	composeProject := "orbit-e2e-" + testId
	target := windowsSSHRuntimeTarget("environment-"+testId, "", targetConfig)
	runtime := sshrunner.NewRuntime()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	workspace := deploymentport.Workspace{
		ServiceCode:  serviceCode,
		DeploymentId: testId,
		Compose: `services:
  smoke:
    image: alpine:latest
    command: ["/bin/sh", "-ec", "echo pomelo-orbit-windows-ssh-e2e"]
`,
	}
	if err := runtime.StageWorkspace(ctx, target, workspace); err != nil {
		t.Fatalf("stage remote Compose workspace: %v", err)
	}
	serviceDir, err := runtime.ServiceDir(target, serviceCode)
	if err != nil {
		t.Fatalf("resolve remote service directory: %v", err)
	}
	t.Cleanup(func() { cleanupE2EWorkspace(t, targetConfig, serviceDir) })
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := runtime.Run(cleanupCtx, target, serviceCode, io.Discard, "docker", "compose", "-p", composeProject, "down", "--volumes", "--remove-orphans"); err != nil {
			t.Errorf("clean up remote Compose project: %v", err)
		}
	})

	exists, err := runtime.ServiceDirExists(ctx, target, serviceCode)
	if err != nil || !exists {
		t.Fatalf("inspect staged remote service directory: exists=%t err=%v", exists, err)
	}
	var log bytes.Buffer
	if err := runtime.Run(ctx, target, serviceCode, &log, "docker", "compose", "-p", composeProject, "up", "--abort-on-container-exit", "--exit-code-from", "smoke", "--no-color"); err != nil {
		t.Fatalf("run remote Compose smoke service: %v\n%s", err, log.String())
	}
	if !strings.Contains(log.String(), "Running: docker compose") {
		t.Fatalf("remote Compose log missing command record: %s", log.String())
	}
}

func cleanupE2EWorkspace(t *testing.T, target windowsSSHTarget, serviceDir string) {
	t.Helper()
	signer, err := ssh.ParsePrivateKey([]byte(target.privateKey))
	if err != nil {
		t.Errorf("parse test SSH private key for cleanup: %v", err)
		return
	}
	address := net.JoinHostPort(target.host, strconv.Itoa(target.port))
	client, err := ssh.Dial("tcp", address, &ssh.ClientConfig{
		User: target.username,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			if ssh.FingerprintSHA256(key) != target.hostKeyFingerprint {
				return errors.New("SSH host key fingerprint mismatch during cleanup")
			}
			return nil
		},
	})
	if err != nil {
		t.Errorf("open SSH cleanup connection: %v", err)
		return
	}
	defer func() { _ = client.Close() }()
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		t.Errorf("open SFTP cleanup connection: %v", err)
		return
	}
	defer func() { _ = sftpClient.Close() }()
	if err := removeE2ERemoteTree(sftpClient, serviceDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Errorf("remove remote test workspace: %v", err)
	}
}

func removeE2ERemoteTree(client *sftp.Client, directory string) error {
	entries, err := client.ReadDir(directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		entryPath := path.Join(directory, entry.Name())
		if entry.IsDir() {
			if err := removeE2ERemoteTree(client, entryPath); err != nil {
				return err
			}
			continue
		}
		if err := client.Remove(entryPath); err != nil {
			return err
		}
	}
	return client.RemoveDirectory(directory)
}
func loadWindowsSSHTarget(t *testing.T) windowsSSHTarget {
	t.Helper()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve user home directory: %v", err)
	}
	privateKeyPath := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_PRIVATE_KEY_PATH"))
	if privateKeyPath == "" {
		privateKeyPath = filepath.Join(homeDir, ".ssh", "pomelo_orbit_ed25519")
	}
	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("read deployment SSH key %s: %v", privateKeyPath, err)
	}

	port := 2222
	if configuredPort := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_PORT")); configuredPort != "" {
		parsedPort, err := strconv.Atoi(configuredPort)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			t.Fatalf("invalid POMELO_ORBIT_WINDOWS_SSH_E2E_PORT: %q", configuredPort)
		}
		port = parsedPort
	}
	username := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_USERNAME"))
	if username == "" {
		current, err := user.Current()
		if err != nil {
			t.Fatalf("resolve current Windows user: %v", err)
		}
		username = current.Username
	}
	workspaceRoot := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_WORKSPACE_ROOT"))
	if workspaceRoot == "" {
		workspaceRoot = filepath.Join(homeDir, ".pomelo-orbit")
	}
	host := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	fingerprint := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_HOST_KEY_FINGERPRINT"))
	if fingerprint == "" {
		fingerprint = localWindowsHostKeyFingerprint(t)
	}
	return windowsSSHTarget{
		host: host, port: port, username: username, workspaceRoot: workspaceRoot,
		privateKey: string(privateKey), hostKeyFingerprint: fingerprint,
	}
}

func windowsSSHTraefikRestApiUrl() string {
	if value := strings.TrimSpace(os.Getenv("POMELO_ORBIT_WINDOWS_SSH_E2E_TRAEFIK_REST_API_URL")); value != "" {
		return value
	}
	return "http://localhost:8080"
}

type windowsSSHTargetResolver struct {
	target environmentport.Target
}

func (r windowsSSHTargetResolver) ResolveProjectTarget(_ context.Context, projectId string) (environmentport.Target, error) {
	if projectId != r.target.Environment.ProjectId {
		return environmentport.Target{}, errors.New("unexpected Project target")
	}
	return r.target, nil
}

func windowsSSHRuntimeTarget(environmentId string, projectId string, target windowsSSHTarget) environmentport.Target {
	return environmentport.Target{
		Environment: model.Environment{
			Id: environmentId, ProjectId: projectId, TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: target.workspaceRoot,
			SSH: &model.EnvironmentSSHTarget{
				Platform: model.EnvironmentPlatformWindows, Host: target.host, Port: target.port,
				Username:           target.username,
				HostKeyFingerprint: target.hostKeyFingerprint,
			},
		},
		PrivateKey: &environmentdto.DeploymentSSHPrivateKey{PrivateKey: target.privateKey},
	}
}

func localWindowsHostKeyFingerprint(t *testing.T) string {
	t.Helper()
	programData := os.Getenv("ProgramData")
	if programData == "" {
		t.Fatal("POMELO_ORBIT_WINDOWS_SSH_E2E_HOST_KEY_FINGERPRINT is required when ProgramData is unavailable")
	}
	publicKey, err := os.ReadFile(filepath.Join(programData, "ssh", "ssh_host_ed25519_key.pub"))
	if err != nil {
		t.Fatalf("read local OpenSSH host public key: %v", err)
	}
	key, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		t.Fatalf("parse local OpenSSH host public key: %v", err)
	}
	return ssh.FingerprintSHA256(key)
}

func createE2EWindowsSSHProject(t *testing.T, handler http.Handler, token, code string) e2eProjectResponse {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"name": "Windows SSH E2E",
		"code": code,
	})
	if err != nil {
		t.Fatalf("encode project create request: %v", err)
	}
	status, body := doE2ERequest(t, handler, token, http.MethodPost, "/api/project", payload)
	if status != http.StatusCreated {
		t.Fatalf("create project status=%d body=%s", status, body)
	}
	if bytes.Contains(body, []byte("PRIVATE KEY")) {
		t.Fatalf("project create response exposed deployment key: %s", body)
	}
	var project e2eProjectResponse
	if err := json.Unmarshal(body, &project); err != nil {
		t.Fatalf("decode project create response: %v", err)
	}
	if project.Id == "" {
		t.Fatalf("project create response missing id: %s", body)
	}
	return project
}

func updateE2EWindowsSSHEnvironment(t *testing.T, handler http.Handler, token, projectId string, target windowsSSHTarget) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"state":       model.EnvironmentStateActive,
		"target_type": model.EnvironmentTargetTypeSSH,
		"ssh": map[string]any{
			"platform":       model.EnvironmentPlatformWindows,
			"host":           target.host,
			"port":           target.port,
			"username":       target.username,
			"workspace_root": target.workspaceRoot,
		},
	})
	if err != nil {
		t.Fatalf("encode Windows SSH environment: %v", err)
	}
	status, body := doE2ERequest(t, handler, token, http.MethodPut, "/api/environment?project_id="+url.QueryEscape(projectId), payload)
	if status != http.StatusOK {
		t.Fatalf("update environment status=%d body=%s", status, body)
	}
}

func assertEnvironmentCredentialEncrypted(t *testing.T, database *sql.DB, projectId string) {
	t.Helper()
	var credentialId, publicKey, encryptedPrivateKey string
	if err := database.QueryRowContext(context.Background(), `SELECT id, public_key, encrypted_private_key FROM environment_credential WHERE project_id = ?`, projectId).Scan(&credentialId, &publicKey, &encryptedPrivateKey); err != nil {
		t.Fatalf("load environment credential: %v", err)
	}
	if credentialId == "" || publicKey == "" || encryptedPrivateKey == "" || strings.Contains(encryptedPrivateKey, "PRIVATE KEY") {
		t.Fatal("environment credential was not stored with a public key and encrypted private key")
	}
}

func getE2EEnvironment(t *testing.T, handler http.Handler, token, projectId string) e2eEnvironmentResponse {
	t.Helper()
	status, body := doE2ERequest(t, handler, token, http.MethodGet, "/api/environment?project_id="+url.QueryEscape(projectId), nil)
	if status != http.StatusOK {
		t.Fatalf("get environment status=%d body=%s", status, body)
	}
	var environment e2eEnvironmentResponse
	if err := json.Unmarshal(body, &environment); err != nil {
		t.Fatalf("decode environment response: %v", err)
	}
	return environment
}

func probeE2EEnvironment(t *testing.T, handler http.Handler, token, projectId string) e2eEnvironmentResponse {
	t.Helper()
	status, body := doE2ERequest(t, handler, token, http.MethodPost, "/api/environment/probe?project_id="+url.QueryEscape(projectId), nil)
	if status != http.StatusOK {
		t.Fatalf("probe environment status=%d body=%s", status, body)
	}
	var environment e2eEnvironmentResponse
	if err := json.Unmarshal(body, &environment); err != nil {
		t.Fatalf("decode environment probe response: %v", err)
	}
	return environment
}

func doE2ERequest(t *testing.T, handler http.Handler, token, method, path string, body []byte) (int, []byte) {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code, response.Body.Bytes()
}
