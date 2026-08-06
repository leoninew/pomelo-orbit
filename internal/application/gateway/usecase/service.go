package gatewaysvc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	gatewaydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/dto"
	gatewayport "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/port"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var gatewayCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

const (
	defaultHTTPEntrypoint = "web"
	defaultGatewayTLSMode = "none"
	entrypointWeb         = "web"
	entrypointWebSecure   = "websecure"
)

type Service struct {
	project         gatewayport.ProjectReader
	application     gatewayport.ApplicationStore
	config          gatewayport.ConfigStore
	service         gatewayport.ServiceReader
	workspace       gatewayport.Workspace
	serviceCommands servicesvc.Service
	deployer        gatewayDeployer
}

type gatewayDeployer interface {
	DeployService(context.Context, string, string, deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error)
	WaitDeployment(context.Context, string, string, *time.Duration) (deploymentdto.DeploymentWaitResult, error)
	ExternalNetworkInspect(context.Context, string) (deploymentdto.RuntimeNetwork, error)
}

func New(
	project gatewayport.ProjectReader,
	application gatewayport.ApplicationStore,
	config gatewayport.ConfigStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	workspace gatewayport.Workspace,
) Service {
	return Service{
		project: project, application: application, config: config,
		service: service, workspace: workspace,
		serviceCommands: servicesvc.New(project, application, service, deployment),
	}
}

// WithDeployer completes the one-way dependency injection for the Gateway
// workflow. Deployment only depends on the gateway port, so this avoids an
// application-package import cycle.
func (s Service) WithDeployer(deployer gatewayDeployer) Service {
	s.deployer = deployer
	return s
}

func (s Service) ListGateways(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[gatewaydto.GatewayView], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, err
	}
	apps, err := s.application.ListApplications(ctx, &projectId, page, perPage, search, status.ApplicationKindGateway)
	if err != nil {
		return repository.Page[gatewaydto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list gateways", err)
	}
	items := make([]gatewaydto.GatewayView, 0, len(apps.Items))
	for _, app := range apps.Items {
		cfg, err := s.config.GatewayConfig(ctx, app.Id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return repository.Page[gatewaydto.GatewayView]{}, apperror.New(apperror.KindInternal, "Gateway config missing for application "+app.Id)
			}
			return repository.Page[gatewaydto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
		}
		items = append(items, gatewaydto.GatewayView{Application: app, Config: cfg})
	}
	return repository.Page[gatewaydto.GatewayView]{Items: items, Total: apps.Total, Page: apps.Page, PerPage: apps.PerPage}, nil
}

func (s Service) CreateGateway(ctx context.Context, userId string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return gatewaydto.GatewayView{}, err
	}

	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	imagePullPolicy := strings.TrimSpace(input.InitialComponentPullPolicy)
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
	policy, err := normalizeGatewayIngressPolicy(input.DefaultEntrypoint, input.TLSMode, true)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	image, err := requireGatewayImage(input.InitialComponentImage)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if _, err := s.application.ApplicationByCode(ctx, code); err == nil {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "Application code already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}

	app := model.Application{
		Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, Kind: status.ApplicationKindGateway,
	}
	cfg := model.GatewayConfig{
		ApplicationId:     app.Id,
		RestApiUrl:        restApiUrl,
		BaseDomain:        baseDomain,
		DefaultEntrypoint: policy.DefaultEntrypoint,
		TLSMode:           policy.TLSMode,
	}
	if err := s.application.CreateApplication(ctx, app); err != nil {
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway application", err)
	}
	if err := s.config.UpsertGatewayConfig(ctx, cfg); err != nil {
		_ = s.application.DeleteApplication(ctx, app.Id)
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway config", err)
	}
	version := model.Version{Id: idutil.NewId(), ApplicationId: app.Id, Label: "managed", Status: status.VersionStatusUnpublished}
	if err := s.application.CreateVersion(ctx, version); err != nil {
		_ = s.application.DeleteApplication(ctx, app.Id)
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway version", err)
	}
	if err := s.application.ReplaceVersionComponents(ctx, version.Id, []model.VersionComponent{{Id: idutil.NewId(), VersionId: version.Id, Name: gatewayManagedComponentName, Image: *image, PullPolicy: imagePullPolicy}}); err != nil {
		_ = s.application.DeleteApplication(ctx, app.Id)
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway component", err)
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg); err != nil {
		_ = s.application.DeleteApplication(ctx, app.Id)
		return gatewaydto.GatewayView{}, err
	}
	return s.GatewayForUser(ctx, userId, app.Id)
}

