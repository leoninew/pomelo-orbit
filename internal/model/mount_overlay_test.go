package model

import (
	"strings"
	"testing"
)

func TestMergeServiceComponentMountPreservesIndependentExplicitValues(t *testing.T) {
	declaration := VersionComponentMount{Target: "/config", SourceType: "controlled_file", Source: "./config", Content: "version", Mode: "0644", IgnoreIfExists: true}
	empty := ""
	zeroMode := "0000"
	disabled := false
	overlay := ServiceComponentMount{Target: declaration.Target, State: ServiceComponentOverlayOverride, Content: &empty, Mode: &zeroMode, IgnoreIfExists: &disabled}
	merged, err := MergeServiceComponentMount(declaration, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Content != "" || merged.Mode != "0000" || merged.IgnoreIfExists || merged.ReadOnly || merged.Source != declaration.Source {
		t.Fatalf("merged mount = %#v", merged)
	}
	overlay.Content = nil
	declaration.Content = "new version"
	merged, err = MergeServiceComponentMount(declaration, overlay)
	if err != nil || merged.Content != "new version" || merged.Mode != "0000" {
		t.Fatalf("inherited content = %#v, error = %v", merged, err)
	}
}

func TestMergeServiceComponentMountRejectsInvalidOverlays(t *testing.T) {
	declaration := VersionComponentMount{Target: "/config", SourceType: "controlled_file", Source: "./config", Content: "version", Mode: "0644"}
	content := "value"
	invalidMode := "0778"
	oversized := strings.Repeat("x", 256*1024+1)
	readOnly := false
	for _, overlay := range []ServiceComponentMount{
		{Target: declaration.Target, State: ServiceComponentOverlayOverride},
		{Target: declaration.Target, State: ServiceComponentOverlayDeleted, Content: &content},
		{Target: declaration.Target, State: ServiceComponentOverlayOverride, Mode: &invalidMode},
		{Target: declaration.Target, State: ServiceComponentOverlayOverride, Content: &oversized},
		{Target: "/other", State: ServiceComponentOverlayOverride, Content: &content},
		{Target: declaration.Target, State: ServiceComponentOverlayOverride, ReadOnly: &readOnly},
	} {
		if _, err := MergeServiceComponentMount(declaration, overlay); err == nil {
			t.Fatalf("accepted invalid overlay %#v", overlay)
		}
	}
	directory := VersionComponentMount{Target: "/dir", SourceType: "directory", Source: "./dir"}
	if _, err := MergeServiceComponentMount(directory, ServiceComponentMount{Target: directory.Target, State: ServiceComponentOverlayOverride, Content: &content}); err == nil {
		t.Fatal("accepted file content for a directory")
	}
	merged, err := MergeServiceComponentMount(directory, ServiceComponentMount{Target: directory.Target, State: ServiceComponentOverlayOverride, ReadOnly: &readOnly})
	if err != nil || merged.ReadOnly {
		t.Fatalf("directory read-only override = %#v, error = %v", merged, err)
	}
}
