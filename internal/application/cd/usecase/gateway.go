package cdsvc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListGateways(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[cdto.GatewayView], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[cdto.GatewayView]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[cdto.GatewayView]{}, err
	}
	apps, err := s.store.ListApplications(ctx, &projectId, page, perPage, search, status.ApplicationKindGateway)
	if err != nil {
		return repository.Page[cdto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to list gateways", err)
	}
	items := make([]cdto.GatewayView, 0, len(apps.Items))
	for _, app := range apps.Items {
		cfg, err := s.store.GatewayConfig(ctx, app.Id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return repository.Page[cdto.GatewayView]{}, apperror.New(apperror.KindInternal, "Gateway config missing for application "+app.Id)
			}
			return repository.Page[cdto.GatewayView]{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
		}
		items = append(items, cdto.GatewayView{Application: app, Config: cfg})
	}
	return repository.Page[cdto.GatewayView]{Items: items, Total: apps.Total, Page: apps.Page, PerPage: apps.PerPage}, nil
}

func (s Service) CreateGateway(ctx context.Context, userId string, input cdto.GatewayCreateInput) (cdto.GatewayView, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cdto.GatewayView{}, err
	}
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	imagePullPolicy := strings.TrimSpace(input.ImagePullPolicy)
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) {
		return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "Invalid gateway fields")
	}
	if !validImagePullPolicy(imagePullPolicy) {
		return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "image_pull_policy must be always, missing, or never")
	}
	restApiUrl, err := normalizeRestApiUrl(input.RestApiUrl)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	baseDomain, err := normalizeBaseDomain(input.BaseDomain)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	policy, err := normalizeGatewayIngressPolicy(input.DefaultEntrypoint, input.TLSMode, true)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	image, err := requireGatewayImage(input.Image)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return cdto.GatewayView{}, err
	}
	if _, err := s.store.ApplicationByCode(ctx, code); err == nil {
		return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "Application code already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	app := model.Application{
		Id:              idutil.NewId(),
		ProjectId:       &projectId,
		Name:            name,
		Code:            code,
		Kind:            status.ApplicationKindGateway,
		ImagePullPolicy: imagePullPolicy,
	}
	cfg := model.GatewayConfig{
		ApplicationId:     app.Id,
		RestApiUrl:        restApiUrl,
		BaseDomain:        baseDomain,
		Image:             image,
		DefaultEntrypoint: policy.DefaultEntrypoint,
		TLSMode:           policy.TLSMode,
	}
	if err := s.store.CreateApplicationWithGatewayConfig(ctx, app, cfg); err != nil {
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway", err)
	}
	tcpListens, err := s.collectActivePublicTCPListens(ctx, "")
	if err != nil {
		_ = s.store.DeleteApplication(ctx, app.Id)
		return cdto.GatewayView{}, err
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg, tcpListens...); err != nil {
		_ = s.store.DeleteApplication(ctx, app.Id)
		return cdto.GatewayView{}, err
	}
	return s.GatewayForUser(ctx, userId, app.Id)
}

func (s Service) GatewayForUser(ctx context.Context, userId string, applicationId string) (cdto.GatewayView, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway {
		return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "Application is not a gateway")
	}
	cfg, err := s.store.GatewayConfig(ctx, app.Id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return cdto.GatewayView{}, apperror.New(apperror.KindNotFound, "Gateway config not found")
		}
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	exposures, err := s.listActiveGatewayExposures(ctx, &cfg)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	return cdto.GatewayView{Application: app, Config: cfg, Exposures: exposures}, nil
}

func (s Service) UpdateGateway(ctx context.Context, userId string, applicationId string, input cdto.GatewayUpdateInput) (cdto.GatewayView, error) {
	view, err := s.GatewayForUser(ctx, userId, applicationId)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	app := view.Application
	cfg := view.Config
	appDirty := false
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "Invalid gateway name")
		}
		if name != app.Name {
			if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
				return cdto.GatewayView{}, err
			}
			app.Name = name
			appDirty = true
		}
	}
	if input.ImagePullPolicy != nil {
		policy := strings.TrimSpace(*input.ImagePullPolicy)
		if !validImagePullPolicy(policy) {
			return cdto.GatewayView{}, apperror.New(apperror.KindValidation, "image_pull_policy must be always, missing, or never")
		}
		if policy != app.ImagePullPolicy {
			app.ImagePullPolicy = policy
			appDirty = true
		}
	}
	if appDirty {
		if err := s.store.UpdateApplication(ctx, app); err != nil {
			return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway application", err)
		}
	}
	if input.RestApiUrl != nil {
		restApiUrl, err := normalizeRestApiUrl(*input.RestApiUrl)
		if err != nil {
			return cdto.GatewayView{}, err
		}
		cfg.RestApiUrl = restApiUrl
	}
	if input.BaseDomain != nil {
		baseDomain, err := normalizeBaseDomain(*input.BaseDomain)
		if err != nil {
			return cdto.GatewayView{}, err
		}
		cfg.BaseDomain = baseDomain
	}
	if input.Image != nil {
		image, err := requireGatewayImage(input.Image)
		if err != nil {
			return cdto.GatewayView{}, err
		}
		cfg.Image = image
	}
	if _, err := requireGatewayImage(cfg.Image); err != nil {
		return cdto.GatewayView{}, err
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
		return cdto.GatewayView{}, err
	}
	cfg.DefaultEntrypoint = policy.DefaultEntrypoint
	cfg.TLSMode = policy.TLSMode
	if err := s.store.UpsertGatewayConfig(ctx, cfg); err != nil {
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway config", err)
	}
	tcpListens, err := s.collectActivePublicTCPListens(ctx, "")
	if err != nil {
		return cdto.GatewayView{}, err
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg, tcpListens...); err != nil {
		return cdto.GatewayView{}, err
	}
	return s.GatewayForUser(ctx, userId, applicationId)
}

