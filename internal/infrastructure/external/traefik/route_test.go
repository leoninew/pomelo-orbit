package traefik

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func testRouteTimeouts() routeport.SyncTimeouts {
	return routeport.SyncTimeouts{Total: 30 * time.Second, ApiRequest: 3 * time.Second, Reload: 5 * time.Second, ConfigurationMatch: 10 * time.Second, Recovery: 10 * time.Second}
}

func TestRouteManagerListRoutersUsesRemoteTraefikApi(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = `[{"name":"api@docker","provider":"docker","status":"enabled","rule":"Host(api.example.test)","service":"api-service","entryPoints":["websecure"],"tls":{}}]`
	runtime.responses["/api/tcp/routers"] = `[{"name":"redis@file","provider":"file","status":"enabled","rule":"HostSNI(` + "`*`" + `)","service":"redis-service","entryPoints":["tcp16379"],"tls":null}]`
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return true })
	routers, err := manager.ListRouters(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	if len(routers) != 2 || routers[0].Name != "api@docker" || !routers[0].TLS || routers[1].Name != "redis@file" || routers[1].TLS {
		t.Fatalf("unexpected routers: %+v", routers)
	}
	if runtime.lastTarget.Environment.ProjectId != "project-1" {
		t.Fatalf("remote target project = %q", runtime.lastTarget.Environment.ProjectId)
	}
	if runtime.environmentQueries != 2 {
		t.Fatalf("environment-root query count = %d, want 2", runtime.environmentQueries)
	}
	if got, want := runtime.requests, []string{
		"http://traefik:8080/api/http/routers",
		"http://traefik:8080/api/tcp/routers",
	}; !equalStrings(got, want) {
		t.Fatalf("REST requests = %#v, want %#v", got, want)
	}
}

func TestRouteManagerListServicesMapsRemoteHTTPAndTCPResponses(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/services"] = `[{"name":"api-service@file","provider":"file","status":"enabled","loadBalancer":{"servers":[{"url":"http://api:8080"}]}}]`
	runtime.responses["/api/tcp/services"] = `[{"name":"redis-service@file","provider":"file","status":"enabled","loadBalancer":{"servers":[{"address":"redis:6379"}]}}]`
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return true })
	services, err := manager.ListServices(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 || services[0].Servers[0] != "http://api:8080" || services[1].Protocol != "tcp" || services[1].Servers[0] != "redis:6379" {
		t.Fatalf("unexpected services: %+v", services)
	}
}

func TestRouteManagerUsesHostEndpointForHostLocalOrbit(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = `[]`
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })

	if _, err := manager.listRouters(context.Background(), "project-1", testGateway(), "http"); err != nil {
		t.Fatal(err)
	}
	if got, want := runtime.requests, []string{"http://127.0.0.1:8080/api/http/routers"}; !equalStrings(got, want) {
		t.Fatalf("REST requests = %#v, want %#v", got, want)
	}
}

func TestRouteManagerUsesHostEndpointForSSHOrbit(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = `[]`
	manager := newRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH}, runtime, testRouteTimeouts(), func() bool { return true })

	if _, err := manager.listRouters(context.Background(), "project-1", testGateway(), "http"); err != nil {
		t.Fatal(err)
	}
	if got, want := runtime.requests, []string{"http://127.0.0.1:8080/api/http/routers"}; !equalStrings(got, want) {
		t.Fatalf("REST requests = %#v, want %#v", got, want)
	}
}

func TestRouteManagerDoesNotFallbackAfterRequestFailure(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = `[]`
	runtime.failures["/api/http/routers"] = 1
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return true })

	if _, err := manager.listRouters(context.Background(), "project-1", testGateway(), "http"); err == nil {
		t.Fatal("expected Traefik request failure")
	}
	if got, want := runtime.requests, []string{"http://traefik:8080/api/http/routers"}; !equalStrings(got, want) {
		t.Fatalf("REST requests = %#v, want %#v", got, want)
	}
}

