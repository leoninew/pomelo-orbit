package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RouteStore persists ingress routes.
type RouteStore interface {
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Route], error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
	ListEnabledRoutesByProject(ctx context.Context, projectId string) ([]model.Route, error)
	Route(ctx context.Context, id string) (model.Route, error)
	RouteByProjectAndDomain(ctx context.Context, projectId string, domain string) (model.Route, error)
	CreateRoute(ctx context.Context, route model.Route) error
	UpdateRoute(ctx context.Context, route model.Route) error
	DeleteRoute(ctx context.Context, id string) error
}
