package environmentsvc

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestValidWorkspaceRootAllowsSSHHome(t *testing.T) {
	for _, platform := range []string{model.EnvironmentPlatformLinux, model.EnvironmentPlatformWindows} {
		if !validWorkspaceRoot(platform, "~/.pomelo-orbit") {
			t.Fatalf("%s workspace root must allow the SSH user's home directory", platform)
		}
	}
}

func TestValidWorkspaceRootAllowsLocalHomeAndAbsolutePath(t *testing.T) {
	if !validWorkspaceRoot(model.EnvironmentPlatformLinux, "~/.pomelo-orbit") {
		t.Fatal("local linux workspace root must allow the user home prefix")
	}
	if !validWorkspaceRoot(model.EnvironmentPlatformWindows, "~/.pomelo-orbit") {
		t.Fatal("local windows workspace root must allow the user home prefix")
	}
	if !validWorkspaceRoot(model.EnvironmentPlatformLinux, "/tmp/orbit-workspace") {
		t.Fatal("local linux workspace root must allow an absolute path")
	}
	if !validWorkspaceRoot(model.EnvironmentPlatformWindows, `C:\orbit-workspace`) {
		t.Fatal("local windows workspace root must allow a drive-absolute path")
	}
	if validWorkspaceRoot(model.EnvironmentPlatformLinux, ".pomelo-orbit") {
		t.Fatal("local workspace root must not allow a relative path without tilde")
	}
}
