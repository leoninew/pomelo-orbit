package environmenthandler

import (
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestEnvironmentResponseIncludesOnlyApplicableTargetFields(t *testing.T) {
	local := environmentResponse(environmentdto.View{
		Id: "local", TargetType: model.EnvironmentTargetTypeLocal,
		Local: &environmentdto.LocalTargetView{WorkspaceRoot: "/srv/orbit/deployment", Platform: "linux", Host: "orbit-host", Username: "orbit"},
	})
	if local.Local == nil || local.Local.WorkspaceRoot != "/srv/orbit/deployment" || local.Local.Platform != "linux" || local.Local.Host != "orbit-host" || local.Local.Username != "orbit" || local.Ssh != nil {
		t.Fatalf("local response = %#v", local)
	}

	ssh := environmentResponse(environmentdto.View{
		Id: "ssh", TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &environmentdto.SSHTargetView{Platform: model.EnvironmentPlatformLinux, Host: "127.0.0.1", Port: 22, Username: "orbit", WorkspaceRoot: "/srv/orbit"},
	})
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
