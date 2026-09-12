package gatewaysvc

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var gatewayCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const (
	// Product identity for the managed gateway — not process configuration.
	managedGatewayCode = "traefik"
	managedGatewayName = "Traefik"

	entrypointWeb       = "web"
	entrypointWebSecure = "websecure"
)

type Service struct {
	project         gatewayport.ProjectReader
	environment     gatewayport.EnvironmentStore
	application     gatewayport.ApplicationStore
	config          gatewayport.ConfigStore
	service         gatewayport.ServiceReader
	route           repository.RouteStore
	serviceCommands servicesvc.Service
	transaction     gatewayport.TransactionRunner
}

func New(
	project gatewayport.ProjectReader,
	environment gatewayport.EnvironmentStore,
	application gatewayport.ApplicationStore,
	configStore gatewayport.ConfigStore,
	service repository.ServiceStore,
	route repository.RouteStore,
	deployment repository.DeploymentStore,
	resolvePath gatewayport.PhysicalPathResolver,
	transaction gatewayport.TransactionRunner,
) Service {
	return Service{
		project: project, environment: environment, application: application, config: configStore,
		service:         service,
		route:           route,
		serviceCommands: servicesvc.New(project, application, service, deployment),
		transaction:     transaction,
	}
}

func ManagedGatewayCode() string {
	return managedGatewayCode
}

func ManagedGatewayName() string {
	return managedGatewayName
}

func (s Service) ListGateways(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[gatewaydto.GatewayView], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, err
	}
	if s.environment == nil {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.New(apperror.KindInternal, "gateway environment store is not configured")
	}
	environment, err := s.environment.EnvironmentByProject(ctx, projectId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Page: max(page, 1), PerPage: max(perPage, 1)}, nil
		}
		return repository.Page[gatewaydto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	if environment.GatewayApplicationId == nil {
		return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Page: max(page, 1), PerPage: max(perPage, 1)}, nil
	}
	app, err := s.application.Application(ctx, *environment.GatewayApplicationId)
	if err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to load bound gateway application", err)
	}
	if app.ProjectId == nil || *app.ProjectId != projectId {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.New(apperror.KindInternal, "Gateway environment binding belongs to another project")
	}
	search = strings.ToLower(strings.TrimSpace(search))
	if search != "" && !strings.Contains(strings.ToLower(app.Name), search) && !strings.Contains(strings.ToLower(app.Code), search) {
		return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Page: max(page, 1), PerPage: max(perPage, 1)}, nil
	}
	cfg, err := s.config.GatewayConfig(ctx, app.Id)
	if err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	view, err := s.gatewayView(ctx, app, cfg, false)
	if err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, err
	}
	page = max(page, 1)
	perPage = max(perPage, 1)
	if page > 1 {
		return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Total: 1, Page: page, PerPage: perPage}, nil
	}
	return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{view}, Total: 1, Page: page, PerPage: perPage}, nil
}

