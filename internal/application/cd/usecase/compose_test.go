package cdsvc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRenderComposeFromComponents(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	version := model.Version{Id: "v1", Label: "v1"}
	envJSON := `[{"key":"APP_ENV","value":"prod"}]`
	version.EnvJSON = &envJSON
	componentEnv := `[{"key":"APP_ENV","value":"stage"},{"key":"DEBUG","value":"1"}]`
	ports := `["8080:80"]`
	depends := `["db"]`
	components := []model.VersionComponent{
		{Name: "db", Image: "postgres:16"},
		{Name: "web", Image: "nginx:1.27", EnvJSON: &componentEnv, PortsJSON: &ports, DependsOnJSON: &depends},
	}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:        model.Application{Id: "app-1", Code: "demo", Kind: "standard"},
		Version:    version,
		Components: components,
		Env:        model.Environment{Id: "env-1", Code: "local"},
		Service:    model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "image: nginx:1.27") {
		t.Fatalf("expected web image, got:\n%s", got)
	}
	if !strings.Contains(got, "container_name: demo_web") {
		t.Fatalf("expected container_name demo_web, got:\n%s", got)
	}
	if !strings.Contains(got, "container_name: demo_db") {
		t.Fatalf("expected container_name demo_db, got:\n%s", got)
	}
	if !strings.Contains(got, "APP_ENV: stage") {
		t.Fatalf("expected component env override, got:\n%s", got)
	}
	if !strings.Contains(got, "DEBUG: \"1\"") && !strings.Contains(got, "DEBUG: 1") {
		t.Fatalf("expected merged DEBUG env, got:\n%s", got)
	}
	if !strings.Contains(got, "depends_on") || !strings.Contains(got, "- db") {
		t.Fatalf("expected depends_on with service key db, got:\n%s", got)
	}
	if strings.Contains(got, "traefik.") {
		t.Fatalf("expected no traefik labels without exposes, got:\n%s", got)
	}
}

func TestRenderComposeRejectsDuplicateComponentNames(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
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
		Components: []model.VersionComponent{
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
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
		},
		Exposes: []model.VersionExpose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "example.com",
			DefaultEntrypoint: "websecure",
			TLSMode:           "letsencrypt",
		},
		Service: model.Service{InstanceKey: "default"},
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
	if !strings.Contains(got, "container_name: demo_web") {
		t.Fatalf("expected container_name, got:\n%s", got)
	}
	// E1: standard + Expose joins gateway network as external consumer.
	if !strings.Contains(got, "external: true") {
		t.Fatalf("expected platform network external:true for consumer, got:\n%s", got)
	}
	if !strings.Contains(got, "name: traefik") {
		t.Fatalf("expected platform network name traefik, got:\n%s", got)
	}
	if !strings.Contains(got, "- default") || !strings.Contains(got, "- traefik") {
		t.Fatalf("expected service networks default + traefik, got:\n%s", got)
	}
}

func TestRenderComposeStandardWithoutExposeDoesNotJoinPlatformNetwork(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
		},
		Env:     model.Environment{Code: "local", BaseDomain: "example.com", DefaultEntrypoint: "web"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "external: true") {
		t.Fatalf("standard without expose must not inject platform network, got:\n%s", got)
	}
}

func TestRenderComposeStandardExposeOnlyJoinsExposedComponents(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
			{Name: "db", Image: "postgres:16"},
		},
		Exposes: []model.VersionExpose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "example.com",
			DefaultEntrypoint: "web",
			TLSMode:           "none",
		},
		Service: model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// web service block should list both networks; db should not get platform network attachment lines
	// in a multi-service yaml this is approximate: ensure top-level external network exists
	// and web has network list. db with no networks key is correct (only project default).
	if !strings.Contains(got, "external: true") {
		t.Fatalf("expected external platform network, got:\n%s", got)
	}
	// crude: after "db:" section should not contain "- traefik" before next top-level-ish marker;
	// assert web section includes traefik by checking full yaml has service networks on web only once is hard;
	// check that "db:" appears and the networks list under services isn't forced on every service via counting.
	webIdx := strings.Index(got, "web:")
	dbIdx := strings.Index(got, "db:")
	if webIdx < 0 || dbIdx < 0 {
		t.Fatalf("expected web and db services, got:\n%s", got)
	}
	// Isolate web service body between web: and db: (or reverse order from yaml map)
	var webBody, dbBody string
	if webIdx < dbIdx {
		webBody = got[webIdx:dbIdx]
		dbBody = got[dbIdx:]
	} else {
		dbBody = got[dbIdx:webIdx]
		webBody = got[webIdx:]
	}
	if !strings.Contains(webBody, "traefik") {
		t.Fatalf("expected web to join traefik network, web body:\n%s", webBody)
	}
	if strings.Contains(dbBody, "traefik") {
		t.Fatalf("db without expose must not join traefik network, db body:\n%s", dbBody)
	}
}

