package cdsvc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"backend/internal/config"
	"backend/internal/repository/model"
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
	service := Service{cfg: cfg, workspace: newWorkspaceWithResolver(cfg.DataRoot(), func(ctx context.Context, logicalDataRoot string) (string, error) {
		return physicalRoot, nil
	})}

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
	return Service{cfg: cfg, workspace: newWorkspaceWithResolver(cfg.DataRoot(), func(ctx context.Context, logicalDataRoot string) (string, error) {
		return logicalDataRoot, nil
	})}
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
