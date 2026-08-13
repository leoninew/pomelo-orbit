package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RouteStore persists ingress routes.
type RouteStore interface {
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Route], error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
	ListEnabledRoutes(ctx context.Context) ([]model.Route, error)
	Route(ctx context.Context, id string) (model.Route, error)
	RouteByDomain(ctx context.Context, domain string) (model.Route, error)
	CreateRoute(ctx context.Context, route model.Route) error
	UpdateRoute(ctx context.Context, route model.Route) error
	DeleteRoute(ctx context.Context, id string) error
}
