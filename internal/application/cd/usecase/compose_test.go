package cdsvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderApplicationTemplateUsesLiquidSyntax(t *testing.T) {
	cfg := config.Config{
		Orbit:   config.OrbitConfig{Root: t.TempDir()},
		Traefik: config.TraefikConfig{DomainSuffix: "lvh.me"},
		Cert:    config.CertConfig{LetsEncrypt: config.LetsEncryptConfig{Email: "ops@example.com", Challenge: "http"}},
	}
	service := newTemplateTestService(t, cfg)
	got, err := service.renderApplicationTemplate(context.Background(), "{{ app.code }}.{{ config.domain_suffix }} {{ cert.letsencrypt.email }}", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "demo.lvh.me ops@example.com" {
		t.Fatalf("unexpected render output: %q", got)
	}
}

func TestRenderApplicationTemplateSupportsLiquidDefaultFilter(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	service := newTemplateTestService(t, cfg)
	got, err := service.renderApplicationTemplate(context.Background(), "{{ image | default: 'nginx' }}", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "nginx" {
		t.Fatalf("unexpected render output: %q", got)
	}
}

func TestRenderApplicationTemplateRejectsMissingVariable(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	service := newTemplateTestService(t, cfg)
	_, err := service.renderApplicationTemplate(context.Background(), "{{ app.missing }}", "demo")
	if err == nil {
		t.Fatal("expected missing variable error")
	}
}

func TestRenderApplicationTemplateUsesPhysicalPaths(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	physicalRoot := filepath.Join(t.TempDir(), "host-data")
	service := Service{cfg: cfg, workspace: testWorkspaceWithPhysicalRoot(cfg.DataRoot(), physicalRoot, nil)}

	got, err := service.renderApplicationTemplate(context.Background(), "{{ app.physical_dir }} {{ app.physical_app_dir }}", "demo")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(physicalRoot) + " " + filepath.ToSlash(filepath.Join(physicalRoot, "cd", "demo"))
	if got != want {
		t.Fatalf("unexpected physical paths: %q", got)
	}
}

func newTemplateTestService(t *testing.T, cfg config.Config) Service {
	t.Helper()
	return Service{cfg: cfg, workspace: testWorkspace(cfg.DataRoot())}
}

func TestInjectApplicationRouteLabelsMatchesManagedRouteSemantics(t *testing.T) {
	compose := "services:\n  web:\n    image: nginx\n    labels:\n      - keep=false\n"
	got, err := injectApplicationRouteLabels(compose, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "keep=false") {
		t.Fatalf("expected managed route labels to be cleared, got:\n%s", got)
	}
}

func TestInjectApplicationRouteLabelsRejectsMissingService(t *testing.T) {
	compose := "services:\n  web:\n    image: nginx\n"
	_, err := injectApplicationRouteLabels(compose, []model.ApplicationRoute{{ServiceName: "api", Domain: "api.example.com", Port: 8080}}, false)
	if err == nil {
		t.Fatal("expected missing service error")
	}
}

func TestInjectApplicationRouteLabelsMergesDomainsForSameService(t *testing.T) {
	compose := "services:\n  pomelo-orbit:\n    image: pomelo-orbit\n"
	routes := []model.ApplicationRoute{
		{ServiceName: "pomelo-orbit", Domain: "orbit.typing-island.site", Port: 80},
		{ServiceName: "pomelo-orbit", Domain: "orbit.preflite.cn", Port: 80},
	}

	got, err := injectApplicationRouteLabels(compose, routes, true)
	if err != nil {
		t.Fatal(err)
	}
	wantRule := "traefik.http.routers.pomelo-orbit.rule=Host(`orbit.typing-island.site`) || Host(`orbit.preflite.cn`)"
	if !strings.Contains(got, wantRule) {
		t.Fatalf("expected merged router rule %q, got:\n%s", wantRule, got)
	}
	if !strings.Contains(got, "traefik.http.services.pomelo-orbit.loadbalancer.server.port=80") {
		t.Fatalf("expected service port label, got:\n%s", got)
	}
	if !strings.Contains(got, "traefik.http.routers.pomelo-orbit.tls.certresolver=letsencrypt") {
		t.Fatalf("expected letsencrypt label, got:\n%s", got)
	}
}

func TestInjectApplicationRouteLabelsRejectsSameServiceWithDifferentPorts(t *testing.T) {
	compose := "services:\n  web:\n    image: nginx\n"
	routes := []model.ApplicationRoute{
		{ServiceName: "web", Domain: "web.example.com", Port: 80},
		{ServiceName: "web", Domain: "api.example.com", Port: 8080},
	}

	_, err := injectApplicationRouteLabels(compose, routes, false)
	if err == nil {
		t.Fatal("expected different ports error")
	}
}

func TestNormalizeApplicationRouteInputRejectsInvalidDomain(t *testing.T) {
	invalidDomains := []string{"bad host.example.com", "bad`host.example.com", "-bad.example.com", "bad-.example.com", ""}
	for _, domain := range invalidDomains {
		_, _, _, err := normalizeApplicationRouteInput("web", domain, 80)
		if err == nil {
			t.Fatalf("expected domain %q to be rejected", domain)
		}
	}
}

func TestNormalizeApplicationRouteInputNormalizesDomain(t *testing.T) {
	_, domain, _, err := normalizeApplicationRouteInput("web", " Orbit.Example.COM ", 80)
	if err != nil {
		t.Fatal(err)
	}
	if domain != "orbit.example.com" {
		t.Fatalf("unexpected normalized domain: %q", domain)
	}
}
