package routesvc

import (
	"context"
	"errors"
	"sort"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// A recreated Route owns its sync action, including withdrawal of same-name
// publications whose business Route has been deleted. Keep those prerequisites
// in its frozen publication hash rather than exposing historical IDs as items.
func (s Service) planOrphanWithdrawals(ctx context.Context, projectId string, plan *syncPlan, deletedIds map[string]bool) error {
	destinations := make(map[string]string)
	for _, route := range plan.routes {
		if deletedIds[route.Id] {
			continue
		}
		published, found := plan.publications[route.Id]
		if !found || !published.Route.Enabled || published.Route.Name != route.Name {
			destinations[route.Name] = route.Id
		}
	}
	if len(destinations) == 0 {
		return nil
	}
	names := make([]string, 0, len(destinations))
	for name := range destinations {
		names = append(names, name)
	}
	sort.Strings(names)
	owners, err := s.routePublisher.FindPublicationOwners(ctx, projectId, plan.gateway, names)
	if err != nil {
		return err
	}
	withdrawn := make(map[string]bool)
	for _, name := range names {
		currentId := destinations[name]
		for _, publication := range owners[name] {
			id := publication.Route.Id
			if !deletedIds[id] {
				_, err := s.route.Route(ctx, projectId, id)
				if err == nil {
					continue // A live Route retains its files, including unsynced edits.
				}
				if !errors.Is(err, repository.ErrNotFound) {
					return err
				}
			}
			if len(plan.withdrawals[currentId]) != 0 {
				return apperror.NewWithCode(apperror.KindConflict, "route_sync_publication_ownership_conflict", "Multiple deleted Routes claim the same Route file.")
			}
			plan.withdrawals[currentId] = append(plan.withdrawals[currentId], publication)
			withdrawn[id] = true
		}
	}
	if len(withdrawn) == 0 {
		return nil
	}
	ids, routes, originals := plan.ids, plan.routes, plan.originals
	plan.ids, plan.routes, plan.originals = nil, nil, nil
	for index, id := range ids {
		if !withdrawn[id] {
			plan.ids = append(plan.ids, id)
			plan.routes = append(plan.routes, routes[index])
			plan.originals = append(plan.originals, originals[index])
		}
	}
	return nil
}

func (s Service) publishSyncRoute(ctx context.Context, projectId string, plan syncPlan, withdrawn map[string]bool) (routeport.PublicationResult, error) {
	route := plan.routes[0]
	result := routeport.PublicationResult{FileCommit: "not_applicable", ConfigurationMatch: "not_applicable", CertificateVerification: "not_applicable", Recovery: "not_needed", Cleanup: "not_applicable"}
	for _, publication := range plan.withdrawals[route.Id] {
		if withdrawn[publication.Route.Id] {
			continue
		}
		withdrawal := publication.Route
		withdrawal.Enabled = false
		var err error
		result, err = s.routePublisher.PublishRoute(ctx, projectId, plan.gateway, withdrawal, hashSyncValue(publication))
		if err != nil {
			return result, err
		}
		withdrawn[withdrawal.Id] = true
	}
	fingerprint := ""
	if publication, found := plan.publications[route.Id]; found {
		fingerprint = hashSyncValue(publication)
	}
	if !route.Enabled && fingerprint == "" {
		return result, nil
	}
	return s.routePublisher.PublishRoute(ctx, projectId, plan.gateway, route, fingerprint)
}