func TestRouteManagerReportsSafeAccessContextForRESTRequestFailure(t *testing.T) {
	tests := []struct {
		name        string
		targetType  string
		inContainer bool
		want        string
	}{
		{name: "container local", inContainer: true, want: "Traefik REST API is unavailable in the Orbit container at http://traefik:8080."},
		{name: "host local", inContainer: false, want: "Traefik REST API is unavailable on the local host at http://127.0.0.1:8080."},
		{name: "ssh", targetType: model.EnvironmentTargetTypeSSH, inContainer: true, want: "Traefik REST API is unavailable on the remote host at http://127.0.0.1:8080."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.responses["/api/http/routers"] = `[]`
			runtime.failures["/api/http/routers"] = 1
			runtime.failureErrors["/api/http/routers"] = errors.New("Process exited with status 6: private runtime detail")
			manager := newRouteManager(routeTargetResolver{targetType: tt.targetType}, runtime, testRouteTimeouts(), func() bool { return tt.inContainer })

			_, err := manager.listRouters(context.Background(), "project-1", testGateway(), "http")
			if err == nil {
				t.Fatal("expected Traefik request failure")
			}
			message, ok := manager.TraefikUnavailableMessage(err)
			if !ok || message != tt.want {
				t.Fatalf("TraefikUnavailableMessage() = (%q, %t), want (%q, true)", message, ok, tt.want)
			}
			if strings.Contains(message, "status 6") || strings.Contains(message, "private runtime detail") {
				t.Fatalf("unavailable message leaked runtime error: %q", message)
			}
		})
	}
}

func TestRouteManagerDoesNotClassifyConfigurationOrDecodeErrorsAsUnavailable(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.responses["/api/http/routers"] = "not json"
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return true })

	_, err := manager.listRouters(context.Background(), "project-1", testGateway(), "http")
	if err == nil {
		t.Fatal("expected Traefik response decode error")
	}
	if _, ok := manager.TraefikUnavailableMessage(err); ok {
		t.Fatalf("decode error was classified as unavailable: %v", err)
	}

	gateway := testGateway()
	gateway.RestApiUrl = ""
	_, err = manager.listRouters(context.Background(), "project-1", gateway, "http")
	if err == nil {
		t.Fatal("expected Gateway configuration error")
	}
	if _, ok := manager.TraefikUnavailableMessage(err); ok {
		t.Fatalf("configuration error was classified as unavailable: %v", err)
	}
}

