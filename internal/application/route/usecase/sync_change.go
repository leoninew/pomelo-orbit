package routesvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type syncPlan struct {
	ids          []string
	routes       []model.Route
	originals    []model.Route
	updates      []model.Route
	changes      []routedto.RouteSyncChange
	publications map[string]routeport.Publication
	gateway      model.GatewayConfig
	dependencies string
}

func (s Service) buildSyncPlan(ctx context.Context, projectId string, input routedto.RouteSyncPreviewInput, gateway model.GatewayConfig) (syncPlan, error) {
	plan := syncPlan{changes: input.Changes, gateway: gateway, publications: make(map[string]routeport.Publication)}
	switch input.Scope {
	case "project":
		if len(input.RouteIds) != 0 {
			return plan, apperror.New(apperror.KindValidation, "project scope does not accept route_ids")
		}
		routes, err := s.route.ListAllRoutes(ctx, projectId)
		if err != nil {
			return plan, apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
		}
		ids := make(map[string]bool)
		for _, route := range routes {
			ids[route.Id] = true
		}
		publishedIds, err := s.routePublisher.ListPublicationRouteIds(ctx, projectId, gateway)
		if err != nil {
			return plan, err
		}
		for _, id := range publishedIds {
			ids[id] = true
		}
		for id := range ids {
			plan.ids = append(plan.ids, id)
		}
		sort.Strings(plan.ids)
	case "selected":
		if len(input.RouteIds) == 0 {
			return plan, apperror.New(apperror.KindValidation, "selected scope requires route_ids")
		}
		plan.ids = append([]string(nil), input.RouteIds...)
	default:
		return plan, apperror.New(apperror.KindValidation, "scope must be selected or project")
	}
	seen := make(map[string]bool)
	for _, id := range plan.ids {
		if id == "" || seen[id] {
			return plan, apperror.New(apperror.KindValidation, "route_ids must contain unique nonempty identities")
		}
		seen[id] = true
	}
	changes := make(map[string]routedto.RouteSyncChange)
	for _, change := range input.Changes {
		if !seen[change.RouteId] || change.Enabled == nil {
			return plan, apperror.New(apperror.KindValidation, "sync changes require an enabled value and an identity in the selected scope")
		}
		if _, found := changes[change.RouteId]; found {
			return plan, apperror.New(apperror.KindValidation, "duplicate route sync change")
		}
		changes[change.RouteId] = change
	}
	processedIds := make([]string, 0, len(plan.ids))
	for _, id := range plan.ids {
		original, err := s.route.Route(ctx, projectId, id)
		deleted := errors.Is(err, repository.ErrNotFound)
		if deleted {
			original = model.Route{Id: id}
		} else if err != nil {
			return plan, err
		}
		published, err := s.routePublisher.InspectPublication(ctx, projectId, gateway, original)
		if err != nil {
			return plan, err
		}
		if published != nil {
			plan.publications[id] = *published
		}
		if deleted {
			if published == nil {
				if input.Scope == "project" {
					continue
				}
				return plan, apperror.New(apperror.KindNotFound, "Route not found")
			}
			original = published.Route
			original.Enabled = false
			if _, found := changes[id]; found {
				return plan, apperror.New(apperror.KindValidation, "a deleted Route cannot receive sync changes")
			}
		}
		processedIds = append(processedIds, id)
		plan.originals = append(plan.originals, original)
		desired := original
		if change, found := changes[id]; found {
			desired.Enabled = *change.Enabled
			plan.updates = append(plan.updates, desired)
		}
		plan.routes = append(plan.routes, desired)
	}
	plan.ids = processedIds
	return plan, nil
}

func (plan syncPlan) routePlan(index int) syncPlan {
	route := plan.routes[index]
	one := syncPlan{
		ids: []string{route.Id}, routes: []model.Route{route}, originals: []model.Route{plan.originals[index]},
		publications: plan.publications, gateway: plan.gateway, dependencies: plan.dependencies,
	}
	for _, change := range plan.changes {
		if change.RouteId == route.Id {
			one.changes = append(one.changes, change)
		}
	}
	for _, update := range plan.updates {
		if update.Id == route.Id {
			one.updates = append(one.updates, update)
		}
	}
	return one
}

