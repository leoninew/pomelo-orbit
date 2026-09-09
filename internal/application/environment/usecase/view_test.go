package environmentsvc

import (
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestToViewUsesInjectedLocalDisplayWithoutWorkspaceYaml(t *testing.T) {
	item := model.Environment{
		Id: "environment-local", ProjectId: "project-1", Code: "project",
		State: model.EnvironmentStateActive, TargetType: model.EnvironmentTargetTypeLocal,
	}
	view := New(nil, nil, nil, nil, nil, nil).WithLocalDisplay(environmentdto.LocalDisplaySnapshot{
		Platform: "linux", Host: "orbit-host", Username: "orbit",
	}).toView(item)
	if view.Local == nil || view.Local.WorkspaceRoot != "" || view.Local.Platform != "linux" || view.Local.Host != "orbit-host" || view.Local.Username != "orbit" || view.SSH != nil {
		t.Fatalf("local view = %#v", view)
	}
}

func TestToViewLeavesLocalDisplayEmptyWithoutSnapshot(t *testing.T) {
	item := model.Environment{Id: "environment-local", TargetType: model.EnvironmentTargetTypeLocal}
	view := New(nil, nil, nil, nil, nil, nil).toView(item)
	if view.Local == nil || view.Local.Platform != "" || view.Local.Host != "" || view.Local.Username != "" || view.Local.WorkspaceRoot != "" {
		t.Fatalf("undecorated local view = %#v", view)
	}
}

func TestToViewOmitsSSHCredential(t *testing.T) {
	item := testProbeEnvironment("project-1")
	view := New(nil, nil, nil, nil, nil, nil).toView(item)
	if view.SSH == nil || view.SSH.Host != item.SSH.Host || view.SSH.WorkspaceRoot != item.SSH.WorkspaceRoot || view.Local != nil {
		t.Fatalf("ssh view = %#v", view)
	}
}