func (s Service) DeleteGateway(ctx context.Context, userId string, applicationId string, removeDir bool) error {
	view, err := s.GatewayForUser(ctx, userId, applicationId)
	if err != nil {
		return err
	}
	return s.DeleteApplication(ctx, userId, view.Application.Id, cdto.ApplicationDeleteInput{RemoveDir: removeDir})
}

// resolveGatewayForRender returns the active gateway config for compose Host / rest publish.
func (s Service) resolveGatewayForRender(ctx context.Context) (*model.GatewayConfig, error) {
	reader := s.gatewayConfigReader()
	if reader == nil {
		return nil, apperror.New(apperror.KindInternal, "gateway config store is not available")
	}
	cfg, err := reader.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindValidation, "no gateway configured: create and configure a gateway first")
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	return &cfg, nil
}

type gatewayConfigReader interface {
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
}

func (s Service) gatewayConfigReader() gatewayConfigReader {
	if s.store != nil {
		return s.store
	}
	if reader, ok := any(s.executionStore).(gatewayConfigReader); ok && s.executionStore != nil {
		return reader
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
	v := strings.TrimSpace(*image)
	if v == "" {
		return nil, apperror.New(apperror.KindValidation, "image is required")
	}
	if len(v) > 512 {
		return nil, apperror.New(apperror.KindValidation, "image is too long")
	}
	return &v, nil
}

const (
	defaultHTTPEntrypoint = "web"
	defaultGatewayTLSMode = "none"
	entrypointWeb         = "web"
	entrypointWebSecure   = "websecure"
)

type gatewayIngressPolicyValues struct {
	DefaultEntrypoint string
	TLSMode           string
}

func normalizeGatewayIngressPolicy(
	defaultEntrypoint *string,
	tlsMode *string,
	applyCreateDefaults bool,
) (gatewayIngressPolicyValues, error) {
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

// collectActivePublicTCPListens unions public TCP listen ports from active standard services.
// excludeApplicationId skips that app (used when replacing self during deploy).
func (s Service) collectActivePublicTCPListens(ctx context.Context, excludeApplicationId string) ([]int, error) {
	if s.store == nil {
		return nil, nil
	}
	occupancy, err := s.buildExposeOccupancy(ctx, excludeApplicationId)
	if err != nil {
		return nil, err
	}
	return occupancy.PublicTCPListens(), nil
}

func (s Service) listActiveGatewayExposures(ctx context.Context, gateway *model.GatewayConfig) ([]cdto.GatewayExposureItem, error) {
	if s.store == nil {
		return nil, nil
	}
	// List all non-gateway applications and their active services.
	apps, err := s.store.ListApplications(ctx, nil, 1, 10000, "", status.ApplicationKindStandard)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list applications for exposures", err)
	}
	var items []cdto.GatewayExposureItem
	for _, app := range apps.Items {
		services, err := s.store.ListServicesByApplication(ctx, app.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
		}
		for _, svc := range services {
			if !isActiveServiceStatus(svc.Status) {
				continue
			}
			exposes, err := s.store.VersionExposesByVersion(ctx, svc.VersionId)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindInternal, "Failed to list exposes", err)
			}
			for _, expose := range exposes {
				item, err := buildGatewayExposureItem(app, expose, gateway)
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func buildGatewayExposureItem(app model.Application, expose model.VersionExpose, gateway *model.GatewayConfig) (cdto.GatewayExposureItem, error) {
	internal, err := runtimeName(app.Code, expose.ComponentName)
	if err != nil {
		return cdto.GatewayExposureItem{}, apperror.New(apperror.KindValidation, err.Error())
	}
	listen := effectiveListen(expose)
	access := exposeAccessOf(expose)
	publicHost := ""
	clientHint := ""
	switch access {
	case exposeAccessLocal:
		clientHint = fmt.Sprintf("127.0.0.1:%d", listen)
	case exposeAccessPublic:
		if gateway != nil {
			if host, err := deriveHost(gateway, app.Code); err == nil {
				publicHost = host
			}
		}
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		switch protocol {
		case "http":
			if publicHost != "" {
				scheme := "http"
				if gateway != nil {
					tlsMode := strings.ToLower(strings.TrimSpace(gateway.TLSMode))
					if tlsMode == "tls" || tlsMode == "letsencrypt" {
						scheme = "https"
					}
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
	// Always surface cluster DNS for every expose row.
	if clientHint == "" {
		clientHint = fmt.Sprintf("%s:%d", internal, expose.ContainerPort)
	}
	return cdto.GatewayExposureItem{
		ApplicationId:   app.Id,
		ApplicationCode: app.Code,
		ComponentName:   expose.ComponentName,
		Protocol:        expose.Protocol,
		Access:          access,
		ContainerPort:   expose.ContainerPort,
		ListenPort:      listen,
		PublicHost:      publicHost,
		InternalDns:     internal,
		ClientHint:      clientHint,
	}, nil
}

func isActiveServiceStatus(st string) bool {
	switch strings.TrimSpace(st) {
	case status.ServiceStatusRunning, status.ServiceStatusDeploying:
		return true
	default:
		return false
	}
}