func (s Service) CreateGateway(ctx context.Context, userId string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if s.environment == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "gateway environment store is not configured")
	}

	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	imagePullPolicy := model.GatewayInitialPullPolicy()
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !gatewayCreateCodePattern.MatchString(code) {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "Invalid gateway fields")
	}
	if !validImagePullPolicy(imagePullPolicy) {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "initial_component_pull_policy must be always, missing, or never")
	}
	restApiUrl, err := normalizeRestApiUrl(input.RestApiUrl)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	baseDomain, err := normalizeBaseDomain(input.BaseDomain)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	policy, err := parseGatewayIngressPolicy(input.DefaultEntrypoint, input.TLSMode)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	componentName := model.GatewayComponentName()
	restReadyTimeoutSeconds, err := normalizeRestReadyTimeoutSeconds(input.RestReadyTimeoutSeconds)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	acmeProfile, acmeEmail, dnsApiToken, err := normalizeGatewayCertificateConfig(input.AcmeProfile, input.AcmeEmail, input.DNSApiToken, policy.TLSMode)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	image, err := requireGatewayImage(input.InitialComponentImage)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, projectId, name); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, projectId, code); err != nil {
		return gatewaydto.GatewayView{}, err
	}

	app := model.Application{
		Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: status.ApplicationKindStandard,
	}
	cfg := model.GatewayConfig{
		ApplicationId:           app.Id,
		RestApiUrl:              restApiUrl,
		RestReadyTimeoutSeconds: restReadyTimeoutSeconds,
		BaseDomain:              baseDomain,
		DefaultEntrypoint:       policy.DefaultEntrypoint,
		TLSMode:                 policy.TLSMode,
		AcmeProfile:             acmeProfile,
		AcmeEmail:               acmeEmail,
		DNSApiToken:             dnsApiToken,
	}
	if s.transaction == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "gateway transaction runner is not configured")
	}
	if s.route == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "gateway route store is not configured")
	}
	if err := s.transaction.RunInTransaction(ctx, func(txCtx context.Context) error {
		environment, err := s.environment.EnvironmentByProject(txCtx, projectId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return apperror.New(apperror.KindValidation, "Project environment is required before provisioning a gateway")
			}
			return apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
		}
		if !environment.IsActive() {
			return apperror.New(apperror.KindValidation, "Project environment must be active before provisioning a gateway")
		}
		if !environment.HasFreshSuccessfulProbe() {
			return apperror.New(apperror.KindValidation, "Project environment must pass probe before provisioning a gateway")
		}
		if environment.GatewayApplicationId != nil {
			return apperror.New(apperror.KindConflict, "Project environment already has a gateway")
		}
		networkName := model.GatewayNetworkName()
		versions := buildInitialGatewayVersions(app.Id, *image, imagePullPolicy, componentName, networkName)
		cfg.VersionBindings = make([]model.GatewayVersionBinding, 0, len(versions))
		for _, item := range versions {
			cfg.VersionBindings = append(cfg.VersionBindings, model.GatewayVersionBinding{Profile: item.Role, VersionId: item.Version.Id})
		}
		if err := s.application.CreateApplication(txCtx, app); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to create gateway application", err)
		}
		if err := s.config.UpsertGatewayConfig(txCtx, cfg); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to create gateway config", err)
		}
		for _, item := range versions {
			if err := s.application.CreateVersion(txCtx, item.Version); err != nil {
				return apperror.Wrap(apperror.KindInternal, "Failed to create gateway version", err)
			}
			if err := s.application.ReplaceVersionComponents(txCtx, item.Version.Id, []model.VersionComponent{item.Component}); err != nil {
				return apperror.Wrap(apperror.KindInternal, "Failed to create gateway component", err)
			}
		}
		if err := s.config.ReplaceGatewayVersionBindings(txCtx, app.Id, cfg.VersionBindings); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to bind gateway versions", err)
		}
		if _, err := s.serviceCommands.CreateService(txCtx, userId, servicedto.ServiceCreateInput{
			ApplicationId: app.Id,
			VersionId:     cfg.VersionIDForProfile(gatewayVersionRoleBase),
			InstanceKey:   "default",
			Code:          app.Code + "-default",
		}); err != nil {
			return err
		}
		if err := s.route.CreateRoute(txCtx, buildInitialGatewayDashboardRoute(app, cfg)); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to create gateway dashboard route", err)
		}
		bound, err := s.environment.BindGatewayApplication(txCtx, environment.Id, app.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to bind gateway to project environment", err)
		}
		if !bound {
			return apperror.New(apperror.KindConflict, "Project environment already has a gateway")
		}
		return nil
	}); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	return s.GatewayForUser(ctx, userId, app.Id)
}

func (s Service) GatewayForUser(ctx context.Context, userId string, applicationId string) (gatewaydto.GatewayView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if err := s.ensureGatewayEnvironmentBinding(ctx, app); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	cfg, err := s.config.GatewayConfig(ctx, app.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return gatewaydto.GatewayView{}, apperror.New(apperror.KindNotFound, "Gateway config not found")
		}
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	return s.gatewayView(ctx, app, cfg, true)
}

