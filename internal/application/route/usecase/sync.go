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
	"strings"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const routeSyncPreviewExpiredCode = "route_sync_preview_expired"

type syncRouter struct {
	Name       string `json:"name"`
	Rule       string `json:"rule"`
	Protocol   string `json:"protocol"`
	ListenPort int    `json:"listen_port,omitempty"`
	Service    string `json:"-"`
}

type syncService struct {
	Name     string   `json:"name"`
	Protocol string   `json:"protocol"`
	Servers  []string `json:"servers"`
}

type syncSnapshot struct {
	Routers  []syncRouter  `json:"routers"`
	Services []syncService `json:"services"`
}

// syncRouteState keeps the router and its referenced service together for a
// user-facing route comparison. Service references are intentionally excluded
// from snapshot hashes because they are Traefik implementation details.
type syncRouteState struct {
	Router     syncRouter
	HasRouter  bool
	Service    syncService
	HasService bool
}

// PreviewRouteSync compares the requested business snapshot with Traefik
// without changing either side.
func (s Service) PreviewRouteSync(ctx context.Context, userId string, projectId string, changes []routedto.RouteSyncChange) (routedto.RouteSyncPreview, error) {
	routes, traefikRouters, traefikServices, err := s.loadSyncState(ctx, userId, projectId, changes)
	if err != nil {
		return routedto.RouteSyncPreview{}, err
	}
	return compareSyncState(routes, traefikRouters, traefikServices), nil
}

// ConfirmRouteSync applies the requested business state and then replaces the
// complete Traefik REST snapshot. The database transaction intentionally ends
// before the external PUT starts.
func (s Service) ConfirmRouteSync(ctx context.Context, userId string, projectId string, input routedto.RouteSyncConfirmInput) error {
	if strings.TrimSpace(input.BusinessHash) == "" || strings.TrimSpace(input.TraefikHash) == "" {
		return apperror.New(apperror.KindValidation, "business_hash and traefik_hash are required")
	}
	routes, traefikRouters, traefikServices, err := s.loadSyncState(ctx, userId, projectId, input.Changes)
	if err != nil {
		return err
	}
	preview := compareSyncState(routes, traefikRouters, traefikServices)
	if preview.BusinessHash != input.BusinessHash || preview.TraefikHash != input.TraefikHash {
		return apperror.NewWithCode(
			apperror.KindConflict,
			routeSyncPreviewExpiredCode,
			"Route sync preview is out of date. Preview the current state again.",
		)
	}
	if s.transactionRunner == nil {
		return apperror.New(apperror.KindInternal, "route sync transaction is not configured")
	}

	if err := s.transactionRunner.RunInTransaction(ctx, func(txCtx context.Context) error {
		return s.applyRouteSyncChanges(txCtx, projectId, input.Changes)
	}); err != nil {
		return err
	}

	updatedRoutes, err := s.listEnabledRoutesForPublish(ctx, projectId)
	if err != nil {
		return err
	}
	return s.applyRouteSnapshot(ctx, projectId, updatedRoutes, false)
}

func (s Service) loadSyncState(ctx context.Context, userId string, projectId string, changes []routedto.RouteSyncChange) ([]model.Route, []routeport.TraefikRouter, []routeport.TraefikService, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return nil, nil, nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return nil, nil, nil, err
	}
	routes, err := s.syncCandidateRoutes(ctx, projectId, changes)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := s.prepareSyncRoutes(ctx, routes); err != nil {
		return nil, nil, nil, err
	}
	gateway, err := s.resolveGatewayForRender(ctx, projectId)
	if err != nil {
		return nil, nil, nil, err
	}
	items, err := s.traefikRouterClient.ListRouters(ctx, projectId, *gateway)
	if err != nil {
		if s.traefikRouterClient.IsConnectionError(err) {
			return nil, nil, nil, apperror.Wrap(apperror.KindUnavailable, "Traefik is unavailable.", err)
		}
		return nil, nil, nil, apperror.Wrap(apperror.KindInternal, "Failed to inspect Traefik routers", err)
	}
	services, err := s.traefikRouterClient.ListServices(ctx, projectId, *gateway)
	if err != nil {
		if s.traefikRouterClient.IsConnectionError(err) {
			return nil, nil, nil, apperror.Wrap(apperror.KindUnavailable, "Traefik is unavailable.", err)
		}
		return nil, nil, nil, apperror.Wrap(apperror.KindInternal, "Failed to inspect Traefik services", err)
	}
	return routes, items, services, nil
}

