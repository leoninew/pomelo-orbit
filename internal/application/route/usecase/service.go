package routesvc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type Service struct {
	project              repository.ProjectReader
	application          repository.ApplicationStore
	service              repository.ServiceStore
	route                repository.RouteStore
	gateway              repository.GatewayStore
	routePublisher       routeport.RouteConfigPublisher
	certificateGenerator routeport.RouteCertificateGenerator
	traefikRouterClient  routeport.TraefikRouterClient
	transactionRunner    routeport.TransactionRunner
}

func New(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	service repository.ServiceStore,
	route repository.RouteStore,
	gateway repository.GatewayStore,
	routePublisher routeport.RouteConfigPublisher,
	certificateGenerator routeport.RouteCertificateGenerator,
	traefikRouterClient routeport.TraefikRouterClient,
	transactionRunner routeport.TransactionRunner,
) Service {
	return Service{
		project: project, application: application, service: service, route: route, gateway: gateway,
		routePublisher: routePublisher, certificateGenerator: certificateGenerator,
		traefikRouterClient: traefikRouterClient, transactionRunner: transactionRunner,
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

func (s Service) resolveGatewayForRoute(ctx context.Context) (*model.GatewayConfig, error) {
	return s.resolveGatewayForRender(ctx)
}

func (s Service) withRouteACMECapabilities(ctx context.Context, route model.Route) model.Route {
	gateway, err := s.resolveGatewayForRoute(ctx)
	if err != nil {
		route.ACMEChallengeHint = "Gateway configuration is required before enabling Let's Encrypt"
		return route
	}
	if route.ProjectId == nil {
		route.ACMEChallengeHint = "Route project is required"
		return route
	}
	gatewayApp, err := s.application.Application(ctx, gateway.ApplicationId)
	if err != nil || gatewayApp.ProjectId == nil || *gatewayApp.ProjectId != *route.ProjectId {
		route.ACMEChallengeHint = "The active Gateway must belong to this project"
		return route
	}
	route.GatewayApplicationId = gateway.ApplicationId
	route.HTTP01Available = gatewaySupportsHTTP01(gateway.AcmeProfile)
	route.DNS01Available = gatewaySupportsDNS01(gateway.AcmeProfile) && strings.TrimSpace(gateway.DNSApiToken) != ""
	switch {
	case !route.HTTP01Available && !gatewaySupportsDNS01(gateway.AcmeProfile):
		route.ACMEChallengeHint = "Select an ACME profile in Gateway settings"
	case gatewaySupportsDNS01(gateway.AcmeProfile) && !route.DNS01Available:
		route.ACMEChallengeHint = "Gateway DNS API token is required for DNS-01"
	}
	return route
}

const (
	certTypeManual      = "manual"
	certTypeLetsEncrypt = "letsencrypt"
	certTypeMkcert      = "mkcert"
	acmeChallengeHTTP   = "http"
	acmeChallengeDNS    = "dns"
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
	route, err := s.routeFromCreateInput(ctx, projectId, input)
	if err != nil {
		return model.Route{}, err
	}
	route.Id = idutil.NewId()
	route.ProjectId = &projectId
	if err := s.route.CreateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to create route", err)
	}
	created, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return created, nil
}

// RouteForUser loads a route visible to the current user.
func (s Service) RouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	return s.withRouteACMECapabilities(ctx, route), nil
}

