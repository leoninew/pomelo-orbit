package deploymentsvc

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveMountSpecsUsesHostPathFileSource(t *testing.T) {
	resolved, err := resolveMountSpecs([]MountSpec{{
		SourceType:       mountSourceFile,
		Source:           "/var/run/docker.sock",
		SourceIsHostPath: true,
		Target:           "/var/run/docker.sock",
		ReadOnly:         true,
	}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved mount count = %d, want 1", len(resolved))
	}
	if resolved[0].Compose != "/var/run/docker.sock:/var/run/docker.sock:ro" {
		t.Fatalf("compose mount = %q", resolved[0].Compose)
	}
	if resolved[0].ShouldMaterialize {
		t.Fatal("host path mount must not be materialized")
	}
}

func TestResolveMountSpecsRejectsUnsupportedMountType(t *testing.T) {
	_, err := resolveMountSpecs([]MountSpec{{
		SourceType: "unsupported",
		Source:     "source",
		Target:     "/var/run/docker.sock",
		ReadOnly:   true,
	}}, "")
	if err == nil {
		t.Fatal("unsupported mount type must be rejected")
	}
}

func TestResolveMountSpecsPreservesMountTarget(t *testing.T) {
	resolved, err := resolveMountSpecs([]MountSpec{{
		SourceType: mountSourceNamedVolume,
		Source:     "app-data",
		Target:     "relative-target",
	}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].Compose != "app-data:relative-target" {
		t.Fatalf("compose mount = %q", resolved[0].Compose)
	}
	if resolved[0].NamedVolumeName != "app-data" {
		t.Fatalf("named volume = %q", resolved[0].NamedVolumeName)
	}
}

func TestResolveMountSpecsDoesNotMaterializePlainFile(t *testing.T) {
	resolved, err := resolveMountSpecs([]MountSpec{{
		SourceType: mountSourceFile,
		Source:     "app.conf",
		Target:     "/etc/app.conf",
	}}, "/srv/orbit/demo-service")
	if err != nil {
		t.Fatal(err)
	}
	if resolved[0].ShouldMaterialize {
		t.Fatal("plain file mount must not be materialized")
	}
}

func TestResolveMountSpecsMaterializesLogicalDirectoryAtServiceRoot(t *testing.T) {
	physicalServiceDir := filepath.Join(t.TempDir(), "deployment", "sc")
	resolved, err := resolveMountSpecs([]MountSpec{{
		SourceType: mountSourceDirectory,
		Source:     "mysql",
		Target:     "/var/lib/mysql",
	}}, physicalServiceDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved mount count = %d, want 1", len(resolved))
	}
	if !resolved[0].ShouldMaterialize || resolved[0].IsFile {
		t.Fatalf("directory mount materialization = %+v", resolved[0])
	}
	if err := MaterializeLogicalMountSources(resolved); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(physicalServiceDir, "mysql")); err != nil || !info.IsDir() {
		t.Fatalf("mysql mount source was not created: info=%v err=%v", info, err)
	}
}

func TestResolveMountSpecsRejectsInvalidControlledFileMode(t *testing.T) {
	_, err := resolveMountSpecs([]MountSpec{{
		SourceType: mountSourceControlledFile,
		Source:     "config/app.conf",
		Target:     "/etc/app.conf",
		Content:    "key=value",
		Mode:       "644",
	}}, "/srv/orbit/demo-service")
	if err == nil {
		t.Fatal("controlled file without a four-digit Unix mode must be rejected")
	}
}

func TestMaterializeControlledFileRespectsIgnoreIfExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "acme.json")
	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MaterializeLogicalMountSources([]ResolvedMount{{
		HostSource:        path,
		IsFile:            true,
		ShouldMaterialize: true,
		SourceType:        mountSourceControlledFile,
		Content:           "{}",
		IgnoreIfExists:    true,
		FileMode:          0o600,
	}}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "existing" {
		t.Fatalf("preserved content = %q", content)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("file mode = %04o, want 0600", got)
		}
	}
}

func TestCountLogicalMountsIncludesControlledFiles(t *testing.T) {
	items := []ResolvedMount{
		{SourceType: mountSourceDirectory},
		{SourceType: mountSourceFile},
		{SourceType: mountSourceControlledFile},
		{SourceType: mountSourceNamedVolume},
	}

	if got, want := countLogicalMounts(items), 3; got != want {
		t.Fatalf("countLogicalMounts() = %d, want %d", got, want)
	}
}
