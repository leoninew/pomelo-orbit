package deploymentsvc

import (
	"path/filepath"
	"testing"
)

func TestResolveMountSpecsUsesHostPathFileSource(t *testing.T) {
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType:       mountSourceFile,
		Source:           "/var/run/docker.sock",
		SourceIsHostPath: true,
		Target:           "/var/run/docker.sock",
		ReadOnly:         true,
	}}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved mount count = %d, want 1", len(resolved))
	}
	if resolved[0].Source != "/var/run/docker.sock" || resolved[0].Target != "/var/run/docker.sock" || !resolved[0].ReadOnly {
		t.Fatalf("resolved mount = %+v", resolved[0])
	}
	if resolved[0].ShouldMaterialize {
		t.Fatal("host path mount must not be materialized")
	}
}

func TestResolveMountSpecsRejectsUnsupportedMountType(t *testing.T) {
	_, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: "unsupported",
		Source:     "source",
		Target:     "/var/run/docker.sock",
		ReadOnly:   true,
	}}, "", "")
	if err == nil {
		t.Fatal("unsupported mount type must be rejected")
	}
}

func TestResolveMountSpecsPreservesMountTarget(t *testing.T) {
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceNamedVolume,
		Source:     "app-data",
		Target:     "relative-target",
	}}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Source != "app-data" || resolved[0].Target != "relative-target" {
		t.Fatalf("resolved mount = %+v", resolved[0])
	}
	if resolved[0].NamedVolumeName != "app-data" {
		t.Fatalf("named volume = %q", resolved[0].NamedVolumeName)
	}
}

func TestResolveMountSpecsDoesNotMaterializePlainFile(t *testing.T) {
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceFile,
		Source:     "./app.conf",
		Target:     "/etc/app.conf",
	}}, "/srv/orbit/demo-service", "/srv/orbit/demo-service")
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].ShouldMaterialize {
		t.Fatal("plain file mount must not be materialized")
	}
}

func TestResolveMountSpecsRejectsBarePathForNonVolumeMounts(t *testing.T) {
	tests := []MountSpec{
		{SourceType: mountSourceDirectory, Source: "data", Target: "/var/lib/app"},
		{SourceType: mountSourceFile, Source: "app.conf", Target: "/etc/app.conf"},
		{SourceType: mountSourceControlledFile, Source: "config/app.env", Target: "/app/.env", Content: "", Mode: "0644"},
	}
	for _, mount := range tests {
		if err := validateMountSpec(mount); err == nil {
			t.Fatalf("bare source %q for %s mount must be rejected", mount.Source, mount.SourceType)
		}
	}
}

func TestResolveMountSpecsRejectsBackslashMountSources(t *testing.T) {
	tests := []MountSpec{
		{SourceType: mountSourceDirectory, Source: `D:\data`, Target: "/var/lib/app"},
		{SourceType: mountSourceFile, Source: `\\server\share\app.conf`, Target: "/etc/app.conf"},
	}
	for _, mount := range tests {
		if err := validateMountSpec(mount); err == nil {
			t.Fatalf("backslash source %q for %s mount must be rejected", mount.Source, mount.SourceType)
		}
	}
}

func TestResolveMountSpecsResolvesLogicalDirectoryAtServiceRoot(t *testing.T) {
	logicalServiceDir := filepath.Join(t.TempDir(), "deployment", "sc")
	composeMountSourceDir := filepath.Join(t.TempDir(), "host", "deployment", "sc")
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceDirectory,
		Source:     "./mysql",
		Target:     "/var/lib/mysql",
	}}, logicalServiceDir, composeMountSourceDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved mount count = %d, want 1", len(resolved))
	}
	if !resolved[0].ShouldMaterialize || resolved[0].IsFile {
		t.Fatalf("directory mount materialization = %+v", resolved[0])
	}
	if resolved[0].HostSource != filepath.ToSlash(filepath.Join(composeMountSourceDir, "mysql")) {
		t.Fatalf("compose host source = %q", resolved[0].HostSource)
	}
	if resolved[0].LogicalSource != filepath.Join(logicalServiceDir, "mysql") {
		t.Fatalf("logical source = %q", resolved[0].LogicalSource)
	}
}

