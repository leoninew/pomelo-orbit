package environmentsvc

import (
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) toView(item model.Environment) environmentdto.View {
	view := environmentdto.View{
		Id: item.Id, ProjectId: item.ProjectId, Code: item.Code, State: item.State,
		TargetType: item.TargetType, TargetRevision: item.TargetRevision,
		LastProbeRevision: item.LastProbeRevision, LastProbeStatus: item.LastProbeStatus,
		LastProbeAt: item.LastProbeAt, LastProbeDiagnostic: item.LastProbeDiagnostic,
		GatewayApplicationId: item.GatewayApplicationId, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	if item.IsLocal() {
		view.Local = &environmentdto.LocalTargetView{
			WorkspaceRoot: item.WorkspaceRoot,
			Platform:      s.localDisplay.Platform, Host: s.localDisplay.Host, Username: s.localDisplay.Username,
		}
	}
	if item.SSH != nil {
		view.SSH = &environmentdto.SSHTargetView{
			Platform: item.SSH.Platform, Host: item.SSH.Host, Port: item.SSH.Port,
			Username: item.SSH.Username, WorkspaceRoot: item.WorkspaceRoot,
			HostKeyFingerprint: item.SSH.HostKeyFingerprint,
		}
	}
	return view
}