func (s Service) GatewayForUser(ctx context.Context, userId string, applicationId string) (gatewaydto.GatewayView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway {
		return gatewaydto.GatewayView{}, apperror.New(apperror.KindValidation, "Application is not a gateway")
	}
	cfg, err := s.config.GatewayConfig(ctx, app.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return gatewaydto.GatewayView{}, apperror.New(apperror.KindNotFound, "Gateway config not found")
		}
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	exposures, err := s.listActiveGatewayExposures(ctx, &cfg)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	return gatewaydto.GatewayView{Application: app, Config: cfg, Exposures: exposures}, nil
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
			if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
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
	policy, err := normalizeGatewayIngressPolicy(defaultEntrypoint, tlsMode, false)
	if err != nil {
		return gatewaydto.GatewayView{}, err
	}
	cfg.DefaultEntrypoint = policy.DefaultEntrypoint
	cfg.TLSMode = policy.TLSMode
	if err := s.config.UpsertGatewayConfig(ctx, cfg); err != nil {
		return gatewaydto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway config", err)
	}
	return s.GatewayForUser(ctx, userId, applicationId)
}

func (s Service) DeleteGateway(ctx context.Context, userId string, applicationId string, removeDir bool) error {
	view, err := s.GatewayForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	services, err := s.service.ListServicesByApplication(ctx, view.Application.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load services", err)
	}
	for _, service := range services {
		if service.Status == status.ServiceStatusDeploying {
			return apperror.New(apperror.KindValidation, "应用正在部署中, 请稍后再试")
		}
		if service.Status == status.ServiceStatusRunning {
			return apperror.New(apperror.KindValidation, "应用正在运行中, 请先停止后再删除")
		}
	}
	if removeDir {
		if s.workspace == nil {
			return apperror.New(apperror.KindInternal, "gateway workspace is not available")
		}
		if err := s.workspace.RemoveAppDir(view.Application.Code); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to remove application directory", err)
		}
	}
	if err := s.application.DeleteApplication(ctx, view.Application.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application", err)
	}
	return nil
}

// ResolveActiveGatewayConfig returns the active gateway or a user-facing validation error.
func (s Service) ResolveActiveGatewayConfig(ctx context.Context) (*model.GatewayConfig, error) {
	cfg, err := s.config.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindValidation, "no gateway configured: create and configure a gateway first")
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	return &cfg, nil
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

func (s Service) ensureApplicationNameAvailable(ctx context.Context, name string) error {
	existing, err := s.application.ApplicationByName(ctx, name)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application '"+existing.Name+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application name", err)
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

func normalizeGatewayIngressPolicy(defaultEntrypoint *string, tlsMode *string, applyCreateDefaults bool) (gatewayIngressPolicyValues, error) {
	out := gatewayIngressPolicyValues{}
	if defaultEntrypoint != nil {
		out.DefaultEntrypoint = strings.TrimSpace(*defaultEntrypoint)
	}
	if applyCreateDefaults && out.DefaultEntrypoint == "" {
		out.DefaultEntrypoint = defaultHTTPEntrypoint
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
		out.TLSMode = defaultGatewayTLSMode
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

func buildGatewayExposureItem(app model.Application, component model.EffectiveServiceComponent, endpoint model.VersionComponentEndpoint, gateway *model.GatewayConfig) (gatewaydto.GatewayExposureItem, error) {
	internal := strings.ToLower(strings.TrimSpace(app.Code) + "-" + strings.TrimSpace(component.Name))
	listen := endpoint.ContainerPort
	if endpoint.ListenPort != nil {
		listen = *endpoint.ListenPort
	}
	access := "local"
	if endpoint.Mode == "gateway_http" || endpoint.Mode == "gateway_tcp" {
		access = "public"
	}
	publicHost := ""
	clientHint := ""
	switch access {
	case "local":
		clientHint = fmt.Sprintf("127.0.0.1:%d", listen)
	case "public":
		if gateway != nil {
			publicHost = strings.TrimSpace(app.Code) + "." + strings.TrimSpace(gateway.BaseDomain)
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

func (s Service) listActiveGatewayExposures(ctx context.Context, gateway *model.GatewayConfig) ([]gatewaydto.GatewayExposureItem, error) {
	apps, err := s.application.ListApplications(ctx, nil, 1, 10000, "", status.ApplicationKindStandard)
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
					item, err := buildGatewayExposureItem(app, component, endpoint, gateway)
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
				if endpoint.Name != component.Endpoints[index].Name {
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
	case status.ServiceStatusRunning, status.ServiceStatusDeploying:
		return true
	default:
		return false
	}
}