func TestResolveMountSpecsPreservesDirectorySourceForNativeCompose(t *testing.T) {
	logicalServiceDir := filepath.Join(t.TempDir(), "deployment", "mysql-default")
	for _, test := range []struct {
		source string
		want   string
	}{
		{source: "./data", want: "./data"},
	} {
		t.Run(test.source, func(t *testing.T) {
			resolved, err := resolveMountSpecsForPaths([]MountSpec{{
				SourceType: mountSourceDirectory,
				Source:     test.source,
				Target:     "/var/lib/mysql",
			}}, logicalServiceDir, "")
			if err != nil {
				t.Fatal(err)
			}
			if resolved[0].Source != test.want || resolved[0].Target != "/var/lib/mysql" || resolved[0].HostSource != "" {
				t.Fatalf("native compose mount = %+v", resolved[0])
			}
			if resolved[0].LogicalSource != filepath.Join(logicalServiceDir, "data") || !resolved[0].ShouldMaterialize {
				t.Fatalf("native logical mount = %+v", resolved[0])
			}
		})
	}
}

func TestResolveMountSpecsPreservesAbsoluteControlledFileSource(t *testing.T) {
	source := filepath.ToSlash(filepath.Join(t.TempDir(), "etc", "app.env"))
	logicalServiceDir := filepath.Join(t.TempDir(), "deployment", "service")
	composeMountSourceDir := filepath.Join(t.TempDir(), "host", "deployment", "service")
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceControlledFile,
		Source:     source,
		Target:     "/app/.env",
		Content:    "KEY=value\n",
		Mode:       "0644",
	}}, logicalServiceDir, composeMountSourceDir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Source != source || resolved[0].Target != "/app/.env" {
		t.Fatalf("absolute mount = %+v, want source %q and target /app/.env", resolved[0], source)
	}
	if resolved[0].HostSource != source || resolved[0].LogicalSource != filepath.FromSlash(source) {
		t.Fatalf("absolute controlled file paths = %+v", resolved[0])
	}
	if !resolved[0].ShouldMaterialize || !resolved[0].IsFile {
		t.Fatalf("absolute controlled file materialization = %+v", resolved[0])
	}
}

func TestResolveMountSpecsKeepsControlledFileAsNativeBindMount(t *testing.T) {
	logicalServiceDir := filepath.Join(t.TempDir(), "deployment", "traefik-default")
	resolved, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceControlledFile,
		Source:     "./traefik.yml",
		Target:     "/etc/traefik/traefik.yml",
		Content:    "api:\n  dashboard: true\n",
		Mode:       "0644",
	}}, logicalServiceDir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved mount count = %d, want 1", len(resolved))
	}
	if resolved[0].Source != "./traefik.yml" || resolved[0].Target != "/etc/traefik/traefik.yml" {
		t.Fatalf("compose mount = %q, want native bind mount", resolved[0].Source)
	}
	if resolved[0].LogicalSource != filepath.Join(logicalServiceDir, "traefik.yml") || !resolved[0].ShouldMaterialize {
		t.Fatalf("native logical mount = %+v", resolved[0])
	}
}

func TestResolveMountSpecsRejectsInvalidControlledFileMode(t *testing.T) {
	_, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceControlledFile,
		Source:     "./config/app.conf",
		Target:     "/etc/app.conf",
		Content:    "key=value",
		Mode:       "644",
	}}, "/srv/orbit/demo-service", "/srv/orbit/demo-service")
	if err == nil {
		t.Fatal("controlled file without a four-digit Unix mode must be rejected")
	}
}

func TestResolveMountSpecsRejectsLogicalSourceWithoutComposePrefix(t *testing.T) {
	_, err := resolveMountSpecsForPaths([]MountSpec{{
		SourceType: mountSourceControlledFile,
		Source:     "traefik.yml",
		Target:     "/etc/traefik/traefik.yml",
		Content:    "api:\n  dashboard: true\n",
		Mode:       "0644",
	}}, "/logical/service", "/physical/service")
	if err == nil {
		t.Fatal("controlled file source without ./ must be rejected")
	}
}

func TestCountLogicalMountsIncludesControlledFiles(t *testing.T) {
	items := []ResolvedMount{
		{SourceType: mountSourceDirectory, ShouldMaterialize: true},
		{SourceType: mountSourceFile, ShouldMaterialize: false},
		{SourceType: mountSourceControlledFile, ShouldMaterialize: true},
		{SourceType: mountSourceNamedVolume},
	}

	if got, want := countLogicalMounts(items), 2; got != want {
		t.Fatalf("countLogicalMounts() = %d, want %d", got, want)
	}
}
