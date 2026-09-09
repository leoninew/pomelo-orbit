package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestRenderGatewayComposeUsesDeclaredVersionTopology(t *testing.T) {
	baseVersionID := "version-1"
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{
		LogicalSvcDir:         t.TempDir(),
		ComposeMountSourceDir: t.TempDir(),
		Plan: model.EffectiveServicePlan{
			Application: model.Application{Id: "gateway-1", Code: "traefik", Kind: status.ApplicationKindStandard},
			Version:     model.Version{Id: baseVersionID},
			Service:     model.Service{InstanceKey: "default"},
			Gateway: &model.GatewayConfig{
				ApplicationId: "gateway-1", TraefikComponentName: "traefik", NetworkName: "traefik",
				BaseDomain: "example.test", DefaultEntrypoint: "web",
				VersionBindings: []model.GatewayVersionBinding{{Profile: "base", VersionId: baseVersionID}},
			},
			Components: []model.EffectiveServiceComponent{{
				Name: "traefik", Image: "traefik:3.6", PullPolicy: "missing",
				Mounts: []model.VersionComponentMount{{
					SourceType: "controlled_file", Source: "./traefik.yml", Target: gatewayMountTargetTraefikYml, Content: "entryPoints: {}\n", Mode: "0644",
				}},
				Endpoints: []model.VersionComponentEndpoint{
					{Protocol: "tcp", ContainerPort: 80, Mode: "host", BindAddress: stringPointer("0.0.0.0"), ListenPort: intPointer(80)},
					{Protocol: "tcp", ContainerPort: 443, Mode: "host", BindAddress: stringPointer("0.0.0.0"), ListenPort: intPointer(443)},
					{Protocol: "http", ContainerPort: 8080, Mode: "local", BindAddress: stringPointer("127.0.0.1"), ListenPort: intPointer(8080)},
				},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var document struct {
		Networks map[string]struct {
			Name   string `yaml:"name"`
			Driver string `yaml:"driver"`
		} `yaml:"networks"`
		Services map[string]struct {
			Networks []string `yaml:"networks"`
			Ports    []string `yaml:"ports"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal([]byte(compose), &document); err != nil {
		t.Fatal(err)
	}

	network, ok := document.Networks[consumerPlatformNetworkKey]
	if !ok || network.Name != "traefik" || network.Driver != "bridge" {
		t.Fatalf("gateway network = %#v", document.Networks)
	}
	if got := document.Services["traefik"].Networks; len(got) != 1 || got[0] != gatewayNetworkKey {
		t.Fatalf("gateway service networks = %#v", got)
	}
	ports := strings.Join(document.Services["traefik"].Ports, "\n")
	for _, want := range []string{"0.0.0.0:80:80", "0.0.0.0:443:443", "127.0.0.1:8080:8080"} {
		if !strings.Contains(ports, want) {
			t.Fatalf("gateway ports = %#v, missing %q", document.Services["traefik"].Ports, want)
		}
	}
}

func stringPointer(value string) *string { return &value }
func intPointer(value int) *int          { return &value }
