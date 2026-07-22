package cdsvc

import (
	"context"
	"errors"
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
	restURL, err := normalizeRestAPIURL(input.RestAPIURL)
	if err != nil {
		return cdto.GatewayView{}, err
	}
	baseDomain, err := normalizeBaseDomain(input.BaseDomain)
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
		ApplicationId: app.Id,
		RestAPIURL:    restURL,
		BaseDomain:    baseDomain,
		Image:         normalizeOptionalText(input.Image),
	}
	if err := s.store.CreateApplicationWithGatewayConfig(ctx, app, cfg); err != nil {
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to create gateway", err)
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg); err != nil {
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
	return cdto.GatewayView{Application: app, Config: cfg}, nil
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
	if input.RestAPIURL != nil {
		restURL, err := normalizeRestAPIURL(*input.RestAPIURL)
		if err != nil {
			return cdto.GatewayView{}, err
		}
		cfg.RestAPIURL = restURL
	}
	if input.BaseDomain != nil {
		baseDomain, err := normalizeBaseDomain(*input.BaseDomain)
		if err != nil {
			return cdto.GatewayView{}, err
		}
		cfg.BaseDomain = baseDomain
	}
	if input.Image != nil {
		cfg.Image = normalizeOptionalText(input.Image)
	}
	if err := s.store.UpsertGatewayConfig(ctx, cfg); err != nil {
		return cdto.GatewayView{}, apperror.Wrap(apperror.KindInternal, "Failed to update gateway config", err)
	}
	if _, err := s.CompileGatewayToVersion(ctx, app, cfg); err != nil {
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

func normalizeRestAPIURL(raw string) (string, error) {
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
