package deploymentsvc

import (
	"context"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestRenderGatewayComposeUsesStableTraefikNetworkKey(t *testing.T) {
	listenPort := 8080
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{
		Plan: model.EffectiveServicePlan{
			Application: model.Application{Code: "traefik", Kind: status.ApplicationKindGateway},
			Version:     model.Version{Id: "version-1"},
			Service:     model.Service{InstanceKey: "default"},
			Gateway: &model.GatewayConfig{
				BaseDomain:        "example.test",
				DefaultEntrypoint: "web",
			},
			Components: []model.EffectiveServiceComponent{{
				Name: "traefik", Image: "traefik:3.6",
				Endpoints: []model.VersionComponentEndpoint{{
					Protocol: "tcp", ContainerPort: 8080, Mode: "host", ListenPort: &listenPort,
				}},
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
	if !ok {
		t.Fatalf("gateway networks = %#v, want key %q", document.Networks, consumerPlatformNetworkKey)
	}
	if network.Name != defaultGatewayNetworkName || network.Driver != "bridge" {
		t.Fatalf("gateway network = %#v", network)
	}
	if got := document.Services["traefik"].Networks; len(got) != 1 || got[0] != consumerPlatformNetworkKey {
		t.Fatalf("gateway service networks = %#v", got)
	}
	if got := document.Services["traefik"].Ports; len(got) != 1 || got[0] != "0.0.0.0:8080:8080" {
		t.Fatalf("gateway api port mapping = %#v", got)
	}
}
