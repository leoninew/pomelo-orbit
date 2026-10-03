package traefik

import (
	"context"
	"errors"
	"log/slog"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/common/operation"
)

type syncSessionKey struct{}

type syncSession struct {
	manager   *RouteManager
	target    environmentport.Target
	base      string
	gatewayId string
}

func (m *RouteManager) OpenSession(ctx context.Context, projectId string) (context.Context, func(), error) {
	resolveCtx, finish := operation.StartStage(ctx, "Route sync", "resolve_target", m.timeouts.StateLoad)
	target, err := m.resolveTarget(resolveCtx, projectId)
	if err = finish(err); err != nil {
		return nil, nil, err
	}
	ctx = operation.WithLogAttrs(ctx, slog.String("project_code", target.Environment.Code))
	sessions, ok := m.runtime.(deploymentport.RuntimeSessions)
	if !ok {
		return nil, nil, errors.New("route runtime sessions are not configured")
	}
	ctx, closeSession, err := sessions.OpenSession(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	return context.WithValue(ctx, syncSessionKey{}, &syncSession{manager: m, target: target}), closeSession, nil
}

func (m *RouteManager) session(ctx context.Context) *syncSession {
	session, _ := ctx.Value(syncSessionKey{}).(*syncSession)
	if session != nil && session.manager == m {
		return session
	}
	return nil
}

func (m *RouteManager) ensureSession(ctx context.Context, projectId string) (context.Context, func(), error) {
	if session := m.session(ctx); session != nil {
		if session.target.Environment.ProjectId != projectId {
			return nil, nil, apperror.New(apperror.KindConflict, "Route synchronization session belongs to another Project")
		}
		return ctx, func() {}, nil
	}
	return m.OpenSession(ctx, projectId)
}
