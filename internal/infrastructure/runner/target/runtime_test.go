package targetrunner

import (
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRuntimeDispatchesOnlyByExplicitTargetType(t *testing.T) {
	local := &runtimeFake{serviceDir: "local"}
	ssh := &runtimeFake{serviceDir: "ssh"}
	runtime := New(local, ssh)

	localDir, err := runtime.ServiceDir(environmentport.Target{Environment: model.Environment{TargetType: model.EnvironmentTargetTypeLocal}}, "service")
	if err != nil {
		t.Fatal(err)
	}
	if localDir != "local" || local.calls != 1 || ssh.calls != 0 {
		t.Fatalf("local dispatch: dir=%q local=%d ssh=%d", localDir, local.calls, ssh.calls)
	}

	sshDir, err := runtime.ServiceDir(environmentport.Target{Environment: model.Environment{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH:        &model.EnvironmentSSHTarget{Host: "127.0.0.1"},
	}}, "service")
	if err != nil {
		t.Fatal(err)
	}
	if sshDir != "ssh" || local.calls != 1 || ssh.calls != 1 {
		t.Fatalf("SSH loopback dispatch: dir=%q local=%d ssh=%d", sshDir, local.calls, ssh.calls)
	}
}

type runtimeFake struct {
	deploymentport.Runtime
	serviceDir string
	calls      int
}

func (r *runtimeFake) ServiceDir(environmentport.Target, string) (string, error) {
	r.calls++
	return r.serviceDir, nil
}
