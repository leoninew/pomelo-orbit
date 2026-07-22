package cdsvc

import (
	"encoding/json"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestBuildTraefikStaticConfigEnablesRestAndDocker(t *testing.T) {
	yml := buildTraefikStaticConfig()
	for _, want := range []string{
		"providers:",
		"rest:",
		"insecure: true",
		"docker:",
		"unix:///var/run/docker.sock",
		"network: traefik",
		"entryPoints:",
		"web:",
		"websecure:",
		"api:",
		"dashboard: true",
	} {
		if !strings.Contains(yml, want) {
			t.Fatalf("expected %q in static config:\n%s", want, yml)
		}
	}
}

func TestBuildManagedGatewayMountsValid(t *testing.T) {
	mounts := buildManagedGatewayMounts()
	if len(mounts) != 3 {
		t.Fatalf("expected 3 managed mounts, got %d", len(mounts))
	}
	raw, err := json.Marshal(mounts)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	parsed, err := parseMountSpecs(&s)
	if err != nil {
		t.Fatalf("managed mounts must pass schema: %v\n%s", err, s)
	}
	if len(parsed) != 3 {
		t.Fatalf("parsed %d mounts", len(parsed))
	}
	if parsed[0].SourceType != mountSourceSpecial || parsed[0].Source != specialDockerSock {
		t.Fatalf("first mount should be docker.sock special, got %+v", parsed[0])
	}
	if parsed[1].ContentMode != contentModeSync || !strings.Contains(parsed[1].Content, "providers:") {
		t.Fatalf("traefik.yml should be sync with content, got %+v", parsed[1])
	}
	if parsed[2].ContentMode != contentModeSeed {
		t.Fatalf("acme.json should be seed, got %+v", parsed[2])
	}
}

func TestMergeManagedGatewayMountsPreservesCustomTargets(t *testing.T) {
	existing := []MountSpec{
		{SourceType: mountSourceLogical, Source: "custom.conf", Target: "/etc/custom.conf", Content: "x", ContentMode: contentModeSeed},
		{SourceType: mountSourceSpecial, Source: specialDockerSock, Target: gatewayMountTargetDockerSock, ReadOnly: false},
	}
	managed := buildManagedGatewayMounts()
	got := mergeManagedGatewayMounts(existing, managed)
	targets := make(map[string]MountSpec, len(got))
	for _, m := range got {
		targets[m.Target] = m
	}
	if _, ok := targets["/etc/custom.conf"]; !ok {
		t.Fatalf("custom mount lost: %+v", got)
	}
	if sock, ok := targets[gatewayMountTargetDockerSock]; !ok || !sock.ReadOnly {
		t.Fatalf("docker.sock should be rewritten managed, got %+v", sock)
	}
	if yml, ok := targets[gatewayMountTargetTraefikYml]; !ok || yml.ContentMode != contentModeSync {
		t.Fatalf("traefik.yml managed missing: %+v", yml)
	}
}

func TestBuildManagedGatewayComponentUsesImageAndDefault(t *testing.T) {
	custom := "traefik:v3.9"
	c, err := buildManagedGatewayComponent(model.GatewayConfig{Image: &custom}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != gatewayManagedComponentName || c.Image != custom {
		t.Fatalf("got name=%s image=%s", c.Name, c.Image)
	}
	if c.PortsJSON == nil || !strings.Contains(*c.PortsJSON, "80:80") {
		t.Fatalf("expected ports, got %v", c.PortsJSON)
	}

	c2, err := buildManagedGatewayComponent(model.GatewayConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c2.Image != defaultGatewayImage {
		t.Fatalf("default image want %s got %s", defaultGatewayImage, c2.Image)
	}
}

func TestUpsertComponentByNamePreservesOthers(t *testing.T) {
	existing := []model.VersionComponent{
		{Id: "1", Name: "sidecar", Image: "busybox"},
		{Id: "2", Name: gatewayManagedComponentName, Image: "old"},
	}
	managed := model.VersionComponent{Name: gatewayManagedComponentName, Image: "new"}
	got := upsertComponentByName(existing, managed)
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].Name != "sidecar" || got[0].Image != "busybox" {
		t.Fatalf("sidecar lost: %+v", got[0])
	}
	if got[1].Name != gatewayManagedComponentName || got[1].Image != "new" || got[1].Id != "2" {
		t.Fatalf("managed not upserted: %+v", got[1])
	}
}
