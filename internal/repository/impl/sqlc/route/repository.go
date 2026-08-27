package routerepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	routesqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/route"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.RouteStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *routesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *routesqlc.Queries {
		return routesqlc.New(dbtx)
	})
}

func (r Repository) ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	projectID := strings.TrimSpace(projectId)
	q := r.q(ctx)
	total, err := q.CountRoutes(ctx, routesqlc.CountRoutesParams{
		ProjectID:     sql.NullString{String: projectID, Valid: true},
		SearchPattern: searchPattern,
	})
	if err != nil {
		return repository.Page[model.Route]{}, fmt.Errorf("count routes: %w", err)
	}
	rows, err := q.ListRoutes(ctx, routesqlc.ListRoutesParams{
		ProjectID:     sql.NullString{String: projectID, Valid: true},
		SearchPattern: searchPattern,
		Limit:         int32(perPage), Offset: int32((page - 1) * perPage),
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
	return routeFromById(row), nil
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
		ID: route.Id, ProjectID: dbmodel.NullString(route.ProjectId), Name: route.Name, Protocol: route.Protocol, Domain: route.Domain,
		PathPrefix: route.PathPrefix, TargetUrl: route.TargetUrl, ListenPort: dbmodel.NullInt64FromIntPtr(route.ListenPort),
		ServiceID: dbmodel.NullString(route.ServiceId), ComponentName: dbmodel.NullString(route.ComponentName), EndpointProtocol: dbmodel.NullString(route.EndpointProtocol), EndpointContainerPort: dbmodel.NullInt64FromIntPtr(route.EndpointContainerPort), Enabled: dbmodel.BoolInt(route.Enabled),
		HttpsEnabled: dbmodel.BoolInt(route.HTTPSEnabled), CertPem: dbmodel.NullString(route.CertPEM),
		CertKey: dbmodel.NullString(route.CertKey), CertType: route.CertType, AcmeChallenge: route.AcmeChallenge,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("create route %s: %w", route.Name, err)
	}
	return nil
}

func (r Repository) UpdateRoute(ctx context.Context, route model.Route) error {
	err := r.q(ctx).UpdateRoute(ctx, routesqlc.UpdateRouteParams{
		Name: route.Name, Protocol: route.Protocol, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetUrl: route.TargetUrl,
		ListenPort: dbmodel.NullInt64FromIntPtr(route.ListenPort), ServiceID: dbmodel.NullString(route.ServiceId), ComponentName: dbmodel.NullString(route.ComponentName), EndpointProtocol: dbmodel.NullString(route.EndpointProtocol), EndpointContainerPort: dbmodel.NullInt64FromIntPtr(route.EndpointContainerPort), Enabled: dbmodel.BoolInt(route.Enabled), HttpsEnabled: dbmodel.BoolInt(route.HTTPSEnabled),
		CertPem: dbmodel.NullString(route.CertPEM), CertKey: dbmodel.NullString(route.CertKey),
		CertType: route.CertType, AcmeChallenge: route.AcmeChallenge, UpdatedAt: time.Now().UTC(), ID: route.Id,
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

func routeCommon(id string, projectId sql.NullString, name, protocol, domain, pathPrefix, targetUrl string, listenPort sql.NullInt64, serviceId, componentName, endpointProtocol sql.NullString, endpointContainerPort sql.NullInt64, enabled, httpsEnabled int64, certPem, certKey sql.NullString, certType, acmeChallenge string, createdAt, updatedAt time.Time) model.Route {
	return model.Route{
		Id: id, ProjectId: dbmodel.StringPtr(projectId), Name: name, Protocol: protocol, Domain: domain, PathPrefix: pathPrefix,
		TargetUrl: targetUrl, ListenPort: dbmodel.IntPtrFromNullInt64(listenPort), ServiceId: dbmodel.StringPtr(serviceId), ComponentName: dbmodel.StringPtr(componentName), EndpointProtocol: dbmodel.StringPtr(endpointProtocol), EndpointContainerPort: dbmodel.IntPtrFromNullInt64(endpointContainerPort), Enabled: dbmodel.IntBool(enabled), HTTPSEnabled: dbmodel.IntBool(httpsEnabled),
		CertPEM: dbmodel.StringPtr(certPem), CertKey: dbmodel.StringPtr(certKey), CertType: certType, AcmeChallenge: acmeChallenge,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func routeFromRow(row routesqlc.ListRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Protocol, row.Domain, row.PathPrefix, row.TargetUrl, row.ListenPort, row.ServiceID, row.ComponentName, row.EndpointProtocol, row.EndpointContainerPort, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.AcmeChallenge, row.CreatedAt, row.UpdatedAt)
}
func routeFromAll(row routesqlc.ListAllRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Protocol, row.Domain, row.PathPrefix, row.TargetUrl, row.ListenPort, row.ServiceID, row.ComponentName, row.EndpointProtocol, row.EndpointContainerPort, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.AcmeChallenge, row.CreatedAt, row.UpdatedAt)
}
func routeFromEnabled(row routesqlc.ListEnabledRoutesRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Protocol, row.Domain, row.PathPrefix, row.TargetUrl, row.ListenPort, row.ServiceID, row.ComponentName, row.EndpointProtocol, row.EndpointContainerPort, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.AcmeChallenge, row.CreatedAt, row.UpdatedAt)
}
func routeFromById(row routesqlc.RouteByIDRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Protocol, row.Domain, row.PathPrefix, row.TargetUrl, row.ListenPort, row.ServiceID, row.ComponentName, row.EndpointProtocol, row.EndpointContainerPort, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.AcmeChallenge, row.CreatedAt, row.UpdatedAt)
}
func routeFromByDomain(row routesqlc.RouteByDomainRow) model.Route {
	return routeCommon(row.ID, row.ProjectID, row.Name, row.Protocol, row.Domain, row.PathPrefix, row.TargetUrl, row.ListenPort, row.ServiceID, row.ComponentName, row.EndpointProtocol, row.EndpointContainerPort, row.Enabled, row.HttpsEnabled, row.CertPem, row.CertKey, row.CertType, row.AcmeChallenge, row.CreatedAt, row.UpdatedAt)
}
