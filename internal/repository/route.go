package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RouteStore persists ingress routes.
type RouteStore interface {
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.RouteView], error)
	RouteView(ctx context.Context, projectId, id string) (model.RouteView, error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
	ListEnabledRoutesByProject(ctx context.Context, projectId string) ([]model.Route, error)
	Route(ctx context.Context, projectId, id string) (model.Route, error)
	RouteByProjectAndDomain(ctx context.Context, projectId string, domain string) (model.Route, error)
	CreateRoute(ctx context.Context, route model.Route) error
	UpdateRoute(ctx context.Context, projectId string, route model.Route) error
	UpdateRouteCertificate(ctx context.Context, projectId string, route model.Route) (bool, error)
	DeleteRoute(ctx context.Context, projectId, id string) error
}
