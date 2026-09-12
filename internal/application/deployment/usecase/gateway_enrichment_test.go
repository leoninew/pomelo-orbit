package deploymentsvc

import (
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
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

func testGatewayEnrichmentPlan(profile, email, token string) model.EffectiveServicePlan {
	role := profile
	if role == "" {
		role = "base"
	}
	versionID := "version-" + role
	return model.EffectiveServicePlan{
		Application: model.Application{Id: "gateway-1", Code: "traefik"},
		Version:     model.Version{Id: versionID},
		Gateway: &model.GatewayConfig{
			ApplicationId: "gateway-1", NetworkName: "traefik", AcmeProfile: profile, AcmeEmail: email, DNSApiToken: token,
			VersionBindings: []model.GatewayVersionBinding{{Profile: role, VersionId: versionID}},
		},
		Components: []model.EffectiveServiceComponent{{
			Name: "traefik",
			Mounts: []model.VersionComponentMount{{
				SourceType: "controlled_file", Target: gatewayMountTargetTraefikYml, Content: `certificatesResolvers:
  letsencrypt:
    acme:
      email: ""
  letsencrypt-dns:
    acme:
      email: ""
`,
			}},
		}},
	}
}