func (s Service) syncCandidateRoutes(ctx context.Context, projectId string, changes []routedto.RouteSyncChange) ([]model.Route, error) {
	enabledRoutes, err := s.route.ListEnabledRoutesByProject(ctx, projectId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list enabled routes", err)
	}
	byId := make(map[string]model.Route, len(enabledRoutes)+len(changes))
	for _, route := range enabledRoutes {
		byId[route.Id] = route
	}
	requested := make(map[string]bool, len(changes))
	for _, change := range changes {
		routeId := strings.TrimSpace(change.RouteId)
		if routeId == "" {
			return nil, apperror.New(apperror.KindValidation, "route_id is required")
		}
		if previous, found := requested[routeId]; found && previous != change.Enabled {
			return nil, apperror.New(apperror.KindValidation, "a route cannot have conflicting sync changes")
		}
		requested[routeId] = change.Enabled

		route, err := s.route.Route(ctx, routeId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
			}
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load route for sync", err)
		}
		if route.ProjectId == nil || *route.ProjectId != projectId {
			return nil, apperror.New(apperror.KindForbidden, "Permission denied")
		}
		route.Enabled = change.Enabled
		if change.Enabled {
			byId[route.Id] = route
		} else {
			delete(byId, route.Id)
		}
	}

	routes := make([]model.Route, 0, len(byId))
	for _, route := range byId {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Id < routes[j].Id })
	return routes, nil
}

func (s Service) prepareSyncRoutes(ctx context.Context, routes []model.Route) error {
	tcpPorts := make(map[int]string)
	for index := range routes {
		route := &routes[index]
		if route.Protocol != routeProtocolTCP {
			if err := s.validateRoute(ctx, route, route.Id); err != nil {
				return err
			}
			continue
		}
		if route.ListenPort == nil || *route.ListenPort < 1 || *route.ListenPort > 65535 || route.ServiceId == nil || route.ComponentName == nil || route.EndpointProtocol == nil || route.EndpointContainerPort == nil {
			return apperror.New(apperror.KindValidation, "Invalid TCP route fields")
		}
		if reservedTCPRoutePort(*route.ListenPort) {
			return apperror.New(apperror.KindValidation, fmt.Sprintf("TCP listen port %d is reserved by Gateway", *route.ListenPort))
		}
		if existing, found := tcpPorts[*route.ListenPort]; found && existing != route.Name {
			return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d is already used by route %s", *route.ListenPort, existing))
		}
		if err := s.resolveManagedRouteTarget(ctx, route); err != nil {
			return err
		}
		if route.ProjectId == nil || strings.TrimSpace(*route.ProjectId) == "" {
			return apperror.New(apperror.KindValidation, "Route project is required")
		}
		if conflict, err := s.componentPortConflict(ctx, *route.ProjectId, *route.ListenPort); err != nil {
			return err
		} else if conflict != "" {
			return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d conflicts with component endpoint %s", *route.ListenPort, conflict))
		}
		tcpPorts[*route.ListenPort] = route.Name
	}
	return nil
}

func (s Service) applyRouteSyncChanges(ctx context.Context, projectId string, changes []routedto.RouteSyncChange) error {
	for _, change := range changes {
		routeId := strings.TrimSpace(change.RouteId)
		route, err := s.route.Route(ctx, routeId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
			}
			return apperror.Wrap(apperror.KindInternal, "Failed to load route for sync", err)
		}
		if route.ProjectId == nil || *route.ProjectId != strings.TrimSpace(projectId) {
			return apperror.New(apperror.KindForbidden, "Permission denied")
		}
		route.Enabled = change.Enabled
		if err := s.route.UpdateRoute(ctx, route); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to update route sync state", err)
		}
	}
	return nil
}

func compareSyncState(routes []model.Route, traefikRouters []routeport.TraefikRouter, traefikServices []routeport.TraefikService) routedto.RouteSyncPreview {
	business := syncSnapshot{
		Routers:  expectedSyncRouters(routes),
		Services: expectedSyncServices(routes),
	}
	traefik := syncSnapshot{
		Routers:  actualSyncRouters(traefikRouters),
		Services: actualSyncServices(traefikServices),
	}
	businessHash := hashSyncSnapshot(business)
	traefikHash := hashSyncSnapshot(traefik)
	differences := syncDifferences(business, traefik)
	return routedto.RouteSyncPreview{
		BusinessHash: businessHash,
		TraefikHash:  traefikHash,
		Matched:      len(differences) == 0,
		Differences:  differences,
	}
}

