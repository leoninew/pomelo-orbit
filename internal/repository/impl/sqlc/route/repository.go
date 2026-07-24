package routerepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	routesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/route"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.RouteStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *routesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *routesqlc.Queries {
		return routesqlc.New(dbtx)
	})
}

func (r Repository) ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	q := r.q(ctx)
	total, err := q.CountRoutes(ctx, routesqlc.CountRoutesParams{
		ProjectID: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:   raw, Name: pattern, Domain: pattern, TargetUrl: pattern,
	})
	if err != nil {
		return repository.Page[model.Route]{}, fmt.Errorf("count routes: %w", err)
	}
	rows, err := q.ListRoutes(ctx, routesqlc.ListRoutesParams{
		ProjectID: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:   raw, Name: pattern, Domain: pattern, TargetUrl: pattern,
		Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Route]{}, fmt.Errorf("list routes: %w", err)
	}
	items := make([]model.Route, 0, len(rows))
	for _, row := range rows {
		items = append(items, routeFromRow(row))
	}
	return repository.Page[model.Route]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error) {
	rows, err := r.q(ctx).ListAllRoutes(ctx, sql.NullString{String: strings.TrimSpace(projectId), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list all routes: %w", err)
	}
	items := make([]model.Route, 0, len(rows))
	for _, row := range rows {
		items = append(items, routeFromAll(row))
	}
	return items, nil
}

func (r Repository) ListEnabledRoutes(ctx context.Context) ([]model.Route, error) {
	rows, err := r.q(ctx).ListEnabledRoutes(ctx, 1)
	if err != nil {
		return nil, fmt.Errorf("list enabled routes: %w", err)
	}
	items := make([]model.Route, 0, len(rows))
	for _, row := range rows {
		items = append(items, routeFromEnabled(row))
	}
	return items, nil
}

func (r Repository) Route(ctx context.Context, id string) (model.Route, error) {
	row, err := r.q(ctx).RouteByID(ctx, id)
	if err != nil {
		return model.Route{}, fmt.Errorf("load route %s: %w", id, sqlcommon.TranslateError(err))
	}
	return routeFromByID(row), nil
}

func (r Repository) RouteByDomain(ctx context.Context, domain string) (model.Route, error) {
	row, err := r.q(ctx).RouteByDomain(ctx, strings.TrimSpace(domain))
	if err != nil {
		return model.Route{}, fmt.Errorf("load route by domain %s: %w", domain, sqlcommon.TranslateError(err))
	}
	return routeFromByDomain(row), nil
}

func (r Repository) CreateRoute(ctx context.Context, route model.Route) error {
	now := time.Now().UTC()
	err := r.q(ctx).CreateRoute(ctx, routesqlc.CreateRouteParams{
		ID: route.Id, ProjectID: dbmodel.NullString(route.ProjectId), Name: route.Name, Domain: route.Domain,
		PathPrefix: route.PathPrefix, TargetUrl: route.TargetURL, Enabled: dbmodel.BoolInt(route.Enabled),
		HttpsEnabled: dbmodel.BoolInt(route.HTTPSEnabled), CertPem: dbmodel.NullString(route.CertPEM),
		CertKey: dbmodel.NullString(route.CertKey), CertType: route.CertType,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("create route %s: %w", route.Name, err)
	}
	return nil
}

func (r Repository) UpdateRoute(ctx context.Context, route model.Route) error {
	err := r.q(ctx).UpdateRoute(ctx, routesqlc.UpdateRouteParams{
		Name: route.Name, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetUrl: route.TargetURL,
		Enabled: dbmodel.BoolInt(route.Enabled), HttpsEnabled: dbmodel.BoolInt(route.HTTPSEnabled),
		CertPem: dbmodel.NullString(route.CertPEM), CertKey: dbmodel.NullString(route.CertKey),
		CertType: route.CertType, UpdatedAt: time.Now().UTC(), ID: route.Id,
	})
	if err != nil {
		return fmt.Errorf("update route %s: %w", route.Id, err)
	}
	return nil
}

func (r Repository) DeleteRoute(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteRoute(ctx, id); err != nil {
		return fmt.Errorf("delete route %s: %w", id, err)
	}
	return nil
}

func routeCommon(id string, projectID sql.NullString, name, domain, pathPrefix, targetURL string, enabled, httpsEnabled int64, certPem, certKey sql.NullString, certType string, createdAt, updatedAt time.Time) model.Route {
	return model.Route{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), Name: name, Domain: domain, PathPrefix: pathPrefix,
		TargetURL: targetURL, Enabled: dbmodel.IntBool(enabled), HTTPSEnabled: dbmodel.IntBool(httpsEnabled),
		CertPEM: dbmodel.StringPtr(certPem), CertKey: dbmodel.StringPtr(certKey), CertType: certType,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func routeFromRow(row routesqlc.ListRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Domain, row.PathPrefix, row.TargetUrl, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.CreatedAt, row.UpdatedAt)
}
func routeFromAll(row routesqlc.ListAllRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Domain, row.PathPrefix, row.TargetUrl, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.CreatedAt, row.UpdatedAt)
}
func routeFromEnabled(row routesqlc.ListEnabledRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Domain, row.PathPrefix, row.TargetUrl, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.CreatedAt, row.UpdatedAt)
}
func routeFromByID(row routesqlc.RouteByIDRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Domain, row.PathPrefix, row.TargetUrl, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.CreatedAt, row.UpdatedAt)
}
func routeFromByDomain(row routesqlc.RouteByDomainRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Domain, row.PathPrefix, row.TargetUrl, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.CreatedAt, row.UpdatedAt)
}