func TestRouteManagerWaitUntilReadyPollsRemoteHost(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.failures["/api/overview"] = 2
	manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	if err := manager.WaitUntilReady(context.Background(), "project-1", testGateway(), 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if runtime.attempts["/api/overview"] != 3 {
		t.Fatalf("readiness attempts = %d", runtime.attempts["/api/overview"])
	}
}

func TestBuildRouteSnapshotSkipsDisabled(t *testing.T) {
	snapshot := buildRouteSnapshot([]model.Route{
		{Name: "a", Protocol: "http", Domain: "a.test", PathPrefix: "/", TargetUrl: "http://a:1", Enabled: true},
		{Name: "b", Protocol: "http", Domain: "b.test", PathPrefix: "/", TargetUrl: "http://b:1", Enabled: false},
	})
	httpCfg := snapshot["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 1 {
		t.Fatalf("expected one router: %#v", snapshot)
	}
}

func TestBuildRouteSnapshotUsesDNSResolverForDNSChallenge(t *testing.T) {
	snapshot := buildRouteSnapshot([]model.Route{{
		Id: "dns", Name: "dns", Protocol: "http", Domain: "dns.example.test", PathPrefix: "/", TargetUrl: "http://app:8080", Enabled: true,
		HTTPSEnabled: true, CertType: "letsencrypt", AcmeChallenge: "dns",
	}})
	router := snapshot["http"].(map[string]any)["routers"].(map[string]any)["route-dns-route"].(map[string]any)
	if got := router["tls"].(map[string]any)["certResolver"]; got != "letsencrypt-dns" {
		t.Fatalf("DNS cert resolver = %#v", got)
	}
}

func TestBuildRouteSnapshotUsesResolvedManagedHTTPTarget(t *testing.T) {
	snapshot := buildRouteSnapshot([]model.Route{{
		Id: "api", Name: "api", Protocol: "http", Domain: "api.test", PathPrefix: "/", TargetAddress: "api-api", TargetPort: 8080, Enabled: true,
	}})
	httpCfg := snapshot["http"].(map[string]any)
	service := httpCfg["services"].(map[string]any)["route-api-service"].(map[string]any)
	servers := service["loadBalancer"].(map[string]any)["servers"].([]map[string]string)
	if len(servers) != 1 || servers[0]["url"] != "http://api-api:8080" {
		t.Fatalf("unexpected managed HTTP servers: %+v", servers)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func testGateway() model.GatewayConfig {
	return model.GatewayConfig{RestApiUrl: model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl, RuntimeServiceCode: "traefik-default"}
}

type routeTargetResolver struct {
	targetType    string
	workspaceRoot string
	platform      string
}

func (r routeTargetResolver) ResolveProjectTarget(_ context.Context, projectId string) (environmentport.Target, error) {
	targetType := r.targetType
	if targetType == "" {
		targetType = model.EnvironmentTargetTypeLocal
	}
	environment := model.Environment{Id: "environment-1", ProjectId: projectId, TargetType: targetType, WorkspaceRoot: r.workspaceRoot}
	if targetType == model.EnvironmentTargetTypeSSH {
		environment.SSH = &model.EnvironmentSSHTarget{Platform: r.platform}
	}
	return environmentport.Target{Environment: environment}, nil
}

type routeRuntimeCommand struct {
	target      environmentport.Target
	serviceCode string
	name        string
	args        []string
}

type routeRuntimeFake struct {
	responses          map[string]string
	failures           map[string]int
	failureErrors      map[string]error
	attempts           map[string]int
	requests           []string
	files              map[string][]byte
	lastTarget         environmentport.Target
	environmentQueries int
	autoAPI            bool
	writeFailures      map[string]error
	removeFailures     map[string]error
	writes             []deploymentport.WorkspaceFile
	physicalBase       string
	containerId        string
	commands           []routeRuntimeCommand
	reloadRequired     bool
	reloadAttempts     int
	reloadFailures     map[int]error
	loadedFiles        map[string][]byte
}

func newRouteRuntimeFake() *routeRuntimeFake {
	return &routeRuntimeFake{responses: map[string]string{}, failures: map[string]int{}, failureErrors: map[string]error{}, attempts: map[string]int{}, files: map[string][]byte{}}
}

func (r *routeRuntimeFake) ServiceDir(_ environmentport.Target, serviceCode string) (string, error) {
	return "/srv/orbit/" + serviceCode, nil
}
func (r *routeRuntimeFake) ServiceDirExists(context.Context, environmentport.Target, string) (bool, error) {
	return true, nil
}
func (r *routeRuntimeFake) ComposeMountSourceDir(_ context.Context, target environmentport.Target, serviceCode string) (string, error) {
	if r.physicalBase != "" {
		return r.physicalBase, nil
	}
	return r.ServiceDir(target, serviceCode)
}
func (r *routeRuntimeFake) StageWorkspace(context.Context, environmentport.Target, deploymentport.Workspace) error {
	return nil
}
func (r *routeRuntimeFake) Run(context.Context, environmentport.Target, string, io.Writer, string, ...string) error {
	return nil
}
func (r *routeRuntimeFake) Stream(context.Context, environmentport.Target, string, io.Writer, string, ...string) error {
	return nil
}
func (r *routeRuntimeFake) Query(_ context.Context, target environmentport.Target, serviceCode string, name string, args ...string) (string, error) {
	r.commands = append(r.commands, routeRuntimeCommand{target: target, serviceCode: serviceCode, name: name, args: append([]string(nil), args...)})
	if name == "docker" && strings.Join(args, " ") == "compose ps -q traefik" {
		if r.containerId != "" {
			return r.containerId, nil
		}
		return "gateway-container", nil
	}
	if name == "docker" && len(args) == 3 && args[0] == "kill" && args[1] == "--signal=HUP" {
		r.reloadAttempts++
		if err := r.reloadFailures[r.reloadAttempts]; err != nil {
			return "", err
		}
		r.loadedFiles = make(map[string][]byte, len(r.files))
		for file, body := range r.files {
			r.loadedFiles[file] = append([]byte(nil), body...)
		}
		return args[2], nil
	}
	return r.query(target, args...)
}
func (r *routeRuntimeFake) QueryAtEnvironmentRoot(_ context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	return r.QueryAtEnvironmentRootInput(context.Background(), target, nil, name, args...)
}
func (r *routeRuntimeFake) QueryAtEnvironmentRootInput(ctx context.Context, target environmentport.Target, _ []byte, name string, args ...string) (string, error) {
	r.environmentQueries++
	return r.Query(ctx, target, "", name, args...)
}
func (r *routeRuntimeFake) query(target environmentport.Target, args ...string) (string, error) {
	r.lastTarget = target
	endpoint := args[len(args)-1]
	r.requests = append(r.requests, endpoint)
	if r.autoAPI && strings.Contains(endpoint, "/api/") {
		return r.configurationResponse(endpoint), nil
	}
	for suffix, response := range r.responses {
		if strings.HasSuffix(endpoint, suffix) {
			r.attempts[suffix]++
			if r.attempts[suffix] <= r.failures[suffix] {
				if err := r.failureErrors[suffix]; err != nil {
					return "", err
				}
				return "", errors.New("Process exited with status 7")
			}
			return response, nil
		}
	}
	if strings.HasSuffix(endpoint, "/api/overview") {
		r.attempts["/api/overview"]++
		if r.attempts["/api/overview"] <= r.failures["/api/overview"] {
			if err := r.failureErrors["/api/overview"]; err != nil {
				return "", err
			}
			return "", errors.New("Process exited with status 7")
		}
		return `{}`, nil
	}
	return "", errors.New("unexpected remote curl endpoint")
}

func (r *routeRuntimeFake) SyncFiles(_ context.Context, target environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	r.lastTarget = target
	keep := map[string]struct{}{}
	for _, file := range files {
		if err := r.writeFailures[file.Path]; err != nil {
			return err
		}
		if _, found := r.files[file.Path]; found && file.IgnoreIfExists {
			continue
		}
		r.writes = append(r.writes, file)
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

func (r *routeRuntimeFake) ReadFile(_ context.Context, _ environmentport.Target, name string) ([]byte, error) {
	body, found := r.files[name]
	if !found {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), body...), nil
}
func (r *routeRuntimeFake) ListFiles(_ context.Context, _ environmentport.Target, directory string) ([]string, error) {
	names := map[string]bool{}
	for name := range r.files {
		if strings.HasPrefix(name, directory+"/") {
			names[strings.Split(strings.TrimPrefix(name, directory+"/"), "/")[0]] = true
		}
	}
	items := []string{}
	for name := range names {
		items = append(items, name)
	}
	return items, nil
}
func (r *routeRuntimeFake) RemoveFile(_ context.Context, _ environmentport.Target, name string) error {
	if err := r.removeFailures[name]; err != nil {
		return err
	}
	delete(r.files, name)
	return nil
}

func (r *routeRuntimeFake) configurationResponse(endpoint string) string {
	items := []map[string]any{}
	files := r.files
	if r.reloadRequired {
		files = r.loadedFiles
	}
	for name, body := range files {
		if !strings.Contains(name, "/gateway/dynamic/") || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		var config dynamicConfiguration
		_ = yaml.Unmarshal(body, &config)
		protocol := config.HTTP
		if strings.Contains(endpoint, "/tcp/") {
			protocol = config.TCP
		}
		if strings.HasSuffix(endpoint, "/routers") {
			for key, value := range protocol.Routers {
				item := map[string]any{"name": key + "@file", "provider": "file", "status": "enabled", "rule": value.Rule, "service": value.Service, "entryPoints": value.EntryPoints}
				if value.TLS != nil {
					item["tls"] = value.TLS
				}
				items = append(items, item)
			}
		} else if strings.HasSuffix(endpoint, "/services") {
			for key, value := range protocol.Services {
				items = append(items, map[string]any{"name": key + "@file", "provider": "file", "status": "enabled", "loadBalancer": value.LoadBalancer})
			}
		}
	}
	body, _ := json.Marshal(items)
	return string(body)
}