// UpdateRoute updates the business route record. Traefik is updated by the
// explicit full-sync flow.
func (s Service) UpdateRoute(ctx context.Context, userId string, routeId string, input routedto.RouteUpdateInput) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if input.Name != nil {
		route.Name = strings.TrimSpace(*input.Name)
	}
	if input.Protocol != nil {
		route.Protocol = strings.TrimSpace(*input.Protocol)
		switch route.Protocol {
		case routeProtocolHTTP:
			route.ListenPort, route.ServiceId, route.ComponentName, route.EndpointProtocol, route.EndpointContainerPort = nil, nil, nil, nil, nil
		case routeProtocolTCP:
			route.PathPrefix, route.TargetUrl = "", ""
			route.HTTPSEnabled, route.CertPEM, route.CertKey, route.CertType, route.AcmeChallenge = false, nil, nil, certTypeManual, acmeChallengeHTTP
		}
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
	if input.ListenPort != nil {
		route.ListenPort = input.ListenPort
	}
	if input.ServiceId != nil {
		route.ServiceId = optionalString(*input.ServiceId)
	}
	if input.ComponentName != nil {
		route.ComponentName = optionalString(*input.ComponentName)
	}
	if input.EndpointProtocol != nil {
		route.EndpointProtocol = optionalString(*input.EndpointProtocol)
		if route.EndpointProtocol == nil {
			route.EndpointContainerPort = nil
		}
	}
	if input.EndpointContainerPort != nil {
		route.EndpointContainerPort = input.EndpointContainerPort
	}
	if route.Protocol == routeProtocolHTTP && route.PathPrefix == "" {
		route.PathPrefix = "/"
	}
	if err := s.validateRoute(ctx, &route, route.Id); err != nil {
		return model.Route{}, err
	}
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
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
	return nil
}

// EnableRoute marks a route enabled in business data. Traefik is updated by
// the explicit full-sync flow.
func (s Service) EnableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = true
	if err := s.validateRoute(ctx, &route, route.Id); err != nil {
		return model.Route{}, err
	}
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable route", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

// DisableRoute marks a route disabled in business data. Traefik is updated by
// the explicit full-sync flow.
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
	return updated, nil
}