func TestRenderComposeRejectsIncompleteHTTPPolicy(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
		},
		Exposes: []model.VersionExpose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
		},
		Env:     model.Environment{Code: "local", DefaultEntrypoint: "web"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err == nil {
		t.Fatal("expected incomplete policy error")
	}
}

func TestRenderComposeRejectsDuplicateHTTPPath(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
			{Name: "api", Image: "api"},
		},
		Exposes: []model.VersionExpose{
			{ComponentName: "web", Protocol: "http", ContainerPort: 80},
			{ComponentName: "api", Protocol: "http", ContainerPort: 8080},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "local.test",
			DefaultEntrypoint: "web",
			TLSMode:           "none",
		},
		Service: model.Service{InstanceKey: "default"},
	})
	if err == nil {
		t.Fatal("expected duplicate host+path error")
	}
}

func TestRenderComposeGatewayKindDoesNotUseCodeMagic(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "traefik", Kind: "gateway"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "proxy", Image: "traefik:v3"},
		},
		Env:     model.Environment{Code: "local"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "container_name: traefik_proxy") {
		t.Fatalf("expected gateway container_name, got:\n%s", got)
	}
	if !strings.Contains(got, "image: traefik:v3") {
		t.Fatalf("expected component image, got:\n%s", got)
	}
	// Incomplete IngressPolicy → no dashboard labels (ports-only still ok).
	if strings.Contains(got, "traefik.enable") {
		t.Fatalf("gateway without ingress policy should not inject labels, got:\n%s", got)
	}
	if !strings.Contains(got, "networks:") || !strings.Contains(got, "name: traefik") {
		t.Fatalf("expected gateway top-level network, got:\n%s", got)
	}
}

func TestRenderComposeGatewayInjectsDashboardLabelsFromEnv(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "traefik", Kind: "gateway"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "proxy", Image: "traefik:v3"},
		},
		// No Expose: dashboard labels must still appear for gateway.
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "local.test",
			DefaultEntrypoint: "web",
			TLSMode:           "none",
		},
		Service: model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "traefik.enable=true") {
		t.Fatalf("expected traefik.enable, got:\n%s", got)
	}
	if !strings.Contains(got, "Host(`traefik.local.test`)") {
		t.Fatalf("expected Host from env policy, got:\n%s", got)
	}
	if !strings.Contains(got, "entrypoints=web") {
		t.Fatalf("expected entrypoint from env, got:\n%s", got)
	}
	if !strings.Contains(got, "service=api@internal") {
		t.Fatalf("expected api@internal service, got:\n%s", got)
	}
	if strings.Contains(got, "loadbalancer.server.port") {
		t.Fatalf("gateway dashboard must not use loadbalancer port labels, got:\n%s", got)
	}
}

func TestRenderComposeStandardDoesNotInjectGatewayDashboard(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "standard"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
		},
		Env: model.Environment{
			Code:              "local",
			BaseDomain:        "local.test",
			DefaultEntrypoint: "web",
		},
		Service: model.Service{InstanceKey: "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "api@internal") || strings.Contains(got, "traefik.enable") {
		t.Fatalf("standard without expose must not get gateway dashboard labels, got:\n%s", got)
	}
}

