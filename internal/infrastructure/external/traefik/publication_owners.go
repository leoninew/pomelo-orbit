package traefik

import (
	"context"
	"errors"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/common/operation"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

// An interrupted rename can still own both paths until it is reconciled.
func publicationOwnerNames(item routeport.Publication) []string {
	var names []string
	if item.Route.Enabled {
		names = append(names, item.Route.Name)
	}
	if item.Phase == "pending" && item.Previous != nil && item.Previous.Route.Enabled &&
		(!item.Route.Enabled || item.Previous.Route.Name != item.Route.Name) {
		names = append(names, item.Previous.Route.Name)
	}
	return names
}

func (m *RouteManager) FindPublicationOwners(ctx context.Context, projectId string, gateway model.GatewayConfig, routeNames []string) (owners map[string][]routeport.Publication, resultErr error) {
	owners = make(map[string][]routeport.Publication)
	if len(routeNames) == 0 {
		return owners, nil
	}
	names := make(map[string]bool, len(routeNames))
	for _, name := range routeNames {
		if !validPublicationId(name) {
			return nil, errors.New("invalid Route publication code")
		}
		names[name] = true
	}
	ctx, closeSession, err := m.ensureSession(ctx, projectId)
	if err != nil {
		return nil, err
	}
	defer closeSession()
	ctx, finish := operation.StartStage(ctx, "Route sync", "inspect_publication_owners", m.timeouts.StateLoad)
	defer func() { resultErr = finish(resultErr) }()
	target, files, base, err := m.publicationWorkspace(ctx, projectId, gateway)
	if err != nil {
		return nil, err
	}
	ids, err := m.ListPublicationRouteIds(ctx, projectId, gateway)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		item, err := readPublicationRecord(ctx, projectId, gateway, target, files, base, model.Route{Id: id}, names)
		if err != nil {
			return nil, err
		}
		if item == nil {
			continue
		}
		if err := inspectPublication(ctx, target, files, base, item); err != nil {
			return nil, err
		}
		for _, name := range publicationOwnerNames(*item) {
			if names[name] {
				owners[name] = append(owners[name], *item)
			}
		}
	}
	return owners, nil
}