// UploadRouteCert stores a manual certificate and updates the route.
func (s Service) UploadRouteCert(ctx context.Context, userId string, routeId string, certPEM string, certKey string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if err := requireHTTPRoute(route); err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &certKey
	route.CertType = certTypeManual
	route.AcmeChallenge = acmeChallengeHTTP
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route certificate", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

// DisableRouteHTTPS clears HTTPS settings. Certificate files are reconciled by
// the next full snapshot publication.
func (s Service) DisableRouteHTTPS(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if err := requireHTTPRoute(route); err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = false
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeManual
	route.AcmeChallenge = acmeChallengeHTTP
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route HTTPS", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

// EnableRouteLetsEncrypt enables Let's Encrypt for the route.
func (s Service) EnableRouteLetsEncrypt(ctx context.Context, userId string, routeId string, challenge string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if err := requireHTTPRoute(route); err != nil {
		return model.Route{}, err
	}
	challenge = strings.ToLower(strings.TrimSpace(challenge))
	if challenge != acmeChallengeHTTP && challenge != acmeChallengeDNS {
		return model.Route{}, apperror.New(apperror.KindValidation, "challenge must be http or dns")
	}
	if route.HTTPSEnabled && route.CertType == certTypeLetsEncrypt {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt 证书已启用")
	}
	if err := validatePublicACMEDomain(route.Domain); err != nil {
		return model.Route{}, err
	}
	if _, err := s.validateGatewayACMECapability(ctx, route, challenge); err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeLetsEncrypt
	route.AcmeChallenge = challenge
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable Let's Encrypt", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

func validatePublicACMEDomain(domain string) error {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "localhost" || strings.HasSuffix(domain, ".local") || net.ParseIP(domain) != nil {
		return apperror.New(apperror.KindValidation, "Let's Encrypt requires a publicly registered DNS name; use a manual certificate or mkcert for local names")
	}
	if _, err := publicsuffix.EffectiveTLDPlusOne(domain); err != nil {
		return apperror.New(apperror.KindValidation, "Let's Encrypt requires a publicly registered DNS name")
	}
	return nil
}

func (s Service) validateGatewayACMECapability(ctx context.Context, route model.Route, challenge string) (*model.GatewayConfig, error) {
	gatewayConfig, err := s.resolveGatewayForRoute(ctx)
	if err != nil {
		return nil, err
	}
	gatewayApp, err := s.application.Application(ctx, gatewayConfig.ApplicationId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load active gateway", err)
	}
	if gatewayApp.ProjectId == nil || route.ProjectId == nil || *gatewayApp.ProjectId != *route.ProjectId {
		return nil, apperror.New(apperror.KindValidation, "active gateway must belong to the Route project")
	}
	switch challenge {
	case acmeChallengeHTTP:
		if !gatewaySupportsHTTP01(gatewayConfig.AcmeProfile) {
			return nil, apperror.New(apperror.KindValidation, "HTTP-01 requires the Gateway http or http-dns ACME profile")
		}
	case acmeChallengeDNS:
		if !gatewaySupportsDNS01(gatewayConfig.AcmeProfile) {
			return nil, apperror.New(apperror.KindValidation, "DNS-01 requires the Gateway dns or http-dns ACME profile")
		}
		if strings.TrimSpace(gatewayConfig.DNSApiToken) == "" {
			return nil, apperror.New(apperror.KindValidation, "Gateway DNS API token is required for DNS-01")
		}
	}
	return gatewayConfig, nil
}

func gatewaySupportsHTTP01(profile string) bool {
	return profile == "http" || profile == "http-dns"
}

func gatewaySupportsDNS01(profile string) bool {
	return profile == "dns" || profile == "http-dns"
}

// EnableRouteMkcert generates a certificate with mkcert.
func (s Service) EnableRouteMkcert(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if err := requireHTTPRoute(route); err != nil {
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
	route.AcmeChallenge = acmeChallengeHTTP
	if err := s.route.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable mkcert", err)
	}
	updated, err := s.route.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
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
	configView := routedto.TraefikConfigView{DashboardDomain: dashboardDomain, HTTPSEnabled: false, BaseDomain: gw.BaseDomain}
	route, err := s.route.RouteByDomain(ctx, dashboardDomain)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return configView, nil
		}
		return routedto.TraefikConfigView{}, apperror.Wrap(apperror.KindInternal, "Failed to load Traefik route config", err)
	}
	configView.HTTPSEnabled = route.Enabled && route.HTTPSEnabled
	return configView, nil
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

// PublishSnapshot is the worker-facing hook used after a Gateway deployment.
// It only pushes the dynamic HTTP/TCP REST snapshot. Static entrypoint compile
// belongs to Route mutations and must not rewrite Version components mid-deploy.
// compose up success is not sufficient: the Traefik REST control plane must
// accept requests before the snapshot PUT.
func (s Service) PublishSnapshot(ctx context.Context) error {
	routes, err := s.listEnabledRoutesForPublish(ctx)
	if err != nil {
		return err
	}
	gateway, err := s.resolveGatewayForRender(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(gateway.RestApiUrl) == "" {
		return apperror.New(apperror.KindValidation, "gateway rest_api_url is required for route publish")
	}
	if err := s.routePublisher.WaitUntilReady(ctx, gateway.RestApiUrl, time.Duration(gateway.RestReadyTimeoutSeconds)*time.Second); err != nil {
		return err
	}
	return s.applyRouteSnapshot(ctx, routes, false)
}

func (s Service) listEnabledRoutesForPublish(ctx context.Context) ([]model.Route, error) {
	routes, err := s.route.ListEnabledRoutes(ctx)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list enabled routes", err)
	}
	return routes, nil
}

func (s Service) applyRouteSnapshot(ctx context.Context, routes []model.Route, waitReady bool) error {
	gw, err := s.resolveGatewayForRender(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(gw.RestApiUrl) == "" {
		return apperror.New(apperror.KindValidation, "gateway rest_api_url is required for route publish")
	}
	if waitReady {
		if err := s.routePublisher.WaitUntilReady(ctx, gw.RestApiUrl, time.Duration(gw.RestReadyTimeoutSeconds)*time.Second); err != nil {
			return err
		}
	}
	if err := s.resolveManagedRouteTargets(ctx, routes); err != nil {
		return err
	}
	if err := s.routePublisher.ApplySnapshot(ctx, *gw, routes); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to publish traefik rest snapshot", err)
	}
	return nil
}

func validRouteIdentity(name string, domain string, pathPrefix string) bool {
	return routeNamePattern.MatchString(name) && domain != "" && strings.HasPrefix(pathPrefix, "/")
}

func requireHTTPRoute(route model.Route) error {
	if route.Protocol != routeProtocolHTTP {
		return apperror.New(apperror.KindValidation, "HTTPS is only available for HTTP routes")
	}
	return nil
}