func previewSyncPlan(plan syncPlan) routedto.RouteSyncPreview {
	preview := routedto.RouteSyncPreview{Items: make([]routedto.RouteSyncPlanItem, 0, len(plan.routes))}
	for index, route := range plan.routes {
		action := "publish"
		if !route.Enabled {
			action = "withdraw"
			if _, found := plan.publications[route.Id]; !found {
				action = "skip"
			}
		}
		item := routedto.RouteSyncPlanItem{RouteId: route.Id, RouteName: route.Name, Action: action, Rule: plannedRouteRule(route)}
		item.BusinessHash, item.PublicationHash = syncPlanHashes(plan.routePlan(index))
		if route.Enabled && route.HTTPSEnabled {
			item.CertType = route.CertType
			item.AcmeChallenge = route.AcmeChallenge
		}
		preview.Items = append(preview.Items, item)
	}
	preview.RouteIds = append([]string(nil), plan.ids...)
	preview.BusinessHash, preview.PublicationHash = syncPlanHashes(plan)
	return preview
}

func syncPlanHashes(plan syncPlan) (string, string) {
	businessHash := hashSyncValue(struct {
		Routes    []model.Route
		Originals []model.Route
		Changes   []routedto.RouteSyncChange
		Gateway   model.GatewayConfig
		Ids       []string
	}{plan.routes, plan.originals, append([]routedto.RouteSyncChange(nil), plan.changes...), plan.gateway, plan.ids})
	files := make([]routeport.Publication, 0, len(plan.ids))
	for _, id := range plan.ids {
		if item, found := plan.publications[id]; found {
			files = append(files, item)
		}
	}
	publicationHash := hashSyncValue(struct {
		Files                 []routeport.Publication
		Dependencies          string
		RuntimeDirectory      string
		RuntimeTargetRevision int64
	}{files, plan.dependencies, plan.gateway.RuntimeDirectory, plan.gateway.RuntimeTargetRevision})
	return businessHash, publicationHash
}

func plannedRouteRule(route model.Route) *routedto.RouteSyncRule {
	if route.Protocol == routeProtocolTCP {
		match := ""
		if route.ListenPort != nil {
			match = fmt.Sprintf(":%d", *route.ListenPort)
		}
		return &routedto.RouteSyncRule{Protocol: routeProtocolTCP, Match: match, Target: net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))}
	}
	protocol := routeProtocolHTTP
	if route.HTTPSEnabled {
		protocol = "https"
	}
	match := "Host(`" + route.Domain + "`)"
	if route.PathPrefix != "" && route.PathPrefix != "/" {
		match += " && PathPrefix(`" + route.PathPrefix + "`)"
	}
	target := route.TargetUrl
	if route.TargetAddress != "" && route.TargetPort > 0 {
		target = "http://" + net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))
	}
	return &routedto.RouteSyncRule{Protocol: protocol, Match: match, Target: target}
}

func hashSyncValue(value any) string {
	encoded, _ := json.Marshal(value)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func (s Service) applyRouteSyncChanges(ctx context.Context, projectId string, plan syncPlan) error {
	for _, desired := range plan.updates {
		current, err := s.route.Route(ctx, projectId, desired.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to reload route for sync", err)
		}
		var original model.Route
		for _, item := range plan.originals {
			if item.Id == desired.Id {
				original = item
			}
		}
		if hashSyncValue(current) != hashSyncValue(original) {
			return apperror.NewWithCode(apperror.KindConflict, routeSyncPreviewExpiredCode, "Route sync preview is out of date. Preview again.")
		}
		if err := s.route.UpdateRoute(ctx, projectId, desired); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) verifyAndSaveSyncRoute(ctx context.Context, projectId string, plan syncPlan) error {
	original := plan.originals[0]
	current, err := s.route.Route(ctx, projectId, original.Id)
	if errors.Is(err, repository.ErrNotFound) && len(plan.updates) == 0 && !original.Enabled {
		return nil
	}
	if err != nil {
		return err
	}
	if hashSyncValue(current) != hashSyncValue(original) {
		return apperror.NewWithCode(apperror.KindConflict, routeSyncPreviewExpiredCode, "Route changed after preview. Preview again.")
	}
	return s.applyRouteSyncChanges(ctx, projectId, plan)
}
