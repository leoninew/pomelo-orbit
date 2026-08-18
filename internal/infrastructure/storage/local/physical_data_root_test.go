package runtimepath

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveDockerDaemonPathLeavesNativePathUnchanged(t *testing.T) {
	got, err := resolveDockerDaemonPath(context.Background(), "workspace", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if got != "workspace" {
		t.Fatalf("native Docker daemon path = %q, want unchanged path", got)
	}
}

func TestResolveDockerDaemonPathRequiresAbsoluteOrbitPathInContainer(t *testing.T) {
	_, err := resolveDockerDaemonPath(context.Background(), "workspace", "container-id", true)
	if err == nil {
		t.Fatal("expected relative Orbit path to be rejected in a container")
	}
}

func TestResolveMountedContainerPathUsesLongestDestination(t *testing.T) {
	got, err := resolveMountedContainerPath("/app/data/deployment/traefik/data/certs", []dockerInspectMount{
		{Source: "/srv/orbit", Destination: "/app/data"},
		{Source: "/srv/orbit/cd", Destination: "/app/data/deployment"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/srv/orbit/cd", "traefik", "data", "certs")
	if got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
}

func TestResolveMountedContainerPathRejectsUnmappedPath(t *testing.T) {
	_, err := resolveMountedContainerPath("/tmp/orbit-ci", []dockerInspectMount{{Source: "/srv/orbit", Destination: "/work"}})
	if err == nil {
		t.Fatal("expected unmapped path to fail")
	}
}

func TestNormalizeContainerId(t *testing.T) {
	containerId := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	cases := []string{
		containerId,
		"docker-" + containerId + ".scope",
		" " + containerId + " ",
	}
	for _, value := range cases {
		if got := normalizeContainerId(value); got != containerId {
			t.Fatalf("normalizeContainerId(%q) = %q, want %q", value, got, containerId)
		}
	}
	if got := normalizeContainerId("not-a-container-id"); got != "" {
		t.Fatalf("expected invalid container id to normalize to empty string, got %q", got)
	}
}

func TestCleanContainerPathUsesSlashAndCleans(t *testing.T) {
	got := cleanContainerPath(`\\app\\data\\..\\data`)
	if got != `/app/data` {
		t.Fatalf("unexpected clean path: %q", got)
	}
}
