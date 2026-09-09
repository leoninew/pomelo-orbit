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