func (s Service) ensureGatewayEnvironmentBinding(ctx context.Context, app model.Application) error {
	if s.environment == nil {
		return apperror.New(apperror.KindInternal, "gateway environment store is not configured")
	}
	if app.ProjectId == nil || strings.TrimSpace(*app.ProjectId) == "" {
		return apperror.New(apperror.KindValidation, "Gateway application must belong to a project")
	}
	environment, err := s.environment.EnvironmentByProject(ctx, *app.ProjectId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Gateway environment binding not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	if environment.GatewayApplicationId == nil || *environment.GatewayApplicationId != app.Id {
		return apperror.New(apperror.KindNotFound, "Gateway environment binding not found")
	}
	return nil
}

func (s Service) gatewayView(ctx context.Context, app model.Application, cfg model.GatewayConfig, includeExposures bool) (gatewaydto.GatewayView, error) {
	services, err := s.service.ListServicesByApplication(ctx, app.Id)
	if err != nil {
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway services", err)
	}
	var defaultService *model.Service
	for index := range services {
		if services[index].InstanceKey != "default" {
			continue
		}
		if defaultService != nil {
			return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "Gateway has multiple default services")
		}
		defaultService = &services[index]
	}
	if defaultService == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "Gateway default service is missing")
	}
	view := gatewaydto.GatewayView{
		Application:    app,
		Config:         cfg,
		DefaultService: defaultService,
		Services:       services,
	}
	if !includeExposures {
		return view, nil
	}
	if app.ProjectId == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindInternal, "Gateway application has no project")
	}
	exposures, err := s.listActiveGatewayExposures(ctx, *app.ProjectId, &cfg)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	view.Exposures = exposures
	return view, nil
}

func (s Service) UpdateGateway(ctx context.Context, userId string, applicationId string, input gatewaydto.GatewayUpdateInput) (gatewaydto.GatewayView, error) {
	view, err := s.GatewayForUser(ctx, userId, applicationId)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	app := view.Application
	cfg := view.Config
	appDirty := false
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "Invalid gateway name")
		}
		if name != app.Name {
			projectId := ""
			if app.ProjectId != nil {
				projectId = *app.ProjectId
			}
			if err := s.ensureApplicationNameAvailable(ctx, projectId, name); err != nil {
				return gatewaydto.GatewayView{}, err
			}
			app.Name = name
			appDirty = true
		}
	}
	if appDirty {
		if err := s.application.UpdateApplication(ctx, app); err != nil {
			return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway application", err)
		}
	}
	if input.RestApiUrl != nil {
		restApiUrl, err := normalizeRestApiUrl(*input.RestApiUrl)
		if err != nil {
			return gatewaydto.GatewayView{}, err
		}
		cfg.RestApiUrl = restApiUrl
	}
	if input.RestReadyTimeoutSeconds != nil {
		timeout, err := normalizeRestReadyTimeoutSeconds(input.RestReadyTimeoutSeconds)
		if err != nil {
			return gatewaydto.GatewayView{}, err
		}
		cfg.RestReadyTimeoutSeconds = timeout
	}
	if input.BaseDomain != nil {
		baseDomain, err := normalizeBaseDomain(*input.BaseDomain)
		if err != nil {
			return gatewaydto.GatewayView{}, err
		}
		cfg.BaseDomain = baseDomain
	}
	defaultEntrypoint := &cfg.DefaultEntrypoint
	if input.DefaultEntrypoint != nil {
		defaultEntrypoint = input.DefaultEntrypoint
	}
	tlsMode := &cfg.TLSMode
	if input.TLSMode != nil {
		tlsMode = input.TLSMode
	}
	policy, err := parseGatewayIngressPolicy(defaultEntrypoint, tlsMode)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	cfg.DefaultEntrypoint = policy.DefaultEntrypoint
	cfg.TLSMode = policy.TLSMode
	if input.AcmeProfile != nil {
		cfg.AcmeProfile = *input.AcmeProfile
	}
	if input.AcmeEmail != nil {
		cfg.AcmeEmail = *input.AcmeEmail
	}
	if input.DNSApiToken != nil {
		cfg.DNSApiToken = *input.DNSApiToken
	}
	acmeProfile, acmeEmail, dnsApiToken, err := normalizeGatewayCertificateConfig(
		&cfg.AcmeProfile,
		&cfg.AcmeEmail,
		&cfg.DNSApiToken,
		cfg.TLSMode,
	)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	cfg.AcmeProfile = acmeProfile
	cfg.AcmeEmail = acmeEmail
	cfg.DNSApiToken = dnsApiToken
	if err := s.config.UpsertGatewayConfig(ctx, cfg); err != nil {
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway config", err)
	}
	return s.GatewayForUser(ctx, userId, applicationId)
}

