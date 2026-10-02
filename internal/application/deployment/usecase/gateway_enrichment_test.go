package deploymentsvc

import (
	"reflect"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestEnrichGatewayPlanPatchesDeclaredResolverAndInjectsDNSToken(t *testing.T) {
	plan := testGatewayEnrichmentPlan("http-dns", "ops@example.test", "cfat_gateway_token")
	if err := enrichGatewayPlan(&plan); err != nil {
		t.Fatal(err)
	}
	component := plan.Components[0]
	if !strings.Contains(component.Mounts[0].Content, `email: "ops@example.test"`) {
		t.Fatalf("resolver email was not patched: %s", component.Mounts[0].Content)
	}
	if !strings.Contains(component.Mounts[0].Content, "letsencrypt:") || !strings.Contains(component.Mounts[0].Content, "letsencrypt-dns:") {
		t.Fatalf("declared resolver structure changed: %s", component.Mounts[0].Content)
	}
	if len(component.Env) != 1 || component.Env[0].Key != gatewayDNSApiEnvKey || component.Env[0].Value != "cfat_gateway_token" {
		t.Fatalf("effective environment = %#v", component.Env)
	}
}

func TestEnrichGatewayBaseProfileDoesNotRequireEmailOrToken(t *testing.T) {
	plan := testGatewayEnrichmentPlan("", "", "")
	if err := enrichGatewayPlan(&plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Components[0].Env) != 0 {
		t.Fatalf("base profile added environment: %#v", plan.Components[0].Env)
	}
}

func TestEnrichGatewayPlanReplacesManagedProviderAndMounts(t *testing.T) {
	plan := testGatewayEnrichmentPlan("", "", "")
	plan.Components[0].Mounts[0].Content = `api:
  dashboard: true
entryPoints:
  web:
    address: ":8081"
providers:
  providersThrottleDuration: 3s
  docker:
    endpoint: unix:///var/run/docker.sock
    exposedByDefault: false
    network: custom-network
  rest:
    insecure: true
  file:
    filename: /obsolete/routes.yaml
    watch: false
log:
  level: DEBUG
certificatesResolvers:
  custom:
    acme:
      email: custom@example.test
      storage: /letsencrypt/custom.json
`
	plan.Components[0].Mounts[0].IgnoreIfExists = true
	plan.Components[0].Mounts[1] = model.VersionComponentMount{SourceType: "named_volume", Source: "obsolete", Target: model.GatewayRouteConfigTarget}
	plan.Components[0].Mounts[2].ReadOnly = false
	plan.Components[0].Mounts = append(plan.Components[0].Mounts[:3], model.VersionComponentMount{SourceType: "directory", Source: "./custom", Target: "/custom", ReadOnly: true})
	customMount := plan.Components[0].Mounts[3]
	var before map[string]any
	if err := yaml.Unmarshal([]byte(plan.Components[0].Mounts[0].Content), &before); err != nil {
		t.Fatal(err)
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		t.Fatal(err)
	}
	var after map[string]any
	if err := yaml.Unmarshal([]byte(plan.Components[0].Mounts[0].Content), &after); err != nil {
		t.Fatal(err)
	}
	providers := after["providers"].(map[string]any)
	if _, found := providers["rest"]; found {
		t.Fatal("REST provider is still enabled")
	}
	if !reflect.DeepEqual(providers["file"], map[string]any{"directory": model.GatewayRouteConfigTarget, "watch": true}) {
		t.Fatalf("File provider = %#v", providers["file"])
	}
	delete(providers, "file")
	delete(before["providers"].(map[string]any), "file")
	delete(before["providers"].(map[string]any), "rest")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("unmanaged static config changed: %#v", after)
	}
	if plan.Components[0].Mounts[0].IgnoreIfExists {
		t.Fatal("static config will not overwrite the existing file")
	}
	for _, want := range []model.VersionComponentMount{
		{SourceType: "directory", Source: model.GatewayRouteConfigSource, Target: model.GatewayRouteConfigTarget, ReadOnly: true},
		{SourceType: "directory", Source: "./gateway/certs", Target: "/etc/traefik/certs", ReadOnly: true},
		{SourceType: "directory", Source: "./gateway/acme", Target: "/letsencrypt"},
		customMount,
	} {
		found := false
		for _, mount := range plan.Components[0].Mounts {
			if reflect.DeepEqual(mount, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing mount: %+v", want)
		}
	}
	if len(plan.Components[0].Mounts) != 5 {
		t.Fatalf("mounts = %+v", plan.Components[0].Mounts)
	}
	hash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := enrichGatewayPlan(&plan); err != nil {
		t.Fatal(err)
	}
	repeatedHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	if hash != repeatedHash {
		t.Fatal("repeated enrichment changed the deployment plan hash")
	}
}

func testGatewayEnrichmentPlan(profile, email, token string) model.EffectiveServicePlan {
	role := profile
	if role == "" {
		role = "base"
	}
	versionId := "version-" + role
	return model.EffectiveServicePlan{
		Application: model.Application{Id: "gateway-1", Code: "traefik", Kind: status.ApplicationKindStandard},
		Version:     model.Version{Id: versionId},
		Gateway: &model.GatewayConfig{
			ApplicationId: "gateway-1", NetworkName: "traefik", AcmeProfile: profile, AcmeEmail: email, DNSApiToken: token,
			InternalDomain: "example.test", DefaultEntrypoint: "web",
			VersionBindings: []model.GatewayVersionBinding{{Profile: role, VersionId: versionId}},
		},
		Components: []model.EffectiveServiceComponent{{
			Name: "traefik",
			Mounts: []model.VersionComponentMount{{
				SourceType: "controlled_file", Source: "./traefik.yml", Target: gatewayMountTargetTraefikYml, Mode: "0644", Content: `providers:
  file:
    directory: /etc/traefik/dynamic
    watch: true
certificatesResolvers:
  letsencrypt:
    acme:
      email: ""
  letsencrypt-dns:
    acme:
      email: ""
`,
			},
				{SourceType: "directory", Source: model.GatewayRouteConfigSource, Target: model.GatewayRouteConfigTarget, ReadOnly: true},
				{SourceType: "directory", Source: "./gateway/certs", Target: "/etc/traefik/certs", ReadOnly: true},
				{SourceType: "directory", Source: "./gateway/acme", Target: "/letsencrypt"},
			},
		}},
	}
}
