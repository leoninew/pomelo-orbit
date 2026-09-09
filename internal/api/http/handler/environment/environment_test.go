package environmenthandler

import (
	"testing"

	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestEnvironmentResponseIncludesOnlyApplicableTargetFields(t *testing.T) {
	localTarget := localTargetInfo{WorkspaceRoot: "/srv/orbit/deployment", Platform: "linux", Host: "orbit-host", Username: "orbit"}
	local := environmentResponse(model.Environment{Id: "local", TargetType: model.EnvironmentTargetTypeLocal}, localTarget)
	if local.Local == nil || local.Local.WorkspaceRoot != "/srv/orbit/deployment" || local.Local.Platform != "linux" || local.Local.Host != "orbit-host" || local.Local.Username != "orbit" || local.Ssh != nil {
		t.Fatalf("local response = %#v", local)
	}

	ssh := environmentResponse(model.Environment{
		Id: "ssh", TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &model.EnvironmentSSHTarget{Platform: model.EnvironmentPlatformLinux, Host: "127.0.0.1", Port: 22, Username: "orbit", WorkspaceRoot: "/srv/orbit"},
	}, localTarget)
	if ssh.Ssh == nil || ssh.Ssh.Host != "127.0.0.1" || ssh.Local != nil {
		t.Fatalf("SSH response = %#v", ssh)
	}
}

func TestProjectEnvironmentInitializeInputKeepsTemporaryAuthenticationInRequestMapping(t *testing.T) {
	input := projectEnvironmentInitializeInput(&environmentv1.ProjectEnvironmentInitializeReq{
		Username:             "bootstrap-user",
		Password:             "password",
		PrivateKey:           "private-key",
		PrivateKeyPassphrase: "passphrase",
	})
	if input.Username != "bootstrap-user" || input.Password != "password" || input.PrivateKey != "private-key" || input.PrivateKeyPassphrase != "passphrase" {
		t.Fatalf("initialize input = %#v", input)
	}
}
