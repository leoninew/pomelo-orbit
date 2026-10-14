package routesvc

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// routeRuleFacts supplies stored inputs and records ordinary writes. It does
// not emulate SQL constraints, revision checks, transactions or publication rules.
type routeRuleFacts struct {
	repository.ProjectReader
	repository.RouteStore
	repository.GatewayStore
	routes  map[string]model.Route
	gateway model.GatewayConfig
	updates int
}

func newRouteRuleService(names ...string) (Service, *recordingRoutePublisher, *recordingTraefikClient, *routeRuleFacts) {
	facts := &routeRuleFacts{
		routes:  make(map[string]model.Route),
		gateway: model.GatewayConfig{ApplicationId: "gateway", AcmeProfile: "http-dns"},
	}
	projectId := routeTestProjectId
	for _, name := range names {
		facts.routes[name] = model.Route{Id: name, ProjectId: &projectId, Name: name, Protocol: "http", Domain: name + ".example.test", PathPrefix: "/", TargetUrl: "http://" + name + ":8080", Enabled: true, CertType: certTypeManual, AcmeChallenge: acmeChallengeHTTP}
	}
	publisher, client := &recordingRoutePublisher{}, &recordingTraefikClient{}
	return New(facts, nil, nil, facts, facts, publisher, nil, client, routeRuleTransaction{}, testRouteSyncTimeouts()), publisher, client, facts
}

func (*routeRuleFacts) Project(_ context.Context, projectId string) (model.Project, error) {
	return model.Project{Id: projectId, Code: "route-project"}, nil
}

func (*routeRuleFacts) IsProjectMember(_ context.Context, projectId, userId string) (bool, error) {
	return projectId == routeTestProjectId && userId == routeTestUserId, nil
}

func (f *routeRuleFacts) GatewayConfigByProject(context.Context, string) (model.GatewayConfig, error) {
	return f.gateway, nil
}

func (f *routeRuleFacts) Route(_ context.Context, _ string, id string) (model.Route, error) {
	if route, ok := f.routes[id]; ok {
		return route, nil
	}
	return model.Route{}, repository.ErrNotFound
}

func (f *routeRuleFacts) ListAllRoutes(context.Context, string) ([]model.Route, error) {
	routes := make([]model.Route, 0, len(f.routes))
	for _, route := range f.routes {
		routes = append(routes, route)
	}
	return routes, nil
}

func (f *routeRuleFacts) UpdateRoute(_ context.Context, _ string, route model.Route) error {
	f.updates++
	f.routes[route.Id] = route
	return nil
}

// The migrated sync cases perform no business writes; this only executes the
// usecase callback. Rollback and persistence assertions remain in integration tests.
type routeRuleTransaction struct{}

func (routeRuleTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
