package cdsvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"database/sql"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

const (
	certTypeManual      = "manual"
	certTypeLetsEncrypt = "letsencrypt"
	certTypeMkcert      = "mkcert"
)

var routeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
var routeTargetURLPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+:\d+$`)

// RouteCreateInput creates a project route.
// RouteUpdateInput updates a project route.
// TraefikConfigResp reports the dashboard route state.
// TraefikRouteListResp wraps the Traefik router list.
// ListRoutes returns routes visible to the user.
func (s Service) ListRoutes(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Route]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Route]{}, err
	}
	items, err := s.store.ListRoutes(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Route]{}, apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
	}
	return items, nil
}

// CreateRoute creates a route and persists it.
func (s Service) CreateRoute(ctx context.Context, userId string, projectId string, input cdto.RouteCreateInput) (model.Route, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return model.Route{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Route{}, err
	}
	name, domain, pathPrefix, targetURL, enabled, err := normalizeRouteInput(input.Name, input.Domain, input.PathPrefix, input.TargetURL, input.Enabled)
	if err != nil {
		return model.Route{}, err
	}
	route := model.Route{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Domain: domain, PathPrefix: pathPrefix, TargetURL: targetURL, Enabled: enabled, HTTPSEnabled: false, CertType: certTypeManual}
	if err := s.store.CreateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to create route", err)
	}
	created, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if created.Enabled {
		if err := s.syncRouteFiles(ctx, created); err != nil {
			return model.Route{}, err
		}
	}
	return created, nil
}

// RouteForUser loads a route visible to the current user.
func (s Service) RouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	return s.loadRouteForUser(ctx, userId, routeId)
}

// UpdateRoute updates a route and synchronizes its files.
func (s Service) UpdateRoute(ctx context.Context, userId string, routeId string, input cdto.RouteUpdateInput) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	oldName := route.Name
	if input.Name != nil {
		route.Name = strings.TrimSpace(*input.Name)
	}
	if input.Domain != nil {
		route.Domain = strings.TrimSpace(*input.Domain)
	}
	if input.PathPrefix != nil {
		route.PathPrefix = strings.TrimSpace(*input.PathPrefix)
	}
	if input.TargetURL != nil {
		route.TargetURL = strings.TrimSpace(*input.TargetURL)
	}
	if input.Enabled != nil {
		route.Enabled = *input.Enabled
	}
	if !validRouteFields(route.Name, route.Domain, route.PathPrefix, route.TargetURL) {
		return model.Route{}, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if oldName != updated.Name {
		if err := s.revokeRouteFiles(ctx, oldName); err != nil {
			return model.Route{}, err
		}
		if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
			return model.Route{}, err
		}
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DeleteRoute removes a route after confirming it is disabled.
func (s Service) DeleteRoute(ctx context.Context, userId string, routeId string) error {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return err
	}
	if route.Enabled {
		return apperror.New(apperror.KindValidation, "Cannot delete enabled route. Please disable it first.")
	}
	if err := s.store.DeleteRoute(ctx, route.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete route", err)
	}
	if err := s.revokeRouteFiles(ctx, route.Name); err != nil {
		return err
	}
	return nil
}

// EnableRoute marks a route enabled and deploys its config.
func (s Service) EnableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = true
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable route", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DisableRoute marks a route disabled and removes its config.
func (s Service) DisableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = false
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route", err)
	}
	if err := s.revokeRouteFiles(ctx, route.Name); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

// SyncRoutes synchronizes every route in a project.
func (s Service) SyncRoutes(ctx context.Context, userId string, projectId string) error {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return err
	}
	routes, err := s.store.ListAllRoutes(ctx, projectId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
	}
	for _, route := range routes {
		if err := s.syncRouteFiles(ctx, route); err != nil {
			return err
		}
	}
	return nil
}

// UploadRouteCert stores a manual certificate and updates the route.
func (s Service) UploadRouteCert(ctx context.Context, userId string, routeId string, certPEM string, certKey string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &certKey
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route certificate", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DisableRouteHTTPS clears HTTPS settings and removes certificate files.
func (s Service) DisableRouteHTTPS(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	oldName := route.Name
	route.HTTPSEnabled = false
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route HTTPS", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// EnableRouteLetsEncrypt enables Let's Encrypt for the route.
func (s Service) EnableRouteLetsEncrypt(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if !s.cfg.Cert.LetsEncrypt.Enabled {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt not enabled. Please set cert.letsencrypt.enabled=true and configure email in config")
	}
	if strings.TrimSpace(s.cfg.Cert.LetsEncrypt.Email) == "" {
		return model.Route{}, apperror.New(apperror.KindValidation, "请在配置中设置 cert.letsencrypt.email")
	}
	if route.HTTPSEnabled && route.CertType == certTypeLetsEncrypt {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt 证书已启用")
	}
	if route.Domain == "localhost" || strings.HasSuffix(route.Domain, ".local") {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt 不支持内网域名,请使用手动证书或 mkcert")
	}
	oldName := route.Name
	route.HTTPSEnabled = true
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeLetsEncrypt
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable Let's Encrypt", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// EnableRouteMkcert generates a certificate with mkcert.
func (s Service) EnableRouteMkcert(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	certPEM, keyPEM, err := s.certificateGenerator.Generate(ctx, route.Domain)
	if err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &keyPEM
	route.CertType = certTypeMkcert
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable mkcert", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// TraefikRouteConfig reports the dashboard route state.
func (s Service) TraefikRouteConfig(ctx context.Context, userId string, projectId string) (cdto.TraefikConfigResp, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return cdto.TraefikConfigResp{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cdto.TraefikConfigResp{}, err
	}
	dashboardDomain := fmt.Sprintf("traefik.%s", s.cfg.Traefik.DomainSuffix)
	route, err := s.store.RouteByDomain(ctx, dashboardDomain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return cdto.TraefikConfigResp{DashboardDomain: dashboardDomain, HTTPSEnabled: false}, nil
		}
		return cdto.TraefikConfigResp{}, apperror.Wrap(apperror.KindInternal, "Failed to load Traefik route config", err)
	}
	return cdto.TraefikConfigResp{DashboardDomain: dashboardDomain, HTTPSEnabled: route.Enabled && route.HTTPSEnabled}, nil
}

// ListTraefikRoutes fetches routers from the Traefik API.
func (s Service) ListTraefikRoutes(ctx context.Context, userId string, projectId string) (cdto.TraefikRouteListResp, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return cdto.TraefikRouteListResp{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cdto.TraefikRouteListResp{}, err
	}
	items, err := s.traefikRouterClient.ListRouters(ctx)
	if err != nil {
		if s.traefikRouterClient.IsConnectionError(err) {
			return cdto.TraefikRouteListResp{}, apperror.New(apperror.KindValidation, fmt.Sprintf("无法连接到 Traefik: %v", err))
		}
		return cdto.TraefikRouteListResp{}, apperror.Wrap(apperror.KindInternal, "Failed to list Traefik routes", err)
	}
	responses := make([]cdto.TraefikRouterResp, 0, len(items))
	for _, item := range items {
		responses = append(responses, cdto.TraefikRouterResp{Name: item.Name, Provider: item.Provider, Status: item.Status, Rule: item.Rule, Service: item.Service, Entrypoints: append([]string(nil), item.Entrypoints...), TLS: item.TLS})
	}
	return cdto.TraefikRouteListResp{Items: responses, Total: len(responses)}, nil
}

func (s Service) loadRouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	routeId = strings.TrimSpace(routeId)
	route, err := s.store.Route(ctx, routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Route{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
		}
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if route.ProjectId == nil {
		return model.Route{}, apperror.New(apperror.KindForbidden, "Permission denied")
	}
	if err := s.ensureProjectMembership(ctx, *route.ProjectId, userId); err != nil {
		return model.Route{}, err
	}
	return route, nil
}

func (s Service) syncRouteFiles(ctx context.Context, route model.Route) error {
	return s.routePublisher.Sync(ctx, route)
}

func (s Service) revokeRouteFiles(ctx context.Context, routeName string) error {
	return s.routePublisher.Revoke(ctx, routeName)
}

func (s Service) revokeRouteCertFiles(ctx context.Context, routeName string) error {
	return s.routePublisher.RevokeCertificate(ctx, routeName)
}

func normalizeRouteInput(name string, domain string, pathPrefix string, targetURL string, enabled bool) (string, string, string, string, bool, error) {
	name = strings.TrimSpace(name)
	domain = strings.TrimSpace(domain)
	pathPrefix = strings.TrimSpace(pathPrefix)
	targetURL = strings.TrimSpace(targetURL)
	if pathPrefix == "" {
		pathPrefix = "/"
	}
	if !validRouteFields(name, domain, pathPrefix, targetURL) {
		return "", "", "", "", false, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	return name, domain, pathPrefix, targetURL, enabled, nil
}

func validRouteFields(name string, domain string, pathPrefix string, targetURL string) bool {
	return routeNamePattern.MatchString(name) && domain != "" && strings.HasPrefix(pathPrefix, "/") && routeTargetURLPattern.MatchString(targetURL)
}