func (s Service) DeleteGateway(ctx context.Context, userId string, applicationId string) error {
	if _, err := s.GatewayForUser(ctx, userId, applicationId); err != nil {
		return err
	}
	return apperror.New(apperror.KindValidation, "Gateway cannot be deleted after it is bound to a project environment")
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	applicationId = strings.TrimSpace(applicationId)
	app, err := s.application.Application(ctx, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
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

func (s Service) ensureApplicationNameAvailable(ctx context.Context, projectId string, name string) error {
	existing, err := s.application.ApplicationByProjectAndName(ctx, projectId, name)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application '"+existing.Name+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
	}
	return nil
}

func (s Service) ensureApplicationCodeAvailable(ctx context.Context, projectId string, code string) error {
	existing, err := s.application.ApplicationByProjectAndCode(ctx, projectId, code)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	return nil
}

func normalizeRestApiUrl(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperror.New(apperror.KindValidation, "rest_api_url is required")
	}
	if len(raw) > 512 {
		return "", apperror.New(apperror.KindValidation, "rest_api_url is too long")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", apperror.New(apperror.KindValidation, "rest_api_url must be an absolute http(s) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", apperror.New(apperror.KindValidation, "rest_api_url must use http or https")
	}
	return strings.TrimRight(raw, "/"), nil
}

func normalizeBaseDomain(raw string) (string, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "", apperror.New(apperror.KindValidation, "base_domain is required")
	}
	if len(raw) > 255 {
		return "", apperror.New(apperror.KindValidation, "base_domain is too long")
	}
	if strings.Contains(raw, "://") || strings.Contains(raw, "/") || strings.Contains(raw, " ") {
		return "", apperror.New(apperror.KindValidation, "base_domain must be a bare domain (e.g. example.com)")
	}
	return raw, nil
}

func requireGatewayImage(image *string) (*string, error) {
	if image == nil {
		return nil, apperror.New(apperror.KindValidation, "image is required")
	}
	value := strings.TrimSpace(*image)
	if value == "" {
		return nil, apperror.New(apperror.KindValidation, "image is required")
	}
	if len(value) > 512 {
		return nil, apperror.New(apperror.KindValidation, "image is too long")
	}
	return &value, nil
}

func validImagePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}

type gatewayIngressPolicyValues struct {
	DefaultEntrypoint string
	TLSMode           string
}

// parseGatewayIngressPolicy validates request/persisted GatewayConfig policy fields only.
// Process-level Traefik defaults are not consulted here.
func parseGatewayIngressPolicy(defaultEntrypoint *string, tlsMode *string) (gatewayIngressPolicyValues, error) {
	out := gatewayIngressPolicyValues{}
	if defaultEntrypoint != nil {
		out.DefaultEntrypoint = strings.TrimSpace(*defaultEntrypoint)
	}
	if out.DefaultEntrypoint == "" {
		return gatewayIngressPolicyValues{}, apperror.New(apperror.KindValidation, "default_entrypoint is required")
	}
	if !validGatewayEntrypoint(out.DefaultEntrypoint) {
		return gatewayIngressPolicyValues{}, apperror.New(apperror.KindValidation, "default_entrypoint must be web or websecure")
	}
	if tlsMode != nil {
		out.TLSMode = strings.ToLower(strings.TrimSpace(*tlsMode))
	}
	if out.TLSMode == "" {
		return gatewayIngressPolicyValues{}, apperror.New(apperror.KindValidation, "tls_mode is required")
	}
	if out.TLSMode != "none" && out.TLSMode != "letsencrypt" && out.TLSMode != "tls" {
		return gatewayIngressPolicyValues{}, apperror.New(apperror.KindValidation, "Invalid tls_mode")
	}
	return out, nil
}

