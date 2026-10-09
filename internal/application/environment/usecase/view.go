package environmentsvc

import (
	"context"
	"errors"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func (s Service) viewWithTargetWarning(ctx context.Context, item model.Environment) (environmentdto.View, error) {
	view := s.toView(item)
	reader, ok := s.environments.(environmentTargetReader)
	if !ok {
		return view, nil
	}
	host, port := "", 0
	if item.SSH != nil {
		host, port = item.SSH.Host, item.SSH.Port
	}
	// Matching configuration suggests overlap; it does not identify a Docker daemon.
	_, err := reader.EnvironmentByTarget(ctx, item.ProjectId, item.TargetType, host, port)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to check environment target", err)
	}
	view.TargetMayBeShared = err == nil
	return view, nil
}

func (s Service) toView(item model.Environment) environmentdto.View {
	view := environmentdto.View{
		Id: item.Id, ProjectId: item.ProjectId, Code: item.Code,
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