func expectedSyncRouters(routes []model.Route) []syncRouter {
	items := make([]syncRouter, 0, len(routes))
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		name := strings.TrimSpace(route.Name)
		if name == "" {
			name = "route"
		}
		if route.Protocol == routeProtocolTCP {
			if route.ListenPort == nil || route.TargetAddress == "" || route.TargetPort < 1 {
				continue
			}
			items = append(items, syncRouter{
				Name:       name + "-route@rest",
				Rule:       "HostSNI(`*`)",
				Protocol:   routeProtocolTCP,
				ListenPort: *route.ListenPort,
				Service:    name + "-service",
			})
			continue
		}
		if strings.TrimSpace(syncHTTPServiceTarget(route)) == "" {
			continue
		}
		rule := "Host(`" + route.Domain + "`)"
		if route.PathPrefix != "" && route.PathPrefix != "/" {
			rule += " && PathPrefix(`" + route.PathPrefix + "`)"
		}
		protocol := "http"
		if route.HTTPSEnabled {
			protocol = "https"
		}
		item := syncRouter{Name: name + "-route@rest", Rule: rule, Protocol: protocol, Service: name + "-service"}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func actualSyncRouters(routers []routeport.TraefikRouter) []syncRouter {
	items := make([]syncRouter, 0, len(routers))
	for _, router := range routers {
		if router.Provider != "rest" {
			continue
		}
		items = append(items, syncRouter{
			Name:       router.Name,
			Rule:       router.Rule,
			Protocol:   syncRouterProtocol(router),
			ListenPort: syncRouterListenPort(router),
			Service:    syncServiceName(router.Service),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func expectedSyncServices(routes []model.Route) []syncService {
	items := make([]syncService, 0, len(routes))
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		name := strings.TrimSpace(route.Name)
		if name == "" {
			name = "route"
		}
		var server string
		if route.Protocol == routeProtocolTCP {
			if route.ListenPort == nil || route.TargetAddress == "" || route.TargetPort < 1 {
				continue
			}
			server = net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))
		} else {
			server = syncHTTPServiceTarget(route)
		}
		if strings.TrimSpace(server) == "" {
			continue
		}
		items = append(items, syncService{
			Name:     name + "-service",
			Protocol: route.Protocol,
			Servers:  []string{server},
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Protocol == items[j].Protocol {
			return items[i].Name < items[j].Name
		}
		return items[i].Protocol < items[j].Protocol
	})
	return items
}

func actualSyncServices(services []routeport.TraefikService) []syncService {
	items := make([]syncService, 0, len(services))
	for _, service := range services {
		if service.Provider != "rest" {
			continue
		}
		name := strings.TrimSuffix(service.Name, "@rest")
		servers := append([]string(nil), service.Servers...)
		sort.Strings(servers)
		items = append(items, syncService{
			Name:     name,
			Protocol: service.Protocol,
			Servers:  servers,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Protocol == items[j].Protocol {
			return items[i].Name < items[j].Name
		}
		return items[i].Protocol < items[j].Protocol
	})
	return items
}

func syncServiceKey(protocol string, name string) string {
	return protocol + "\x00" + name
}

func syncRouterProtocol(router routeport.TraefikRouter) string {
	if strings.HasPrefix(router.Rule, "HostSNI(") {
		return routeProtocolTCP
	}
	if router.TLS {
		return "https"
	}
	return "http"
}

func syncRouterListenPort(router routeport.TraefikRouter) int {
	if syncRouterProtocol(router) != routeProtocolTCP {
		return 0
	}
	for _, entrypoint := range router.Entrypoints {
		port, err := strconv.Atoi(strings.TrimPrefix(entrypoint, "tcp"))
		if err == nil && port > 0 {
			return port
		}
	}
	return 0
}

func syncHTTPServiceTarget(route model.Route) string {
	if route.TargetAddress != "" && route.TargetPort > 0 {
		return "http://" + net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))
	}
	return route.TargetUrl
}

func hashSyncSnapshot(snapshot syncSnapshot) string {
	encoded, _ := json.Marshal(snapshot)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

func syncDifferences(business syncSnapshot, traefik syncSnapshot) []routedto.RouteSyncDiff {
	businessRoutes := syncRouteStates(business)
	traefikRoutes := syncRouteStates(traefik)
	names := make([]string, 0, len(businessRoutes)+len(traefikRoutes))
	seen := make(map[string]struct{}, len(businessRoutes)+len(traefikRoutes))
	for name := range businessRoutes {
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for name := range traefikRoutes {
		if _, found := seen[name]; !found {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	differences := make([]routedto.RouteSyncDiff, 0, len(names))
	for _, name := range names {
		businessRoute, inBusiness := businessRoutes[name]
		traefikRoute, inTraefik := traefikRoutes[name]
		switch {
		case inBusiness && !inTraefik:
			differences = append(differences, routedto.RouteSyncDiff{
				Action:        "added",
				RouteName:     name,
				Field:         "route",
				BusinessValue: syncRouteValue(businessRoute),
			})
		case !inBusiness && inTraefik:
			differences = append(differences, routedto.RouteSyncDiff{
				Action:       "removed",
				RouteName:    name,
				Field:        "route",
				TraefikValue: syncRouteValue(traefikRoute),
			})
		case !syncRouteStatesEqual(businessRoute, traefikRoute):
			differences = append(differences, routedto.RouteSyncDiff{
				Action:        "modified",
				RouteName:     name,
				Field:         "route",
				BusinessValue: syncRouteValue(businessRoute),
				TraefikValue:  syncRouteValue(traefikRoute),
			})
		}
	}
	return differences
}

func syncRouteStates(snapshot syncSnapshot) map[string]syncRouteState {
	services := make(map[string]syncService, len(snapshot.Services))
	for _, service := range snapshot.Services {
		services[syncServiceKey(service.Protocol, service.Name)] = service
	}

	routes := make(map[string]syncRouteState, len(snapshot.Routers)+len(snapshot.Services))
	usedServices := make(map[string]struct{}, len(snapshot.Services))
	for _, router := range snapshot.Routers {
		name := syncRouterRouteName(router.Name)
		state := routes[name]
		state.Router = router
		state.HasRouter = true
		serviceKey := syncServiceKey(syncRouterServiceProtocol(router.Protocol), router.Service)
		if service, found := services[serviceKey]; found {
			state.Service = service
			state.HasService = true
			usedServices[serviceKey] = struct{}{}
		}
		routes[name] = state
	}
	for _, service := range snapshot.Services {
		serviceKey := syncServiceKey(service.Protocol, service.Name)
		if _, used := usedServices[serviceKey]; used {
			continue
		}
		name := syncServiceRouteName(service.Name)
		state := routes[name]
		if !state.HasService {
			state.Service = service
			state.HasService = true
			routes[name] = state
		}
	}
	return routes
}

func syncRouterServiceProtocol(protocol string) string {
	if protocol == routeProtocolTCP {
		return routeProtocolTCP
	}
	return "http"
}

func syncRouteStatesEqual(business syncRouteState, traefik syncRouteState) bool {
	if business.HasRouter != traefik.HasRouter || business.HasService != traefik.HasService {
		return false
	}
	if business.HasRouter && (business.Router.Rule != traefik.Router.Rule || business.Router.Protocol != traefik.Router.Protocol || business.Router.ListenPort != traefik.Router.ListenPort) {
		return false
	}
	if business.HasService && (business.Service.Protocol != traefik.Service.Protocol || !syncServersEqual(business.Service.Servers, traefik.Service.Servers)) {
		return false
	}
	return true
}

func syncServersEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index, server := range left {
		if server != right[index] {
			return false
		}
	}
	return true
}

func syncRouterValue(router syncRouter) string {
	if router.Protocol == routeProtocolTCP {
		if router.ListenPort > 0 {
			return "TCP :" + strconv.Itoa(router.ListenPort)
		}
		return "TCP"
	}
	return strings.TrimSpace(strings.ToUpper(router.Protocol) + " " + router.Rule)
}

func syncServiceField(service syncService) string {
	return strings.Join(service.Servers, ", ")
}

func syncRouteValue(state syncRouteState) string {
	value := ""
	if state.HasRouter {
		value = syncRouterValue(state.Router)
	}
	if !state.HasService {
		return value
	}
	serviceValue := syncServiceField(state.Service)
	if value == "" {
		return serviceValue
	}
	if serviceValue == "" {
		return value
	}
	return value + " -> " + serviceValue
}

func syncRouterRouteName(name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, "@rest"), "-route")
}

func syncServiceRouteName(name string) string {
	return strings.TrimSuffix(strings.TrimSuffix(name, "@rest"), "-service")
}

func syncServiceName(name string) string {
	return strings.TrimSuffix(name, "@rest")
}
