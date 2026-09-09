package gatewaysvc

import (
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/config"
	"gopkg.in/yaml.v3"
)

func testGatewayConfig() config.TraefikConfig {
	return config.TraefikConfig{
		Image: "traefik:3.6", RestApiUrl: "http://localhost:8080", BaseDomain: "example.test", RestReadyTimeout: 20 * time.Second,
	}
}

func TestBuildInitialGatewayVersionsDeclareStaticTopology(t *testing.T) {
	versions := buildInitialGatewayVersions("gateway-1", "traefik:3.6", "missing", "traefik", "traefik")
	if len(versions) != 4 {
		t.Fatalf("initial versions = %#v", versions)
	}
	roles := map[string]bool{}
	for _, version := range versions {
		roles[version.Role] = true
		component := version.Component
		if component.VersionId != version.Version.Id || component.Name != "traefik" || component.Image != "traefik:3.6" || component.PullPolicy != "missing" {
			t.Fatalf("initial component = %#v", component)
		}
		if len(component.Mounts) != 4 || len(component.Endpoints) != 3 {
			t.Fatalf("initial component topology = %#v", component)
		}
	}
	for _, role := range []string{"base", "http", "dns", "http-dns"} {
		if !roles[role] {
			t.Fatalf("missing initial Gateway Version role %q", role)
		}
	}
}

func TestInitialGatewayProfileStaticConfigsAreValidYAML(t *testing.T) {
	for _, role := range []string{gatewayVersionProfileHTTP, gatewayVersionProfileDNS, gatewayVersionProfileBoth} {
		t.Run(role, func(t *testing.T) {
			var document struct {
				Providers struct {
					Docker struct {
						Network string `yaml:"network"`
					} `yaml:"docker"`
				} `yaml:"providers"`
				CertificatesResolvers map[string]struct {
					ACME struct {
						DNSChallenge struct {
							Provider    string   `yaml:"provider"`
							Resolvers   []string `yaml:"resolvers"`
							Propagation struct {
								DelayBeforeChecks string `yaml:"delayBeforeChecks"`
							} `yaml:"propagation"`
						} `yaml:"dnsChallenge"`
					} `yaml:"acme"`
				} `yaml:"certificatesResolvers"`
			}
			if err := yaml.Unmarshal([]byte(initialTraefikStaticConfig(role, "traefik")), &document); err != nil {
				t.Fatalf("parse static config: %v", err)
			}
			if document.Providers.Docker.Network != "traefik" {
				t.Fatalf("Docker provider network = %q", document.Providers.Docker.Network)
			}
			if role == gatewayVersionProfileHTTP {
				return
			}
			dnsResolver := document.CertificatesResolvers["letsencrypt-dns"].ACME.DNSChallenge
			if dnsResolver.Provider != "cloudflare" || len(dnsResolver.Resolvers) != 2 || dnsResolver.Resolvers[0] != "1.1.1.1:53" || dnsResolver.Resolvers[1] != "8.8.8.8:53" || dnsResolver.Propagation.DelayBeforeChecks != "60s" {
				t.Fatalf("DNS resolver configuration = %#v", dnsResolver)
			}
		})
	}
}