func validGatewayEntrypoint(name string) bool {
	switch strings.TrimSpace(name) {
	case entrypointWeb, entrypointWebSecure:
		return true
	default:
		return false
	}
}

func normalizeRestReadyTimeoutSeconds(value *int) (int, error) {
	if value == nil || *value < 1 || *value > 300 {
		return 0, apperror.New(apperror.KindValidation, "rest_ready_timeout_seconds must be between 1 and 300")
	}
	return *value, nil
}

func normalizeGatewayCertificateConfig(profileValue, emailValue, tokenValue *string, tlsMode string) (string, string, string, error) {
	profile := strings.ToLower(strings.TrimSpace(derefString(profileValue)))
	if profile != "" && profile != "http" && profile != "dns" && profile != "http-dns" {
		return "", "", "", apperror.New(apperror.KindValidation, "acme_profile must be http, dns, http-dns, or empty")
	}
	email := strings.TrimSpace(derefString(emailValue))
	token := strings.TrimSpace(derefString(tokenValue))
	if profile != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email {
			return "", "", "", apperror.New(apperror.KindValidation, "acme_email must be a valid email address when an ACME profile is selected")
		}
	}
	if profile == "dns" || profile == "http-dns" {
		if token == "" {
			return "", "", "", apperror.New(apperror.KindValidation, "dns_api_token is required for the selected ACME profile")
		}
		if len(token) > 4096 || strings.ContainsAny(token, "\r\n") {
			return "", "", "", apperror.New(apperror.KindValidation, "dns_api_token is invalid")
		}
	} else {
		token = ""
	}
	if strings.EqualFold(strings.TrimSpace(tlsMode), "letsencrypt") && profile != "http" && profile != "http-dns" {
		return "", "", "", apperror.New(apperror.KindValidation, "acme_profile must support HTTP-01 when tls_mode=letsencrypt")
	}
	return profile, email, token, nil
}

func buildGatewayExposureItem(app model.Application, service model.Service, component model.EffectiveServiceComponent, endpoint model.VersionComponentEndpoint, gateway *model.GatewayConfig) (gatewaydto.GatewayExposureItem, error) {
	internal := strings.ToLower(strings.TrimSpace(app.Code) + "-" + strings.TrimSpace(component.Name))
	listen := endpoint.ContainerPort
	if endpoint.ListenPort != nil {
		listen = *endpoint.ListenPort
	}
	access := "local"
	if endpoint.Mode == "gateway" {
		access = "public"
	}
	publicHost := ""
	clientHint := ""
	switch access {
	case "local":
		clientHint = fmt.Sprintf("127.0.0.1:%d", listen)
	case "public":
		if gateway != nil {
			var err error
			publicHost, err = model.DeriveServiceComponentHost(gateway, service, component.Name)
			if err != nil {
				return gatewaydto.GatewayExposureItem{}, err
			}
		}
		switch endpoint.Protocol {
		case "http":
			if publicHost != "" {
				scheme := "http"
				if gateway != nil && (strings.EqualFold(strings.TrimSpace(gateway.TLSMode), "tls") || strings.EqualFold(strings.TrimSpace(gateway.TLSMode), "letsencrypt")) {
					scheme = "https"
				}
				clientHint = scheme + "://" + publicHost
			}
		case "tcp":
			if publicHost != "" {
				clientHint = fmt.Sprintf("%s:%d", publicHost, listen)
			} else {
				clientHint = fmt.Sprintf(":%d", listen)
			}
		}
	}
	if clientHint == "" {
		clientHint = fmt.Sprintf("%s:%d", internal, endpoint.ContainerPort)
	}
	return gatewaydto.GatewayExposureItem{
		ApplicationId: app.Id, ApplicationCode: app.Code, ComponentName: component.Name,
		Protocol: endpoint.Protocol, Access: access, ContainerPort: endpoint.ContainerPort,
		ListenPort: listen, PublicHost: publicHost, InternalDns: internal, ClientHint: clientHint,
	}, nil
}

