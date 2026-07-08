package cisvc

import (
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
)

func TestDockerRunArgsUseBindMountSyntax(t *testing.T) {
	hostPath := filepath.Join(t.TempDir(), "workspace")
	args, err := dockerRunArgs(RunOptions{
		Image:  "alpine",
		Script: "echo ok",
		Volumes: []ciworkspace.VolumeMount{
			{HostPath: hostPath, ContainerPath: "/workspace", Mode: "rw"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, " -v ") {
		t.Fatalf("expected --mount syntax instead of -v: %s", joined)
	}
	want := "type=bind,source=" + hostPath + ",target=/workspace"
	if !containsArg(args, "--mount", want) {
		t.Fatalf("expected bind mount %q in args %+v", want, args)
	}
}

func TestDockerRunArgsRejectRelativeHostPath(t *testing.T) {
	_, err := dockerRunArgs(RunOptions{
		Image:  "alpine",
		Script: "echo ok",
		Volumes: []ciworkspace.VolumeMount{
			{HostPath: filepath.Join("data", "ci", "repo", "workspace"), ContainerPath: "/workspace", Mode: "rw"},
		},
	})
	if err == nil {
		t.Fatal("expected relative Docker host path to be rejected")
	}
}

func containsArg(args []string, flag string, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
