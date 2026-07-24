package gatewaysvc

import (
	"encoding/json"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildTraefikStaticConfigIncludesProvidersAndTCP(t *testing.T) {
	config := buildTraefikStaticConfig(model.GatewayConfig{TLSMode: "letsencrypt"}, []int{6379, 3306})
	for _, want := range []string{
		"providers:", "rest:", "insecure: true", "docker:", "unix:///var/run/docker.sock",
		"network: traefik", "entryPoints:", "tcp6379:", `address: ":6379"`,
		"certificatesResolvers:", "letsencrypt:", "httpChallenge:",
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("expected %q in static config:\n%s", want, config)
		}
	}
}

func TestBuildManagedGatewayMountsRemainValid(t *testing.T) {
	mounts := buildManagedGatewayMounts(model.GatewayConfig{}, nil)
	raw, err := json.Marshal(mounts)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(raw)
	parsed, err := parseMountSpecs(&serialized)
	if err != nil {
		t.Fatalf("managed mounts must pass schema: %v", err)
	}
	if len(parsed) != 3 {
		t.Fatalf("managed mount count = %d, want 3", len(parsed))
	}
	if parsed[0].SourceType != mountSourceSpecial || parsed[0].Source != specialDockerSock || !parsed[0].ReadOnly {
		t.Fatalf("docker socket mount = %+v", parsed[0])
	}
	if parsed[1].ContentMode != contentModeSync || !strings.Contains(parsed[1].Content, "providers:") {
		t.Fatalf("traefik config mount = %+v", parsed[1])
	}
	if parsed[2].ContentMode != contentModeSeed {
		t.Fatalf("acme mount = %+v", parsed[2])
	}
}

func TestBuildManagedGatewayComponentPreservesCustomMounts(t *testing.T) {
	image := "traefik:3.6"
	existingMounts := `[{
  "source_type":"logical",
  "source":"custom.conf",
  "target":"/etc/custom.conf",
  "content":"x",
  "content_mode":"seed"
}]`
	component, err := buildManagedGatewayComponent(model.GatewayConfig{Image: &image}, []model.VersionComponent{{
		Name: gatewayManagedComponentName, MountsJSON: &existingMounts,
	}}, []int{6379})
	if err != nil {
		t.Fatal(err)
	}
	if component.PortsJSON == nil || !strings.Contains(*component.PortsJSON, "6379:6379") {
		t.Fatalf("managed ports = %v", component.PortsJSON)
	}
	if component.MountsJSON == nil || !strings.Contains(*component.MountsJSON, "/etc/custom.conf") {
		t.Fatalf("custom mount was not retained: %v", component.MountsJSON)
	}
	if _, err := buildManagedGatewayComponent(model.GatewayConfig{}, nil, nil); err == nil {
		t.Fatal("expected missing image error")
	}
}
