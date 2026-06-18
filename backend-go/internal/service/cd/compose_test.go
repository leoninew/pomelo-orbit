package cdsvc

import (
	"strings"
	"testing"

	"backend/internal/config"
	"backend/internal/repository/model"
)

func TestRenderApplicationTemplateUsesLiquidSyntax(t *testing.T) {
	cfg := config.Config{
		Orbit:   config.OrbitConfig{Root: "../.."},
		Traefik: config.TraefikConfig{DomainSuffix: "lvh.me"},
		Cert:    config.CertConfig{LetsEncrypt: config.LetsEncryptConfig{Email: "ops@example.com", Challenge: "http"}},
	}
	got, err := renderApplicationTemplate("{{ app.code }}.{{ config.domain_suffix }} {{ cert.letsencrypt.email }}", "demo", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != "demo.lvh.me ops@example.com" {
		t.Fatalf("unexpected render output: %q", got)
	}
}

func TestRenderApplicationTemplateSupportsLiquidDefaultFilter(t *testing.T) {
	got, err := renderApplicationTemplate("{{ image | default: 'nginx' }}", "demo", config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "nginx" {
		t.Fatalf("unexpected render output: %q", got)
	}
}

func TestRenderApplicationTemplateRejectsMissingVariable(t *testing.T) {
	_, err := renderApplicationTemplate("{{ app.missing }}", "demo", config.Config{})
	if err == nil {
		t.Fatal("expected missing variable error")
	}
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
