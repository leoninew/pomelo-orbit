package cdsvc

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/status"
	"backend/internal/templatex"
	"gopkg.in/yaml.v3"
)

// ConfigFileInput stores an application config file payload.
type ConfigFileInput struct {
	Path    string
	Content string
}

// ApplicationRouteInput stores an application route payload.
type ApplicationRouteInput struct {
	ServiceName string
	Domain      string
	Port        int
}

// ApplicationServiceConfigImportInput stores import payload for service config.
type ApplicationServiceConfigImportInput struct {
	ServiceName string
	Image       *string
	Environment *string
	Volumes     *string
}

// ApplicationImportInput imports an application bundle.
type ApplicationImportInput struct {
	ProjectId         string
	Version           string
	Name              string
	Code              string
	ImagePullPolicy   string
	RouteManaged      bool
	ConfigFiles       []ConfigFileInput
	ServiceConfigs    []ApplicationServiceConfigImportInput
	ApplicationRoutes []ApplicationRouteInput
}

// ApplicationExport bundles application data for handler response.
type ApplicationExport struct {
	Application    repository.Application
	ConfigFiles    []repository.ApplicationConfigFile
	ServiceConfigs []repository.ApplicationServiceConfig
	Routes         []repository.ApplicationRoute
}

// ApplicationServiceConfigView is the service-config view used by HTTP responses.
type ApplicationServiceConfigView struct {
	ServiceName   string  `json:"service_name"`
	DefaultDomain string  `json:"default_domain"`
	DefaultPort   int     `json:"default_port"`
	BaseImage     *string `json:"base_image"`
	Image         *string `json:"image"`
	ConfigId      *string `json:"config_id"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
}

func (s Service) ImportApplication(ctx context.Context, userId string, input ApplicationImportInput) (repository.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Application{}, err
	}
	name, code, imagePullPolicy, err := normalizeApplicationImportInput(input)
	if err != nil {
		return repository.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return repository.Application{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, code); err != nil {
		return repository.Application{}, err
	}
	app := repository.Application{Id: repository.NewId(), ProjectId: &projectId, Name: name, Code: code, ImagePullPolicy: imagePullPolicy, RouteManaged: input.RouteManaged, Status: repository.ApplicationStatusUndeployed}
	files := make([]repository.ApplicationConfigFile, 0, len(input.ConfigFiles))
	for _, item := range input.ConfigFiles {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			return repository.Application{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
		}
		files = append(files, repository.ApplicationConfigFile{Id: repository.NewId(), ApplicationId: app.Id, Path: path, Content: item.Content})
	}
	serviceConfigs := make([]repository.ApplicationServiceConfig, 0, len(input.ServiceConfigs))
	for _, item := range input.ServiceConfigs {
		serviceName := strings.TrimSpace(item.ServiceName)
		if serviceName == "" {
			return repository.Application{}, apperror.New(apperror.KindValidation, "Invalid application service config fields")
		}
		image := normalizeOptionalText(item.Image)
		environment := normalizeOptionalText(item.Environment)
		volumes := normalizeOptionalText(item.Volumes)
		if image == nil && environment == nil && volumes == nil {
			continue
		}
		serviceConfigs = append(serviceConfigs, repository.ApplicationServiceConfig{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Image: image, Environment: environment, Volumes: volumes})
	}
	routes := make([]repository.ApplicationRoute, 0, len(input.ApplicationRoutes))
	for _, item := range input.ApplicationRoutes {
		serviceName, domain, port, err := normalizeApplicationRouteInput(item.ServiceName, item.Domain, item.Port)
		if err != nil {
			return repository.Application{}, err
		}
		routes = append(routes, repository.ApplicationRoute{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Domain: domain, Port: port})
	}
	if err := s.store.CreateApplicationBundle(ctx, app, files, serviceConfigs, routes); err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to import application", err)
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return repository.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return created, nil
}

func (s Service) ExportApplication(ctx context.Context, userId string, applicationId string) (ApplicationExport, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return ApplicationExport{}, err
	}
	files, err := s.store.ConfigFiles(ctx, app.Id)
	if err != nil {
		return ApplicationExport{}, apperror.Wrap(apperror.KindInternal, "Failed to load application config files", err)
	}
	serviceConfigs, err := s.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return ApplicationExport{}, apperror.Wrap(apperror.KindInternal, "Failed to load application service configs", err)
	}
	routes, err := s.store.Routes(ctx, app.Id)
	if err != nil {
		return ApplicationExport{}, apperror.Wrap(apperror.KindInternal, "Failed to load application routes", err)
	}
	return ApplicationExport{Application: app, ConfigFiles: files, ServiceConfigs: serviceConfigs, Routes: routes}, nil
}

func (s Service) ListApplicationFiles(ctx context.Context, userId string, applicationId string) ([]repository.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	files, err := s.store.ConfigFiles(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list application files", err)
	}
	return files, nil
}

func (s Service) CreateApplicationFile(ctx context.Context, userId string, applicationId string, input ConfigFileInput) (repository.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.ApplicationConfigFile{}, err
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return repository.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
	}
	file := repository.ApplicationConfigFile{Id: repository.NewId(), ApplicationId: app.Id, Path: path, Content: input.Content}
	if err := s.store.CreateConfigFile(ctx, file); err != nil {
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to create application file", err)
	}
	created, err := s.store.ConfigFile(ctx, file.Id)
	if err != nil {
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
	}
	return created, nil
}

func (s Service) ApplicationFileForUser(ctx context.Context, userId string, applicationId string, fileId string) (repository.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.ApplicationConfigFile{}, err
	}
	fileId = strings.TrimSpace(fileId)
	file, err := s.store.ConfigFile(ctx, fileId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ApplicationConfigFile{}, apperror.New(apperror.KindNotFound, "Config file "+fileId+" not found")
		}
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
	}
	if file.ApplicationId != app.Id {
		return repository.ApplicationConfigFile{}, apperror.New(apperror.KindNotFound, "Config file "+fileId+" not found")
	}
	return file, nil
}

func (s Service) UpdateApplicationFile(ctx context.Context, userId string, applicationId string, fileId string, input ConfigFileInput) (repository.ApplicationConfigFile, error) {
	file, err := s.ApplicationFileForUser(ctx, userId, applicationId, fileId)
	if err != nil {
		return repository.ApplicationConfigFile{}, err
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return repository.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
	}
	file.Path = path
	file.Content = input.Content
	if err := s.store.UpdateConfigFile(ctx, file); err != nil {
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to update application file", err)
	}
	updated, err := s.store.ConfigFile(ctx, file.Id)
	if err != nil {
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
	}
	return updated, nil
}

func (s Service) DeleteApplicationFile(ctx context.Context, userId string, applicationId string, fileId string) error {
	file, err := s.ApplicationFileForUser(ctx, userId, applicationId, fileId)
	if err != nil {
		return err
	}
	if err := s.store.DeleteConfigFile(ctx, file.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application file", err)
	}
	return nil
}

func (s Service) StopApplication(ctx context.Context, userId string, applicationId string, removeVolumes bool) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if app.Status != repository.ApplicationStatusDeployed {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法停止")
	}
	deployment := newApplicationDeployment(app, "stop")
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	cmd := []string{"docker", "compose", "-f", "docker-compose.yml", "down"}
	if removeVolumes {
		cmd = append(cmd, "-v")
	}
	appDir := filepath.Join(s.dataRoot, "cd", app.Code)
	if output, err := runApplicationCommand(ctx, appDir, cmd...); err != nil {
		_ = s.store.CompleteDeployment(ctx, deployment.Id, repository.WorkStatusFaulted, outputOrError(output, err))
		return "", apperror.New(apperror.KindInternal, "Failed to stop application")
	}
	if err := s.store.MarkApplicationStatus(ctx, app.Id, repository.ApplicationStatusUndeployed); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to update application status", err)
	}
	if err := s.store.CompleteDeployment(ctx, deployment.Id, repository.WorkStatusRanToCompletion, ""); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to complete deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) RestartApplication(ctx context.Context, userId string, applicationId string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if app.Status != repository.ApplicationStatusDeployed {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法重启")
	}
	if err := s.ensureApplicationComposeFile(ctx, app.Id); err != nil {
		return "", err
	}
	deployment := newApplicationDeployment(app, "restart")
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": app.Id, "deployment_id": deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ApplicationStatus(ctx context.Context, userId string, applicationId string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	output, err := runApplicationCommand(ctx, filepath.Join(s.dataRoot, "cd", app.Code), "docker", "compose", "-f", "docker-compose.yml", "ps", "--format", "json")
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

func (s Service) ApplicationLogs(ctx context.Context, userId string, applicationId string, tail int) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if tail < 1 || tail > 1000 {
		return "", apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	output, err := runApplicationCommand(ctx, filepath.Join(s.dataRoot, "cd", app.Code), "docker", "compose", "-f", "docker-compose.yml", "logs", "--tail", strconv.Itoa(tail))
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

func (s Service) ApplicationComposePreview(ctx context.Context, userId string, applicationId string) (string, error) {
	app, compose, err := s.loadApplicationCompose(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	return s.renderApplicationCompose(ctx, app, compose)
}

func (s Service) ListApplicationRoutes(ctx context.Context, userId string, applicationId string) ([]repository.ApplicationRoute, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	routes, err := s.store.Routes(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list application routes", err)
	}
	return routes, nil
}

func (s Service) CreateApplicationRoute(ctx context.Context, userId string, applicationId string, input ApplicationRouteInput) (repository.ApplicationRoute, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.ApplicationRoute{}, err
	}
	if !app.RouteManaged {
		return repository.ApplicationRoute{}, apperror.New(apperror.KindValidation, "应用未启用路由托管")
	}
	serviceName, domain, port, err := normalizeApplicationRouteInput(input.ServiceName, input.Domain, input.Port)
	if err != nil {
		return repository.ApplicationRoute{}, err
	}
	route := repository.ApplicationRoute{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Domain: domain, Port: port}
	if err := s.store.CreateApplicationRoute(ctx, route); err != nil {
		return repository.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to create application route", err)
	}
	created, err := s.store.ApplicationRoute(ctx, route.Id)
	if err != nil {
		return repository.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
	}
	return created, nil
}

func (s Service) UpdateApplicationRoute(ctx context.Context, userId string, applicationId string, routeId string, input ApplicationRouteInput) (repository.ApplicationRoute, error) {
	route, err := s.loadApplicationRouteForUser(ctx, userId, applicationId, routeId)
	if err != nil {
		return repository.ApplicationRoute{}, err
	}
	serviceName, domain, port, err := normalizeApplicationRouteInput(input.ServiceName, input.Domain, input.Port)
	if err != nil {
		return repository.ApplicationRoute{}, err
	}
	route.ServiceName = serviceName
	route.Domain = domain
	route.Port = port
	if err := s.store.UpdateApplicationRoute(ctx, route); err != nil {
		return repository.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to update application route", err)
	}
	updated, err := s.store.ApplicationRoute(ctx, route.Id)
	if err != nil {
		return repository.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
	}
	return updated, nil
}

func (s Service) DeleteApplicationRoute(ctx context.Context, userId string, applicationId string, routeId string) error {
	route, err := s.loadApplicationRouteForUser(ctx, userId, applicationId, routeId)
	if err != nil {
		return err
	}
	if err := s.store.DeleteApplicationRoute(ctx, route.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete application route", err)
	}
	return nil
}

func (s Service) ApplicationComposeServices(ctx context.Context, userId string, applicationId string) ([]ApplicationServiceConfigView, error) {
	views, err := s.applicationServiceConfigViews(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	resp := make([]ApplicationServiceConfigView, 0, len(views))
	for _, view := range views {
		resp = append(resp, ApplicationServiceConfigView{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: view.DefaultPort})
	}
	return resp, nil
}

func (s Service) ApplicationServiceConfigs(ctx context.Context, userId string, applicationId string) ([]ApplicationServiceConfigView, error) {
	return s.applicationServiceConfigViews(ctx, userId, applicationId)
}

func (s Service) UpdateApplicationServiceConfig(ctx context.Context, userId string, applicationId string, serviceName string, imageInput *string) (ApplicationServiceConfigView, error) {
	app, compose, err := s.loadApplicationCompose(ctx, userId, applicationId)
	if err != nil {
		return ApplicationServiceConfigView{}, err
	}
	services, err := s.composeServices(app, compose)
	if err != nil {
		return ApplicationServiceConfigView{}, err
	}
	serviceName = strings.TrimSpace(serviceName)
	raw, exists := services[serviceName]
	if !exists {
		return ApplicationServiceConfigView{}, apperror.New(apperror.KindNotFound, "Service "+serviceName+" not found")
	}
	image := normalizeOptionalText(imageInput)
	if image == nil {
		return ApplicationServiceConfigView{}, apperror.New(apperror.KindValidation, "Image cannot be empty")
	}
	config, err := s.store.ApplicationServiceConfig(ctx, app.Id, serviceName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			config = repository.ApplicationServiceConfig{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: serviceName}
		} else {
			return ApplicationServiceConfigView{}, apperror.Wrap(apperror.KindInternal, "Failed to load application service config", err)
		}
	}
	config.Image = image
	if err := s.store.UpsertApplicationServiceConfig(ctx, config); err != nil {
		return ApplicationServiceConfigView{}, apperror.Wrap(apperror.KindInternal, "Failed to update application service config", err)
	}
	updated, err := s.store.ApplicationServiceConfig(ctx, app.Id, serviceName)
	if err != nil {
		return ApplicationServiceConfigView{}, apperror.Wrap(apperror.KindInternal, "Failed to load application service config", err)
	}
	return s.applicationServiceConfigView(app, serviceName, raw, &updated), nil
}

func (s Service) loadApplicationRouteForUser(ctx context.Context, userId string, applicationId string, routeId string) (repository.ApplicationRoute, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.ApplicationRoute{}, err
	}
	routeId = strings.TrimSpace(routeId)
	route, err := s.store.ApplicationRoute(ctx, routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.ApplicationRoute{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
		}
		return repository.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
	}
	if route.ApplicationId != app.Id {
		return repository.ApplicationRoute{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
	}
	return route, nil
}

func (s Service) loadApplicationCompose(ctx context.Context, userId string, applicationId string) (repository.Application, repository.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return repository.Application{}, repository.ApplicationConfigFile{}, err
	}
	compose, err := s.applicationComposeFile(ctx, app.Id)
	if err != nil {
		return repository.Application{}, repository.ApplicationConfigFile{}, err
	}
	return app, compose, nil
}

func (s Service) applicationComposeFile(ctx context.Context, applicationId string) (repository.ApplicationConfigFile, error) {
	files, err := s.store.ConfigFiles(ctx, applicationId)
	if err != nil {
		return repository.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application config files", err)
	}
	for _, file := range files {
		if file.Path == "docker-compose.yml" || file.Path == "docker-compose.yml.jinja" {
			return file, nil
		}
	}
	return repository.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "No docker-compose file found for this application")
}

func (s Service) renderApplicationCompose(ctx context.Context, app repository.Application, compose repository.ApplicationConfigFile) (string, error) {
	content := compose.Content
	var err error
	if strings.HasSuffix(compose.Path, ".jinja") {
		content, err = renderApplicationTemplate(content, app.Code, s.cfg)
		if err != nil {
			return "", apperror.New(apperror.KindValidation, err.Error())
		}
	}
	serviceConfigs, err := s.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load application service configs", err)
	}
	content = applyApplicationServiceConfigs(content, serviceConfigs)
	if app.RouteManaged {
		routes, err := s.store.Routes(ctx, app.Id)
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to load application routes", err)
		}
		content, err = injectApplicationRouteLabels(content, routes, s.cfg.Cert.LetsEncrypt.Enabled)
		if err != nil {
			return "", apperror.New(apperror.KindValidation, err.Error())
		}
	}
	return content, nil
}

func (s Service) composeServices(app repository.Application, compose repository.ApplicationConfigFile) (map[string]any, error) {
	content := compose.Content
	if strings.HasSuffix(compose.Path, ".jinja") {
		rendered, err := renderApplicationTemplate(content, app.Code, s.cfg)
		if err != nil {
			return nil, apperror.New(apperror.KindValidation, err.Error())
		}
		content = rendered
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return nil, apperror.New(apperror.KindValidation, "docker-compose 解析失败: "+err.Error())
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return nil, apperror.New(apperror.KindValidation, "docker-compose services 节点无效")
	}
	return services, nil
}

func (s Service) applicationServiceConfigViews(ctx context.Context, userId string, applicationId string) ([]ApplicationServiceConfigView, error) {
	app, compose, err := s.loadApplicationCompose(ctx, userId, applicationId)
	if err != nil {
		return nil, err
	}
	services, err := s.composeServices(app, compose)
	if err != nil {
		return nil, err
	}
	configs, err := s.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load application service configs", err)
	}
	byName := map[string]repository.ApplicationServiceConfig{}
	for _, config := range configs {
		byName[config.ServiceName] = config
	}
	resp := make([]ApplicationServiceConfigView, 0, len(services))
	for name, raw := range services {
		var config *repository.ApplicationServiceConfig
		if value, ok := byName[name]; ok {
			config = &value
		}
		resp = append(resp, s.applicationServiceConfigView(app, name, raw, config))
	}
	return resp, nil
}

func (s Service) applicationServiceConfigView(app repository.Application, serviceName string, raw any, config *repository.ApplicationServiceConfig) ApplicationServiceConfigView {
	service, _ := raw.(map[string]any)
	baseImage := optionalStringFromAny(service["image"])
	var image *string
	var configId *string
	var createdAt *string
	var updatedAt *string
	if config != nil {
		image = normalizeOptionalText(config.Image)
		configId = &config.Id
		created := formatTime(config.CreatedAt)
		updated := formatTime(config.UpdatedAt)
		createdAt = &created
		updatedAt = &updated
	}
	return ApplicationServiceConfigView{ServiceName: serviceName, DefaultDomain: app.Code + "." + s.cfg.Traefik.DomainSuffix, DefaultPort: extractDefaultPort(service), BaseImage: baseImage, Image: image, ConfigId: configId, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func normalizeApplicationImportInput(input ApplicationImportInput) (string, string, string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	imagePullPolicy := strings.TrimSpace(input.ImagePullPolicy)
	if imagePullPolicy == "" {
		imagePullPolicy = "missing"
	}
	if name == "" || len(name) > 100 || code == "" || len(code) > 100 || !applicationCreateCodePattern.MatchString(code) || !validImagePullPolicy(imagePullPolicy) {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid application fields")
	}
	return name, code, imagePullPolicy, nil
}

func (s Service) ensureApplicationCodeAvailable(ctx context.Context, code string) error {
	existing, err := s.store.ApplicationByCode(ctx, code)
	if err == nil {
		return apperror.New(apperror.KindValidation, "Application code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check application code", err)
	}
	return nil
}

func normalizeApplicationRouteInput(serviceName string, domain string, port int) (string, string, int, error) {
	serviceName = strings.TrimSpace(serviceName)
	domain = strings.TrimSpace(domain)
	if serviceName == "" || domain == "" || port < 1 || port > 65535 {
		return "", "", 0, apperror.New(apperror.KindValidation, "Invalid application route fields")
	}
	return serviceName, domain, port, nil
}

func runApplicationCommand(ctx context.Context, cwd string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("command is required")
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = cwd
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func renderApplicationTemplate(content string, appCode string, cfg config.Config) (string, error) {
	context := map[string]any{
		"app": map[string]any{
			"code":             appCode,
			"physical_dir":     cfg.DataRoot(),
			"physical_app_dir": filepath.Join(cfg.DataRoot(), "cd", appCode),
		},
		"config": map[string]any{"domain_suffix": cfg.Traefik.DomainSuffix},
		"cert": map[string]any{"letsencrypt": map[string]any{
			"enabled":      cfg.Cert.LetsEncrypt.Enabled,
			"email":        cfg.Cert.LetsEncrypt.Email,
			"challenge":    cfg.Cert.LetsEncrypt.Challenge,
			"dns_provider": cfg.Cert.LetsEncrypt.DNSProvider,
		}},
	}
	return templatex.Render(content, context)
}

func applyApplicationServiceConfigs(compose string, configs []repository.ApplicationServiceConfig) string {
	if len(configs) == 0 {
		return compose
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return compose
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return compose
	}
	for _, config := range configs {
		service, ok := services[config.ServiceName].(map[string]any)
		if !ok || config.Image == nil || strings.TrimSpace(*config.Image) == "" {
			continue
		}
		service["image"] = strings.TrimSpace(*config.Image)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return compose
	}
	return string(out)
}

func injectApplicationRouteLabels(compose string, routes []repository.ApplicationRoute, letsEncrypt bool) (string, error) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return "", err
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return "", errors.New("docker-compose services must be a mapping")
	}
	for _, raw := range services {
		if service, ok := raw.(map[string]any); ok {
			delete(service, "labels")
		}
	}
	for _, route := range routes {
		service, ok := services[route.ServiceName].(map[string]any)
		if !ok {
			return "", errors.New("service " + route.ServiceName + " not found in docker-compose.yml")
		}
		labels := []string{
			"traefik.enable=true",
			"traefik.http.routers." + route.ServiceName + ".rule=Host(`" + route.Domain + "`)",
			"traefik.http.services." + route.ServiceName + ".loadbalancer.server.port=" + strconv.Itoa(route.Port),
		}
		if letsEncrypt {
			labels = append(labels, "traefik.http.routers."+route.ServiceName+".entrypoints=websecure", "traefik.http.routers."+route.ServiceName+".tls=true", "traefik.http.routers."+route.ServiceName+".tls.certresolver=letsencrypt")
		} else {
			labels = append(labels, "traefik.http.routers."+route.ServiceName+".entrypoints=web")
		}
		service["labels"] = labels
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}

func optionalStringFromAny(value any) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(toString(value))
	if text == "" {
		return nil
	}
	return &text
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func extractDefaultPort(service map[string]any) int {
	ports, ok := service["ports"].([]any)
	if !ok || len(ports) == 0 {
		return 80
	}
	first := toString(ports[0])
	parts := strings.Split(strings.Split(first, "/")[0], ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 80
	}
	return port
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
