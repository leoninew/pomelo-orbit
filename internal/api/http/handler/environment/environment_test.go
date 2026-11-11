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
		TargetMayBeShared: true,
		Local:             &environmentdto.LocalTargetView{WorkspaceRoot: "/srv/orbit/deployment", Platform: "linux", Host: "orbit-host", Username: "orbit"},
	})
	if local.Local == nil || local.Local.WorkspaceRoot != "/srv/orbit/deployment" || local.Local.Platform != "linux" || local.Local.Host != "orbit-host" || local.Local.Username != "orbit" || local.Ssh != nil {
		t.Fatalf("local response = %#v", local)
	}
	if !local.TargetMayBeShared {
		t.Fatal("local response omitted the target warning")
	}

	ssh := environmentResponse(environmentdto.View{
		Id: "ssh", TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &environmentdto.SSHTargetView{Platform: model.EnvironmentPlatformLinux, Host: "127.0.0.1", Port: 22, Username: "orbit", WorkspaceRoot: "/srv/orbit", HostIdentityPolicy: model.EnvironmentHostIdentityPolicyStable},
	})
	if ssh.Ssh == nil || ssh.Ssh.Host != "127.0.0.1" || ssh.Local != nil || ssh.Ssh.HostIdentityPolicy != model.EnvironmentHostIdentityPolicyStable {
		t.Fatalf("SSH response = %#v", ssh)
	}
}

func TestProjectEnvironmentUpdateInputPreservesPolicyPresence(t *testing.T) {
	stable, empty := model.EnvironmentHostIdentityPolicyStable, ""
	for _, policy := range []*string{nil, &stable, &empty} {
		input := projectEnvironmentUpdateInput(&environmentv1.ProjectEnvironmentUpdateReq{
			Ssh: &environmentv1.EnvironmentSSHTargetReq{HostIdentityPolicy: policy},
		})
		if input.SSH == nil || input.SSH.HostIdentityPolicy != policy {
			t.Fatalf("policy presence lost: %+v", input)
		}
	}
}

func TestProjectEnvironmentUpdateInputMapsLocalWorkspace(t *testing.T) {
	input := projectEnvironmentUpdateInput(&environmentv1.ProjectEnvironmentUpdateReq{
		Local: &environmentv1.EnvironmentLocalTargetReq{WorkspaceRoot: "/srv/orbit/deployment"},
	})
	if input.Local == nil || input.Local.WorkspaceRoot != "/srv/orbit/deployment" || input.SSH != nil {
		t.Fatalf("update input = %#v", input)
	}
}