func TestRenderComposeResolvesLogicalMounts(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	mounts := `[{"source_type":"logical","source":"data/acme.json","target":"/etc/traefik/acme.json"},{"source_type":"special","source":"docker.sock","target":"/var/run/docker.sock"}]`
	got, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "gw", Kind: "gateway"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "proxy", Image: "traefik:v3", MountsJSON: &mounts},
		},
		Env:            model.Environment{Code: "local"},
		Service:        model.Service{InstanceKey: "default"},
		PhysicalSvcDir: "/data/cd/gw/local/default",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "/data/cd/gw/local/default/data/acme.json:/etc/traefik/acme.json") {
		t.Fatalf("expected resolved acme mount, got:\n%s", got)
	}
	if !strings.Contains(got, "/var/run/docker.sock:/var/run/docker.sock:ro") {
		t.Fatalf("expected docker.sock special mount, got:\n%s", got)
	}
}

func TestMaterializeLogicalMountSourcesCreatesFileOnce(t *testing.T) {
	dir := t.TempDir()
	filePath := dir + "/data/acme.json"
	resolved := []ResolvedMount{{
		SourceType:  mountSourceLogical,
		HostSource:  filePath,
		IsFile:      true,
		ContentMode: contentModeSeed,
	}}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "keep" {
		t.Fatalf("expected existing file not overwritten, got %q", raw)
	}
}

func TestMaterializeLogicalMountSourcesSeedWritesContentOnce(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data", "traefik.yml")
	resolved := []ResolvedMount{{
		SourceType:  mountSourceLogical,
		HostSource:  filePath,
		IsFile:      true,
		Content:     "api:\n  dashboard: true\n",
		ContentMode: contentModeSeed,
	}}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "api:\n  dashboard: true\n" {
		t.Fatalf("expected seed content, got %q", raw)
	}
	// existing must not be overwritten on seed
	if err := os.WriteFile(filePath, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "changed" {
		t.Fatalf("seed must not overwrite existing, got %q", raw)
	}
}

func TestMaterializeLogicalMountSourcesSyncOverwrites(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "data", "traefik.yml")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved := []ResolvedMount{{
		SourceType:  mountSourceLogical,
		HostSource:  filePath,
		IsFile:      true,
		Content:     "new-config",
		ContentMode: contentModeSync,
	}}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "new-config" {
		t.Fatalf("expected sync overwrite, got %q", raw)
	}
}

func TestParseMountSpecsRejectsContentOnDirectory(t *testing.T) {
	raw := `[{"source_type":"logical","source":"data/logs","target":"/logs","content":"x"}]`
	_, err := parseMountSpecs(&raw)
	if err == nil {
		t.Fatal("expected content on directory mount to fail")
	}
}

func TestParseMountSpecsRejectsUnixAbsoluteLogicalSource(t *testing.T) {
	// Common misconfig: paste host docker.sock path as logical instead of special/docker.sock.
	raw := `[{"source_type":"logical","source":"/var/run/docker.sock","target":"/var/run/docker.sock","read_only":true}]`
	_, err := parseMountSpecs(&raw)
	if err == nil {
		t.Fatal("expected absolute logical source to fail")
	}
}

func TestParseMountSpecsAcceptsSpecialDockerSock(t *testing.T) {
	raw := `[{"source_type":"special","source":"docker.sock","target":"/var/run/docker.sock"}]`
	got, err := parseMountSpecs(&raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SourceType != mountSourceSpecial || !got[0].ReadOnly {
		t.Fatalf("expected special docker.sock with default ro, got %+v", got)
	}
	resolved, err := resolveMountSpecs(got, `/data/cd/app/env/svc`)
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Compose != "/var/run/docker.sock:/var/run/docker.sock:ro" {
		t.Fatalf("expected host docker.sock compose entry, got %q", resolved[0].Compose)
	}
}

func TestRenderComposeRejectsUnknownKind(t *testing.T) {
	service := Service{cfg: config.Config{}, executionStore: &fakeDeploymentExecutionStore{}}
	_, err := service.RenderCompose(context.Background(), RenderInput{
		App:     model.Application{Code: "demo", Kind: "weird"},
		Version: model.Version{Id: "v1"},
		Components: []model.VersionComponent{
			{Name: "web", Image: "nginx"},
		},
		Env:     model.Environment{Code: "local"},
		Service: model.Service{InstanceKey: "default"},
	})
	if err == nil {
		t.Fatal("expected unknown kind error")
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
