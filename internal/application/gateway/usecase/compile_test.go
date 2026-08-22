package gatewaysvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func testGatewayConfig() config.Config {
	return config.Config{
		Orbit:     config.OrbitConfig{Root: "/orbit"},
		Workspace: config.WorkspaceConfig{Deployment: "/orbit/work/cd"},
		Traefik: config.TraefikConfig{
			Image:      "traefik:3.6",
			RestApiUrl: "http://localhost:8080", BaseDomain: "lvh.me",
		},
		Cert: config.CertConfig{LetsEncrypt: config.LetsEncryptConfig{
			Enabled: true, Email: "ops@example.test", Challenge: "dns", DNSProvider: "cloudflare",
		}},
	}
}

func TestBuildInitialGatewayComponentExposesDashboardApiAndConfiguredCertDirectory(t *testing.T) {
	cfg := testGatewayConfig()
	component := buildInitialGatewayComponent(
		"version-1", "traefik:3.6", "missing",
		model.GatewayConfig{TLSMode: "letsencrypt"}, cfg.Cert,
		"/srv/orbit/cd/traefik/data/certs",
	)

	var api *model.VersionComponentEndpoint
	for index := range component.Endpoints {
		if component.Endpoints[index].Protocol == "http" && component.Endpoints[index].ContainerPort == 8080 {
			api = &component.Endpoints[index]
			break
		}
	}
	if api == nil {
		t.Fatal("initial gateway does not declare an API endpoint")
		return
	}
	if api.Protocol != "http" || api.ContainerPort != 8080 || api.Mode != "local" {
		t.Fatalf("unexpected API endpoint: %#v", *api)
	}
	if api.BindAddress == nil || *api.BindAddress != "127.0.0.1" {
		t.Fatalf("API endpoint must bind loopback, got %#v", api.BindAddress)
	}
	if api.ListenPort == nil || *api.ListenPort != 8080 {
		t.Fatalf("API endpoint must listen on 8080, got %#v", api.ListenPort)
	}
	if len(component.Mounts) != 4 {
		t.Fatalf("mounts = %#v", component.Mounts)
	}
	if component.Mounts[1].Source != "./traefik.yml" || component.Mounts[1].SourceIsHostPath {
		t.Fatalf("controlled Traefik config mount = %#v", component.Mounts[1])
	}
	if component.Mounts[2].Source != "/srv/orbit/cd/traefik/data/certs" || !component.Mounts[2].SourceIsHostPath {
		t.Fatalf("certificate directory mount = %#v", component.Mounts[2])
	}
	if content := component.Mounts[1].Content; !containsAll(content, "ops@example.test", "dnsChallenge", "cloudflare") {
		t.Fatalf("static config did not use letsencrypt config: %q", content)
	}
}

func TestInitialGatewayComponentResolvesDeploymentCertificateDirectoryToHost(t *testing.T) {
	cfg := testGatewayConfig()
	var resolved string
	service := Service{
		cert:           cfg.Cert,
		deploymentRoot: cfg.Workspace.Deployment,
		resolvePath: func(_ context.Context, logicalPath string) (string, error) {
			resolved = logicalPath
			return "/srv/orbit/cd/traefik/data/certs", nil
		},
	}
	component, err := service.initialGatewayComponent(context.Background(), "version-1", "traefik:3.6", "missing", model.GatewayConfig{})
	if err != nil {
		t.Fatal(err)
	}
	wantLogical := filepath.Join(cfg.Workspace.Deployment, "traefik", "data", "certs")
	if resolved != wantLogical {
		t.Fatalf("resolved logical certificate directory = %q, want %q", resolved, wantLogical)
	}
	if component.Mounts[2].Source != "/srv/orbit/cd/traefik/data/certs" {
		t.Fatalf("certificate mount source = %q", component.Mounts[2].Source)
	}
}

func TestInitialGatewayComponentUsesLogicalCertificateDirectoryWithoutDockerResolver(t *testing.T) {
	cfg := testGatewayConfig()
	service := Service{cert: cfg.Cert, deploymentRoot: cfg.Workspace.Deployment}
	component, err := service.initialGatewayComponent(context.Background(), "version-1", "traefik:3.6", "missing", model.GatewayConfig{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cfg.Workspace.Deployment, "traefik", "data", "certs")
	if component.Mounts[2].Source != want {
		t.Fatalf("certificate mount source = %q, want %q", component.Mounts[2].Source, want)
	}
}

func containsAll(value string, expected ...string) bool {
	for _, item := range expected {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}
