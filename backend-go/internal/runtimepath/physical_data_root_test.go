package runtimepath

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolvePhysicalDataRootResolvesRelativeRootToAbsolutePath(t *testing.T) {
	got, err := ResolvePhysicalDataRoot(context.Background(), "data")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute physical data root, got %q", got)
	}
	if filepath.Base(got) != "data" {
		t.Fatalf("expected data root suffix, got %q", got)
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
