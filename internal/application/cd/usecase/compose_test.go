package cdsvc

import (
	"context"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeFromComponents(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	version := model.Version{Id: "v1", Label: "v1"}
	envJSON := `{"APP_ENV":"prod"}`
	version.EnvJSON = &envJSON
	componentEnv := `{"APP_ENV":"stage","DEBUG":"1"}`
	ports := `["8080:80"]`
	depends := `["db"]`
	components := []model.Component{
		{Name: "db", Image: "postgres:16"},
		{Name: "web", Image: "nginx:1.27", EnvJSON: &componentEnv, PortsJSON: &ports, DependsOnJSON: &depends},
	}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:        model.Application{Id: "app-1", Code: "demo"},
		Version:    version,
		Components: components,
		Env:        model.Environment{Id: "env-1", Code: "local"},
		Service:    model.Service{InstanceKey: "default", IsIngress: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "image: nginx:1.27") {
		t.Fatalf("expected web image, got:\n%s", got)
	}
	if !strings.Contains(got, "APP_ENV: stage") {
		t.Fatalf("expected component env override, got:\n%s", got)
	}
	if !strings.Contains(got, "DEBUG: \"1\"") && !strings.Contains(got, "DEBUG: 1") {
		t.Fatalf("expected merged DEBUG env, got:\n%s", got)
	}
	if !strings.Contains(got, "depends_on") {
		t.Fatalf("expected depends_on, got:\n%s", got)
	}
}

func TestRenderComposeRejectsDuplicateComponentNames(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.Component{
			{Name: "web", Image: "nginx"},
			{Name: "web", Image: "nginx:2"},
		},
		Env:     model.Environment{Code: "local"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err == nil {
		t.Fatal("expected duplicate component error")
	}
}

func TestRenderComposeRejectsMissingDependsOn(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	depends := `["missing"]`
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.Component{
			{Name: "web", Image: "nginx", DependsOnJSON: &depends},
		},
		Env:     model.Environment{Code: "local"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err == nil {
		t.Fatal("expected missing depends_on error")
	}
}

func TestRenderComposeInjectsHTTPExposeLabels(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.Component{
			{Name: "web", Image: "nginx"},
		},
		Exposes: []model.Expose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "example.com",
			DefaultEntrypoint: "websecure",
			TLSMode:           "letsencrypt",
		},
		Service: model.Service{InstanceKey: "default", IsIngress: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRule := "traefik.http.routers.demo-local-default-web-http.rule=Host(`demo.example.com`)"
	if !strings.Contains(got, wantRule) {
		t.Fatalf("expected derived host rule %q, got:\n%s", wantRule, got)
	}
	if !strings.Contains(got, "traefik.http.services.demo-local-default-web-http.loadbalancer.server.port=80") {
		t.Fatalf("expected service port label, got:\n%s", got)
	}
	if !strings.Contains(got, "traefik.http.routers.demo-local-default-web-http.tls.certresolver=letsencrypt") {
		t.Fatalf("expected letsencrypt label, got:\n%s", got)
	}
}

func TestRenderComposeRejectsIncompleteHTTPPolicy(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.Component{
			{Name: "web", Image: "nginx"},
		},
		Exposes: []model.Expose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
		},
		Env:     model.Environment{Code: "local", DefaultEntrypoint: "web"},
		Service: model.Service{InstanceKey: "default", IsIngress: true},
	})
	if err == nil {
		t.Fatal("expected incomplete policy error")
	}
}

func TestRenderComposeRejectsDuplicateHTTPPath(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.Component{
			{Name: "web", Image: "nginx"},
			{Name: "api", Image: "api"},
		},
		Exposes: []model.Expose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
			{ComponentName: "api", Protocol: "http", ContainerPort: 8080},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "local.test",
			DefaultEntrypoint: "web",
			TLSMode:           "none",
		},
		Service: model.Service{InstanceKey: "default", IsIngress: true},
	})
	if err == nil {
		t.Fatal("expected duplicate host+path error")
	}
}

func TestValidDeploymentRouteDomain(t *testing.T) {
	invalidDomains := []string{"bad host.example.com", "bad`host.example.com", "host with\ttab.example.com", ""}
	for _, domain := range invalidDomains {
		if validDeploymentRouteDomain(domain) {
			t.Fatalf("expected domain %q to be rejected", domain)
		}
	}
	if !validDeploymentRouteDomain("orbit.example.com") {
		t.Fatal("expected valid domain")
	}
}
