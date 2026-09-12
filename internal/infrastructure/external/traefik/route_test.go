package traefik

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRouteManagerListRoutersUsesRemoteTraefikAPI(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = `[{"name":"api@docker","provider":"docker","status":"enabled","rule":"Host(api.example.test)","service":"api-service","entryPoints":["websecure"],"tls":{}}]`
	runtime.responses["/api/tcp/routers"] = `[{"name":"redis@rest","provider":"rest","status":"enabled","rule":"HostSNI(` + "`*`" + `)","service":"redis-service","entryPoints":["tcp16379"],"tls":null}]`
	manager := NewRouteManager(routeTargetResolver{}, runtime)
	routers, err := manager.ListRouters(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	if len(routers) != 2 || routers[0].Name != "api@docker" || !routers[0].TLS || routers[1].Name != "redis@rest" || routers[1].TLS {
		t.Fatalf("unexpected routers: %+v", routers)
	}
	if runtime.lastTarget.Environment.ProjectId != "project-1" {
		t.Fatalf("remote target project = %q", runtime.lastTarget.Environment.ProjectId)
	}
	if runtime.environmentQueries != 2 {
		t.Fatalf("environment-root query count = %d, want 2", runtime.environmentQueries)
	}
}

func TestRouteManagerListServicesMapsRemoteHTTPAndTCPResponses(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/services"] = `[{"name":"api-service@rest","provider":"rest","status":"enabled","loadBalancer":{"servers":[{"url":"http://api:8080"}]}}]`
	runtime.responses["/api/tcp/services"] = `[{"name":"redis-service@rest","provider":"rest","status":"enabled","loadBalancer":{"servers":[{"address":"redis:6379"}]}}]`
	manager := NewRouteManager(routeTargetResolver{}, runtime)
	services, err := manager.ListServices(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 || services[0].Servers[0] != "http://api:8080" || services[1].Protocol != "tcp" || services[1].Servers[0] != "redis:6379" {
		t.Fatalf("unexpected services: %+v", services)
	}
}

func TestRouteManagerWaitUntilReadyPollsRemoteHost(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.failures["/api/overview"] = 2
	manager := NewRouteManager(routeTargetResolver{}, runtime)
	if err := manager.WaitUntilReady(context.Background(), "project-1", testGateway(), 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if runtime.attempts["/api/overview"] != 3 {
		t.Fatalf("readiness attempts = %d", runtime.attempts["/api/overview"])
	}
}

func TestRouteManagerApplySnapshotStagesRemoteCertificatesAndPUTBody(t *testing.T) {
	runtime := newRouteRuntimeFake()
	manager := NewRouteManager(routeTargetResolver{}, runtime)
	certificate, key := "certificate", "private key"
	err := manager.ApplySnapshot(context.Background(), "project-1", testGateway(), []model.Route{
		{Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/v1", TargetUrl: "http://app:8080", Enabled: true},
		{Name: "secure", Protocol: "http", Domain: "secure.example.test", PathPrefix: "/", TargetUrl: "http://app:8082", Enabled: true, HTTPSEnabled: true, CertType: "manual", CertPEM: &certificate, CertKey: &key},
		{Name: "redis", Protocol: "tcp", ListenPort: intPtr(16379), TargetAddress: "redis-redis", TargetPort: 6379, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(runtime.files["/srv/orbit/traefik-default/gateway/certs/secure.pem"]) != certificate || string(runtime.files["/srv/orbit/traefik-default/gateway/certs/secure-key.pem"]) != key {
		t.Fatalf("remote certificate files = %#v", runtime.files)
	}
	var payload map[string]any
	if err := json.Unmarshal(runtime.putBody, &payload); err != nil {
		t.Fatal(err)
	}
	httpCfg := payload["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 2 {
		t.Fatalf("unexpected REST snapshot: %s", runtime.putBody)
	}
	if runtime.putURL != "http://127.0.0.1:8080/api/providers/rest" {
		t.Fatalf("PUT URL = %q", runtime.putURL)
	}
	if runtime.putFile != "@-" {
		t.Fatalf("PUT file = %q", runtime.putFile)
	}
	if !json.Valid(runtime.putBody) || !strings.Contains(string(runtime.putBody), `"http"`) {
		t.Fatalf("PUT body = %s", runtime.putBody)
	}
	if runtime.environmentQueries != 1 {
		t.Fatalf("environment-root query count = %d, want 1", runtime.environmentQueries)
	}
}

func TestRestSnapshotLocationNormalizesWindowsServiceDir(t *testing.T) {
	stateDir, snapshotPath := restSnapshotLocation(`C:\Users\wangm25\.pomelo-orbit\traefik-default`)
	if stateDir != "C:/Users/wangm25/.pomelo-orbit/traefik-default/.orbit" {
		t.Fatalf("state dir = %q", stateDir)
	}
	if snapshotPath != "C:/Users/wangm25/.pomelo-orbit/traefik-default/.orbit/traefik-rest.json" {
		t.Fatalf("snapshot path = %q", snapshotPath)
	}
}

func TestBuildRestSnapshotSkipsDisabled(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{
		{Name: "a", Protocol: "http", Domain: "a.test", PathPrefix: "/", TargetUrl: "http://a:1", Enabled: true},
		{Name: "b", Protocol: "http", Domain: "b.test", PathPrefix: "/", TargetUrl: "http://b:1", Enabled: false},
	})
	httpCfg := snapshot["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 1 {
		t.Fatalf("expected one router: %#v", snapshot)
	}
}

func TestBuildRestSnapshotUsesDNSResolverForDNSChallenge(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{{
		Name: "dns", Protocol: "http", Domain: "dns.example.test", PathPrefix: "/", TargetUrl: "http://app:8080", Enabled: true,
		HTTPSEnabled: true, CertType: "letsencrypt", AcmeChallenge: "dns",
	}})
	router := snapshot["http"].(map[string]any)["routers"].(map[string]any)["dns-route"].(map[string]any)
	if got := router["tls"].(map[string]any)["certResolver"]; got != "letsencrypt-dns" {
		t.Fatalf("DNS cert resolver = %#v", got)
	}
}

func TestBuildRestSnapshotUsesResolvedManagedHTTPTarget(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{{
		Name: "api", Protocol: "http", Domain: "api.test", PathPrefix: "/", TargetAddress: "api-api", TargetPort: 8080, Enabled: true,
	}})
	httpCfg := snapshot["http"].(map[string]any)
	service := httpCfg["services"].(map[string]any)["api-service"].(map[string]any)
	servers := service["loadBalancer"].(map[string]any)["servers"].([]map[string]string)
	if len(servers) != 1 || servers[0]["url"] != "http://api-api:8080" {
		t.Fatalf("unexpected managed HTTP servers: %+v", servers)
	}
}

func intPtr(value int) *int { return &value }

func testGateway() model.GatewayConfig {
	return model.GatewayConfig{RestApiUrl: "http://127.0.0.1:8080", RuntimeServiceCode: "traefik-default"}
}

type routeTargetResolver struct{}

func (routeTargetResolver) ResolveProjectTarget(_ context.Context, projectID string) (environmentport.Target, error) {
	return environmentport.Target{Environment: model.Environment{Id: "environment-1", ProjectId: projectID, TargetType: model.EnvironmentTargetTypeLocal}}, nil
}

type routeRuntimeFake struct {
	responses          map[string]string
	failures           map[string]int
	attempts           map[string]int
	files              map[string][]byte
	putBody            []byte
	putStdin           []byte
	putFile            string
	putURL             string
	lastTarget         environmentport.Target
	environmentQueries int
}

func newRouteRuntimeFake() *routeRuntimeFake {
	return &routeRuntimeFake{responses: map[string]string{}, failures: map[string]int{}, attempts: map[string]int{}, files: map[string][]byte{}}
}

func (r *routeRuntimeFake) ServiceDir(_ environmentport.Target, serviceCode string) (string, error) {
	return "/srv/orbit/" + serviceCode, nil
}
func (r *routeRuntimeFake) ServiceDirExists(context.Context, environmentport.Target, string) (bool, error) {
	return true, nil
}
func (r *routeRuntimeFake) ComposeMountSourceDir(_ context.Context, target environmentport.Target, serviceCode string) (string, error) {
	return r.ServiceDir(target, serviceCode)
}
func (r *routeRuntimeFake) StageWorkspace(context.Context, environmentport.Target, deploymentport.Workspace) error {
	return nil
}
func (r *routeRuntimeFake) Run(context.Context, environmentport.Target, string, io.Writer, string, ...string) error {
	return nil
}
func (r *routeRuntimeFake) Query(_ context.Context, target environmentport.Target, _ string, _ string, args ...string) (string, error) {
	return r.query(target, args...)
}
func (r *routeRuntimeFake) QueryAtEnvironmentRoot(_ context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	return r.QueryAtEnvironmentRootInput(context.Background(), target, nil, name, args...)
}
func (r *routeRuntimeFake) QueryAtEnvironmentRootInput(_ context.Context, target environmentport.Target, stdin []byte, _ string, args ...string) (string, error) {
	r.environmentQueries++
	r.putStdin = append([]byte(nil), stdin...)
	return r.query(target, args...)
}
func (r *routeRuntimeFake) query(target environmentport.Target, args ...string) (string, error) {
	r.lastTarget = target
	endpoint := args[len(args)-1]
	for suffix, response := range r.responses {
		if strings.HasSuffix(endpoint, suffix) {
			r.attempts[suffix]++
			if r.attempts[suffix] <= r.failures[suffix] {
				return "", errors.New("remote curl failed")
			}
			return response, nil
		}
	}
	if strings.HasSuffix(endpoint, "/api/overview") {
		r.attempts["/api/overview"]++
		if r.attempts["/api/overview"] <= r.failures["/api/overview"] {
			return "", errors.New("remote curl failed")
		}
		return `{}`, nil
	}
	if strings.HasSuffix(endpoint, "/api/providers/rest") {
		r.putURL = endpoint
		for _, arg := range args {
			if !strings.HasPrefix(arg, "@") {
				continue
			}
			r.putFile = arg
			if arg == "@-" {
				r.putBody = append([]byte(nil), r.putStdin...)
				continue
			}
			r.putBody = restSnapshotBody(r.files, strings.TrimPrefix(arg, "@"))
		}
		return "", nil
	}
	return "", errors.New("unexpected remote curl endpoint")
}
func restSnapshotBody(files map[string][]byte, fileArg string) []byte {
	fileArg = strings.TrimPrefix(strings.ReplaceAll(fileArg, "\\", "/"), "./")
	if body, ok := files[fileArg]; ok {
		return append([]byte(nil), body...)
	}
	for filePath, content := range files {
		normalized := strings.ReplaceAll(filePath, "\\", "/")
		if normalized == fileArg || strings.HasSuffix(normalized, "/"+fileArg) {
			return append([]byte(nil), content...)
		}
	}
	return nil
}

func (r *routeRuntimeFake) SyncFiles(_ context.Context, _ environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	keep := map[string]struct{}{}
	for _, file := range files {
		r.files[file.Path] = append([]byte(nil), file.Content...)
		keep[path.Base(file.Path)] = struct{}{}
	}
	if pruneSuffix != "" {
		for filePath := range r.files {
			if path.Dir(filePath) == directory && strings.HasSuffix(filePath, pruneSuffix) {
				if _, ok := keep[path.Base(filePath)]; !ok {
					delete(r.files, filePath)
				}
			}
		}
	}
	return nil
}
