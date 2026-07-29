package gatewaysvc

import (
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
	for _, mount := range mounts {
		if err := validateMountSpec(mount); err != nil {
			t.Fatalf("managed mount must pass schema: %v", err)
		}
	}
	if len(mounts) != 3 {
		t.Fatalf("managed mount count = %d, want 3", len(mounts))
	}
	if mounts[0].SourceType != mountSourceFile || !mounts[0].SourceIsHostPath || mounts[0].Source != gatewayDockerSocketPath || !mounts[0].ReadOnly {
		t.Fatalf("docker socket mount = %+v", mounts[0])
	}
	if mounts[1].SourceType != mountSourceControlledFile || mounts[1].Mode != "0644" || mounts[1].IgnoreIfExists || !strings.Contains(mounts[1].Content, "providers:") {
		t.Fatalf("traefik config mount = %+v", mounts[1])
	}
	if mounts[2].SourceType != mountSourceControlledFile || mounts[2].Mode != "0600" || !mounts[2].IgnoreIfExists {
		t.Fatalf("acme mount = %+v", mounts[2])
	}
}

func TestBuildManagedGatewayComponentPreservesCustomMounts(t *testing.T) {
	image := "traefik:3.6"
	component, err := buildManagedGatewayComponent(model.GatewayConfig{Image: &image}, []model.VersionComponent{{
		Name: gatewayManagedComponentName,
		Mounts: []model.VersionComponentMount{{
			SourceType: mountSourceControlledFile, Source: "custom.conf", Target: "/etc/custom.conf", Content: "x", Mode: "0644",
		}},
	}}, []int{6379})
	if err != nil {
		t.Fatal(err)
	}
	if len(component.Ports) != 4 || component.Ports[3].HostPort != 6379 || component.Ports[3].ContainerPort != 6379 {
		t.Fatalf("managed ports = %v", component.Ports)
	}
	if len(component.Mounts) != 4 || component.Mounts[0].Target != "/etc/custom.conf" {
		t.Fatalf("custom mount was not retained: %v", component.Mounts)
	}
	if _, err := buildManagedGatewayComponent(model.GatewayConfig{}, nil, nil); err == nil {
		t.Fatal("expected missing image error")
	}
}
