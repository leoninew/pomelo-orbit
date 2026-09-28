package routesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type syncPlan struct {
	routes    []model.Route
	originals []model.Route
	updates   []model.Route
	changes   []routedto.RouteSyncChange
}

func (s Service) buildSyncPlan(ctx context.Context, projectId string, changes []routedto.RouteSyncChange) (syncPlan, error) {
	enabledRoutes, err := s.route.ListEnabledRoutesByProject(ctx, projectId)
	if err != nil {
		return syncPlan{}, apperror.Wrap(apperror.KindInternal, "Failed to list enabled routes", err)
	}
	byId := make(map[string]model.Route, len(enabledRoutes)+len(changes))
	for _, route := range enabledRoutes {
		byId[route.Id] = route
	}
	plan := syncPlan{changes: changes}
	seen := make(map[string]struct{}, len(changes))
	for _, change := range changes {
		if change.RouteId == "" || change.Enabled == nil {
			return syncPlan{}, apperror.New(apperror.KindValidation, "route sync change requires route_id and enabled")
		}
		if _, duplicate := seen[change.RouteId]; duplicate {
			return syncPlan{}, apperror.New(apperror.KindValidation, "a route can only appear once in sync changes")
		}
		seen[change.RouteId] = struct{}{}
		original, err := s.route.Route(ctx, projectId, change.RouteId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return syncPlan{}, apperror.New(apperror.KindNotFound, "Route not found")
			}
			return syncPlan{}, apperror.Wrap(apperror.KindInternal, "Failed to load route for sync", err)
		}
		desired := original
		if change.Enabled != nil {
			desired.Enabled = *change.Enabled
		}
		plan.originals = append(plan.originals, original)
		plan.updates = append(plan.updates, desired)
		if desired.Enabled {
			byId[desired.Id] = desired
		} else {
			delete(byId, desired.Id)
		}
	}
	plan.routes = make([]model.Route, 0, len(byId))
	for _, route := range byId {
		plan.routes = append(plan.routes, route)
	}
	sort.Slice(plan.routes, func(i, j int) bool { return plan.routes[i].Id < plan.routes[j].Id })
	return plan, nil
}

func previewSyncPlan(plan syncPlan, routers []routeport.TraefikRouter, services []routeport.TraefikService) routedto.RouteSyncPreview {
	preview := compareSyncState(plan.routes, routers, services)
	if len(plan.changes) != 0 {
		preview.BusinessHash = hashSyncValue(struct {
			Snapshot  string
			Originals []model.Route
			Changes   []routedto.RouteSyncChange
		}{preview.BusinessHash, plan.originals, plan.changes})
	}
	return preview
}

func hashSyncValue(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func (s Service) applyRouteSyncChanges(ctx context.Context, projectId string, plan syncPlan) error {
	for index, desired := range plan.updates {
		current, err := s.route.Route(ctx, projectId, desired.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to reload route for sync", err)
		}
		if hashSyncValue(current) != hashSyncValue(plan.originals[index]) {
			return apperror.NewWithCode(apperror.KindConflict, routeSyncPreviewExpiredCode, "Route sync preview is out of date. Preview the current state again.")
		}
		if err := s.route.UpdateRoute(ctx, projectId, desired); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to update route sync state", err)
		}
	}
	return nil
}