func (s Service) listActiveGatewayExposures(ctx context.Context, projectID string, gateway *model.GatewayConfig) ([]gatewaydto.GatewayExposureItem, error) {
	apps, err := s.application.ListApplications(ctx, &projectID, 1, 10000, "", status.ApplicationKindStandard)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list applications for exposures", err)
	}
	var items []gatewaydto.GatewayExposureItem
	for _, app := range apps.Items {
		services, err := s.service.ListServicesByApplication(ctx, app.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
		}
		for _, service := range services {
			if !isActiveServiceStatus(service.Status) {
				continue
			}
			declarations, err := s.application.VersionComponentsByVersion(ctx, service.VersionId)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to list version components", err)
			}
			overlays, err := s.service.ServiceComponentsByService(ctx, service.Id)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to list service components", err)
			}
			plan, _, err := buildGatewayEffectivePlan(app, service, declarations, overlays, gateway)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve service endpoints", err)
			}
			for _, component := range plan.Components {
				for _, endpoint := range component.Endpoints {
					if endpoint.Mode == "internal" {
						continue
					}
					item, err := buildGatewayExposureItem(app, service, component, endpoint, gateway)
					if err != nil {
						return nil, err
					}
					items = append(items, item)
				}
			}
		}
	}
	return items, nil
}

func buildGatewayEffectivePlan(app model.Application, service model.Service, declarations []model.VersionComponent, overlays []model.ServiceComponent, gateway *model.GatewayConfig) (model.EffectiveServicePlan, string, error) {
	// Kept local to avoid a dependency from Gateway to the deployment use case.
	bySource := make(map[string]model.ServiceComponent, len(overlays))
	for _, overlay := range overlays {
		bySource[overlay.SourceVersionComponentId] = overlay
	}
	plan := model.EffectiveServicePlan{Application: app, Version: model.Version{Id: service.VersionId}, Service: service, Gateway: gateway}
	for _, declaration := range declarations {
		overlay, ok := bySource[declaration.Id]
		if !ok {
			return plan, "", fmt.Errorf("service component mapping missing for %s", declaration.Name)
		}
		component := model.EffectiveServiceComponent{Name: declaration.Name, Endpoints: append([]model.VersionComponentEndpoint(nil), declaration.Endpoints...)}
		for index := range component.Endpoints {
			for _, endpoint := range overlay.Endpoints {
				if endpoint.Protocol != component.Endpoints[index].Protocol || endpoint.ContainerPort != component.Endpoints[index].ContainerPort {
					continue
				}
				if endpoint.State == model.ServiceComponentOverlayDeleted {
					component.Endpoints[index].Mode, component.Endpoints[index].BindAddress, component.Endpoints[index].ListenPort, component.Endpoints[index].Entrypoint, component.Endpoints[index].PathPrefix = "internal", nil, nil, nil, nil
					continue
				}
				if endpoint.Mode != nil {
					component.Endpoints[index].Mode = *endpoint.Mode
				}
				if endpoint.BindAddress != nil {
					component.Endpoints[index].BindAddress = endpoint.BindAddress
				}
				if endpoint.ListenPort != nil {
					component.Endpoints[index].ListenPort = endpoint.ListenPort
				}
				if endpoint.Entrypoint != nil {
					component.Endpoints[index].Entrypoint = endpoint.Entrypoint
				}
				if endpoint.PathPrefix != nil {
					component.Endpoints[index].PathPrefix = endpoint.PathPrefix
				}
			}
		}
		plan.Components = append(plan.Components, component)
	}
	return plan, "", nil
}

func isActiveServiceStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case status.ServiceStatusRunning:
		return true
	default:
		return false
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
