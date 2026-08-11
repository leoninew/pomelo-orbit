package routesvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	routedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/dto"
	routeport "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/port"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	project              repository.ProjectReader
	route                repository.RouteStore
	gateway              repository.GatewayStore
	cfg                  config.Config
	routePublisher       routeport.RouteConfigPublisher
	certificateGenerator routeport.RouteCertificateGenerator
	traefikRouterClient  routeport.TraefikRouterClient
}

func New(
	project repository.ProjectReader,
	route repository.RouteStore,
	gateway repository.GatewayStore,
	cfg config.Config,
	routePublisher routeport.RouteConfigPublisher,
	certificateGenerator routeport.RouteCertificateGenerator,
	traefikRouterClient routeport.TraefikRouterClient,
) Service {
	return Service{
		project: project, route: route, gateway: gateway, cfg: cfg,
		routePublisher: routePublisher, certificateGenerator: certificateGenerator,
		traefikRouterClient: traefikRouterClient,
	}
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) resolveGatewayForRender(ctx context.Context) (*model.GatewayConfig, error) {
	cfg, err := s.gateway.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindValidation, "no gateway configured: create and configure a gateway first")
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	return &cfg, nil
}

const (
	certTypeManual      = "manual"
	certTypeLetsEncrypt = "letsencrypt"
	certTypeMkcert      = "mkcert"
)

var routeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
var routeTargetUrlPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(?::\d+)?$`)

// ListRoutes returns routes visible to the user.
func (s Service) ListRoutes(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Route]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Route]{}, err
	}
	items, err := s.route.ListRoutes(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Route]{}, apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
	}
	return items, nil
}

// CreateRoute creates a route and persists it.
func (s Service) CreateRoute(ctx context.Context, userId string, projectId string, input routedto.RouteCreateInput) (model.Route, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return model.Route{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Route{}, err
	}
	name, domain, pathPrefix, targetUrl, enabled, err := normalizeRouteInput(input.Name, input.Domain, input.PathPrefix, input.TargetUrl, input.Enabled)
	if err != nil {
		return model.Route{}, err
	}
	route := model.Route{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Domain: domain, PathPrefix: pathPrefix, TargetUrl: targetUrl, Enabled: enabled, HTTPSEnabled: false, CertType: certTypeManual}
	if err := s.route.CreateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to create route", err)
	}
	created, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
		return model.Route{}, err
	}
	return created, nil
}

// RouteForUser loads a route visible to the current user.
func (s Service) RouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	return s.loadRouteForUser(ctx, userId, routeId)
}

// UpdateRoute updates a route and synchronizes its files.
func (s Service) UpdateRoute(ctx context.Context, userId string, routeId string, input routedto.RouteUpdateInput) (model.Route, error) {
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
	if input.TargetUrl != nil {
		route.TargetUrl = strings.TrimSpace(*input.TargetUrl)
	}
	if input.Enabled != nil {
		route.Enabled = *input.Enabled
	}
	if !validRouteFields(route.Name, route.Domain, route.PathPrefix, route.TargetUrl) {
		return model.Route{}, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if oldName != updated.Name {
		if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
			return model.Route{}, err
		}
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
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
	if err := s.route.DeleteRoute(ctx, route.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete route", err)
	}
	if err := s.revokeRouteCertFiles(ctx, route.Name); err != nil {
		return err
	}
	return s.publishRouteSnapshot(ctx)
}

// EnableRoute marks a route enabled and deploys its config.
func (s Service) EnableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = true
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable route", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
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
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// SyncRoutes republishes the platform rest snapshot (all enabled routes).
func (s Service) SyncRoutes(ctx context.Context, userId string, projectId string) error {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return err
	}
	return s.publishRouteSnapshot(ctx)
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
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route certificate", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
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
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route HTTPS", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
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
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable Let's Encrypt", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
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
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable mkcert", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.publishRouteSnapshot(ctx); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// TraefikRouteConfig reports the dashboard route state.
func (s Service) TraefikRouteConfig(ctx context.Context, userId string, projectId string) (routedto.TraefikConfigView, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return routedto.TraefikConfigView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return routedto.TraefikConfigView{}, err
	}
	gw, err := s.resolveGatewayForRender(ctx)
	if err != nil {
		return routedto.TraefikConfigView{}, err
	}
	dashboardDomain := fmt.Sprintf("traefik.%s", gw.BaseDomain)
	route, err := s.route.RouteByDomain(ctx, dashboardDomain)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return routedto.TraefikConfigView{DashboardDomain: dashboardDomain, HTTPSEnabled: false}, nil
		}
		return routedto.TraefikConfigView{}, apperror.Wrap(apperror.KindInternal, "Failed to load Traefik route config", err)
	}
	return routedto.TraefikConfigView{DashboardDomain: dashboardDomain, HTTPSEnabled: route.Enabled && route.HTTPSEnabled}, nil
}

// ListTraefikRoutes fetches routers from the Traefik API.
func (s Service) ListTraefikRoutes(ctx context.Context, userId string, projectId string) ([]routeport.TraefikRouter, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return nil, err
	}
	gw, err := s.resolveGatewayForRender(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.traefikRouterClient.ListRouters(ctx, gw.RestApiUrl)
	if err != nil {
		if s.traefikRouterClient.IsConnectionError(err) {
			return nil, apperror.Wrap(apperror.KindUnavailable, "Traefik is unavailable.", err)
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list Traefik routes", err)
	}
	out := make([]routeport.TraefikRouter, 0, len(items))
	for _, item := range items {
		out = append(out, routeport.TraefikRouter{
			Name:        item.Name,
			Provider:    item.Provider,
			Status:      item.Status,
			Rule:        item.Rule,
			Service:     item.Service,
			Entrypoints: append([]string(nil), item.Entrypoints...),
			TLS:         item.TLS,
		})
	}
	return out, nil
}

func (s Service) loadRouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	routeId = strings.TrimSpace(routeId)
	route, err := s.route.Route(ctx, routeId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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

// publishRouteSnapshot rebuilds the full platform rest config from all enabled routes.
func (s Service) publishRouteSnapshot(ctx context.Context) error {
	gw, err := s.resolveGatewayForRender(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(gw.RestApiUrl) == "" {
		return apperror.New(apperror.KindValidation, "gateway rest_api_url is required for route publish")
	}
	routes, err := s.route.ListEnabledRoutes(ctx)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list enabled routes", err)
	}
	if err := s.routePublisher.ApplySnapshot(ctx, gw.RestApiUrl, routes); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to publish traefik rest snapshot", err)
	}
	return nil
}

func (s Service) revokeRouteCertFiles(ctx context.Context, routeName string) error {
	return s.routePublisher.RevokeCertificate(ctx, routeName)
}

func normalizeRouteInput(name string, domain string, pathPrefix string, targetUrl string, enabled bool) (string, string, string, string, bool, error) {
	name = strings.TrimSpace(name)
	domain = strings.TrimSpace(domain)
	pathPrefix = strings.TrimSpace(pathPrefix)
	targetUrl = strings.TrimSpace(targetUrl)
	if pathPrefix == "" {
		pathPrefix = "/"
	}
	if !validRouteFields(name, domain, pathPrefix, targetUrl) {
		return "", "", "", "", false, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	return name, domain, pathPrefix, targetUrl, enabled, nil
}

func validRouteFields(name string, domain string, pathPrefix string, targetUrl string) bool {
	return routeNamePattern.MatchString(name) && domain != "" && strings.HasPrefix(pathPrefix, "/") && routeTargetUrlPattern.MatchString(targetUrl)
}
