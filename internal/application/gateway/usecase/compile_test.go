package gatewaysvc

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func testGatewayConfig() config.Config {
	return config.Config{
		Orbit: config.OrbitConfig{Root: "/orbit"},
		Traefik: config.TraefikConfig{
			CertDir: "data/deployment/traefik/data/certs", Image: "traefik:3.6",
			RestApiUrl: "http://localhost:8080", BaseDomain: "lvh.me",
		},
		Cert: config.CertConfig{LetsEncrypt: config.LetsEncryptConfig{
			Enabled: true, Email: "ops@example.test", Challenge: "dns", DNSProvider: "cloudflare",
		}},
	}
}

func TestBuildInitialGatewayComponentExposesDashboardApiAndConfiguredCertDirectory(t *testing.T) {
	cfg := testGatewayConfig()
	component, err := buildInitialGatewayComponent(
		"version-1", "traefik:3.6", "missing",
		model.GatewayConfig{TLSMode: "letsencrypt"}, cfg.Cert,
		filepath.Join(cfg.OrbitRoot(), cfg.Traefik.CertDir),
	)
	if err != nil {
		t.Fatalf("build managed gateway component: %v", err)
	}

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
	if component.Mounts[2].Source != filepath.Join(cfg.OrbitRoot(), cfg.Traefik.CertDir) || !component.Mounts[2].SourceIsHostPath {
		t.Fatalf("certificate directory mount = %#v", component.Mounts[2])
	}
	if content := component.Mounts[1].Content; !containsAll(content, "ops@example.test", "dnsChallenge", "cloudflare") {
		t.Fatalf("static config did not use letsencrypt config: %q", content)
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
