package cdsvc

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/template"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"

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

var applicationRouteDomainPattern = regexp.MustCompile(`(?i)^(localhost|([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)$`)

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
	Application    model.Application
	ConfigFiles    []model.ApplicationConfigFile
	ServiceConfigs []model.ApplicationServiceConfig
	Routes         []model.ApplicationRoute
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

func (s Service) ImportApplication(ctx context.Context, userId string, input ApplicationImportInput) (model.Application, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	name, code, imagePullPolicy, err := normalizeApplicationImportInput(input)
	if err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationNameAvailable(ctx, name); err != nil {
		return model.Application{}, err
	}
	if err := s.ensureApplicationCodeAvailable(ctx, code); err != nil {
		return model.Application{}, err
	}
	app := model.Application{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, ImagePullPolicy: imagePullPolicy, RouteManaged: input.RouteManaged, Status: status.ApplicationStatusUndeployed}
	files := make([]model.ApplicationConfigFile, 0, len(input.ConfigFiles))
	for _, item := range input.ConfigFiles {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
		}
		files = append(files, model.ApplicationConfigFile{Id: idutil.NewId(), ApplicationId: app.Id, Path: path, Content: item.Content})
	}
	serviceConfigs := make([]model.ApplicationServiceConfig, 0, len(input.ServiceConfigs))
	for _, item := range input.ServiceConfigs {
		serviceName := strings.TrimSpace(item.ServiceName)
		if serviceName == "" {
			return model.Application{}, apperror.New(apperror.KindValidation, "Invalid application service config fields")
		}
		image := normalizeOptionalText(item.Image)
		environment := normalizeOptionalText(item.Environment)
		volumes := normalizeOptionalText(item.Volumes)
		if image == nil && environment == nil && volumes == nil {
			continue
		}
		serviceConfigs = append(serviceConfigs, model.ApplicationServiceConfig{Id: idutil.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Image: image, Environment: environment, Volumes: volumes})
	}
	routes := make([]model.ApplicationRoute, 0, len(input.ApplicationRoutes))
	for _, item := range input.ApplicationRoutes {
		serviceName, domain, port, err := normalizeApplicationRouteInput(item.ServiceName, item.Domain, item.Port)
		if err != nil {
			return model.Application{}, err
		}
		routes = append(routes, model.ApplicationRoute{Id: idutil.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Domain: domain, Port: port})
	}
	if err := s.store.CreateApplicationBundle(ctx, app, files, serviceConfigs, routes); err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to import application", err)
	}
	created, err := s.store.Application(ctx, app.Id)
	if err != nil {
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
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

func (s Service) ListApplicationFiles(ctx context.Context, userId string, applicationId string) ([]model.ApplicationConfigFile, error) {
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

func (s Service) CreateApplicationFile(ctx context.Context, userId string, applicationId string, input ConfigFileInput) (model.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.ApplicationConfigFile{}, err
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return model.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
	}
	file := model.ApplicationConfigFile{Id: idutil.NewId(), ApplicationId: app.Id, Path: path, Content: input.Content}
	if err := s.store.CreateConfigFile(ctx, file); err != nil {
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to create application file", err)
	}
	created, err := s.store.ConfigFile(ctx, file.Id)
	if err != nil {
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
	}
	return created, nil
}

func (s Service) ApplicationFileForUser(ctx context.Context, userId string, applicationId string, fileId string) (model.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.ApplicationConfigFile{}, err
	}
	fileId = strings.TrimSpace(fileId)
	file, err := s.store.ConfigFile(ctx, fileId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApplicationConfigFile{}, apperror.New(apperror.KindNotFound, "Config file "+fileId+" not found")
		}
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
	}
	if file.ApplicationId != app.Id {
		return model.ApplicationConfigFile{}, apperror.New(apperror.KindNotFound, "Config file "+fileId+" not found")
	}
	return file, nil
}

func (s Service) UpdateApplicationFile(ctx context.Context, userId string, applicationId string, fileId string, input ConfigFileInput) (model.ApplicationConfigFile, error) {
	file, err := s.ApplicationFileForUser(ctx, userId, applicationId, fileId)
	if err != nil {
		return model.ApplicationConfigFile{}, err
	}
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return model.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "Invalid config file fields")
	}
	file.Path = path
	file.Content = input.Content
	if err := s.store.UpdateConfigFile(ctx, file); err != nil {
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to update application file", err)
	}
	updated, err := s.store.ConfigFile(ctx, file.Id)
	if err != nil {
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application file", err)
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
	if app.Status != status.ApplicationStatusDeployed {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法停止")
	}
	deployment := newApplicationDeployment(app, "stop")
	deployment.CommandText = stopComposeCommand(removeVolumes).String()
	if err := s.store.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationStop, map[string]any{"application_id": app.Id, "deployment_id": deployment.Id, "remove_volumes": removeVolumes}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) RestartApplication(ctx context.Context, userId string, applicationId string) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if app.Status != status.ApplicationStatusDeployed {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法重启")
	}
	if err := s.ensureApplicationComposeFile(ctx, app.Id); err != nil {
		return "", err
	}
	deployment := newApplicationDeployment(app, "restart")
	deployment.CommandText = restartComposeCommand().String()
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
	output, err := runApplicationCommand(ctx, s.workspace.AppDir(app.Code), "docker", "compose", "-f", "docker-compose.yml", "ps", "--format", "json")
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
	command := containerLogsTailCommand(strconv.Itoa(tail))
	output, err := runApplicationCommand(ctx, s.workspace.AppDir(app.Code), command.argv()...)
	if err != nil {
		return outputOrError(output, err), apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}

func (s Service) DeploymentContainerLog(ctx context.Context, userId string, deploymentId string, tail int) (DeploymentContainerLog, error) {
	if tail < 1 || tail > 1000 {
		return DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return DeploymentContainerLog{}, err
	}
	if deployment.OperationType == "stop" {
		return DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "container logs are not available for stop deployments")
	}
	if deployment.ApplicationId == nil || strings.TrimSpace(*deployment.ApplicationId) == "" {
		return DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.store.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeploymentContainerLog{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	appDir := s.workspace.AppDir(app.Code)
	sinceCommand := containerLogsSinceCommand(deployment.StartedAt.UTC().Format(time.RFC3339))
	output, err := runApplicationCommand(ctx, appDir, sinceCommand.argv()...)
	if err == nil {
		return DeploymentContainerLog{Logs: output, Source: "since", IsRealtimeSupported: true}, nil
	}
	tailCommand := containerLogsTailCommand(strconv.Itoa(tail))
	output, tailErr := runApplicationCommand(ctx, appDir, tailCommand.argv()...)
	if tailErr != nil {
		return DeploymentContainerLog{}, apperror.New(apperror.KindInternal, outputOrError(output, tailErr))
	}
	return DeploymentContainerLog{Logs: output, Source: "tail", IsRealtimeSupported: true}, nil
}

func (s Service) ApplicationComposePreview(ctx context.Context, userId string, applicationId string) (string, error) {
	app, compose, err := s.loadApplicationCompose(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	return s.renderApplicationCompose(ctx, app, compose)
}

func (s Service) ListApplicationRoutes(ctx context.Context, userId string, applicationId string) ([]model.ApplicationRoute, error) {
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

func (s Service) CreateApplicationRoute(ctx context.Context, userId string, applicationId string, input ApplicationRouteInput) (model.ApplicationRoute, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.ApplicationRoute{}, err
	}
	if !app.RouteManaged {
		return model.ApplicationRoute{}, apperror.New(apperror.KindValidation, "应用未启用路由托管")
	}
	serviceName, domain, port, err := normalizeApplicationRouteInput(input.ServiceName, input.Domain, input.Port)
	if err != nil {
		return model.ApplicationRoute{}, err
	}
	route := model.ApplicationRoute{Id: idutil.NewId(), ApplicationId: app.Id, ServiceName: serviceName, Domain: domain, Port: port}
	if err := s.store.CreateApplicationRoute(ctx, route); err != nil {
		return model.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to create application route", err)
	}
	created, err := s.store.ApplicationRoute(ctx, route.Id)
	if err != nil {
		return model.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
	}
	return created, nil
}

func (s Service) UpdateApplicationRoute(ctx context.Context, userId string, applicationId string, routeId string, input ApplicationRouteInput) (model.ApplicationRoute, error) {
	route, err := s.loadApplicationRouteForUser(ctx, userId, applicationId, routeId)
	if err != nil {
		return model.ApplicationRoute{}, err
	}
	serviceName, domain, port, err := normalizeApplicationRouteInput(input.ServiceName, input.Domain, input.Port)
	if err != nil {
		return model.ApplicationRoute{}, err
	}
	route.ServiceName = serviceName
	route.Domain = domain
	route.Port = port
	if err := s.store.UpdateApplicationRoute(ctx, route); err != nil {
		return model.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to update application route", err)
	}
	updated, err := s.store.ApplicationRoute(ctx, route.Id)
	if err != nil {
		return model.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
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
	services, err := s.composeServices(ctx, app, compose)
	if err != nil {
		return ApplicationServiceConfigView{}, err
	}
	serviceName = strings.TrimSpace(serviceName)
	raw, exists := services[serviceName]
	if !exists {
		return ApplicationServiceConfigView{}, apperror.New(apperror.KindNotFound, "Service "+serviceName+" not found")
	}
	image := normalizeOptionalText(imageInput)
	config, err := s.store.ApplicationServiceConfig(ctx, app.Id, serviceName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if image == nil {
				return s.applicationServiceConfigView(app, serviceName, raw, nil), nil
			}
			config = model.ApplicationServiceConfig{Id: idutil.NewId(), ApplicationId: app.Id, ServiceName: serviceName}
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

func (s Service) loadApplicationRouteForUser(ctx context.Context, userId string, applicationId string, routeId string) (model.ApplicationRoute, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.ApplicationRoute{}, err
	}
	routeId = strings.TrimSpace(routeId)
	route, err := s.store.ApplicationRoute(ctx, routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ApplicationRoute{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
		}
		return model.ApplicationRoute{}, apperror.Wrap(apperror.KindInternal, "Failed to load application route", err)
	}
	if route.ApplicationId != app.Id {
		return model.ApplicationRoute{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
	}
	return route, nil
}

func (s Service) loadApplicationCompose(ctx context.Context, userId string, applicationId string) (model.Application, model.ApplicationConfigFile, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return model.Application{}, model.ApplicationConfigFile{}, err
	}
	compose, err := s.applicationComposeFile(ctx, app.Id)
	if err != nil {
		return model.Application{}, model.ApplicationConfigFile{}, err
	}
	return app, compose, nil
}

func (s Service) applicationComposeFile(ctx context.Context, applicationId string) (model.ApplicationConfigFile, error) {
	files, err := s.store.ConfigFiles(ctx, applicationId)
	if err != nil {
		return model.ApplicationConfigFile{}, apperror.Wrap(apperror.KindInternal, "Failed to load application config files", err)
	}
	for _, file := range files {
		if file.Path == "docker-compose.yml" || file.Path == "docker-compose.yml.liquid" {
			return file, nil
		}
	}
	return model.ApplicationConfigFile{}, apperror.New(apperror.KindValidation, "No docker-compose file found for this application")
}

func (s Service) renderApplicationCompose(ctx context.Context, app model.Application, compose model.ApplicationConfigFile) (string, error) {
	serviceConfigs, err := s.executionStore.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load application service configs", err)
	}
	var routes []model.ApplicationRoute
	if app.RouteManaged {
		routes, err = s.executionStore.Routes(ctx, app.Id)
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to load application routes", err)
		}
	}
	_, content, err := s.renderApplicationConfigFile(ctx, app, compose.Path, compose.Content, serviceConfigs, routes)
	return content, err
}

func (s Service) renderApplicationConfigFile(ctx context.Context, app model.Application, path string, content string, serviceConfigs []model.ApplicationServiceConfig, routes []model.ApplicationRoute) (string, string, error) {
	if strings.HasSuffix(path, ".liquid") {
		path = strings.TrimSuffix(path, ".liquid")
		rendered, err := s.renderApplicationTemplate(ctx, content, app.Code)
		if err != nil {
			return "", "", apperror.New(apperror.KindValidation, err.Error())
		}
		content = rendered
	}
	if path == "docker-compose.yml" {
		content = applyApplicationServiceConfigs(content, serviceConfigs)
		if app.RouteManaged {
			rendered, err := injectApplicationRouteLabels(content, routes, s.cfg.Cert.LetsEncrypt.Enabled)
			if err != nil {
				return "", "", apperror.New(apperror.KindValidation, err.Error())
			}
			content = rendered
		}
	}
	return path, content, nil
}

func (s Service) composeServices(ctx context.Context, app model.Application, compose model.ApplicationConfigFile) (map[string]any, error) {
	content := compose.Content
	if strings.HasSuffix(compose.Path, ".liquid") {
		rendered, err := s.renderApplicationTemplate(ctx, content, app.Code)
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
	services, err := s.composeServices(ctx, app, compose)
	if err != nil {
		return nil, err
	}
	configs, err := s.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load application service configs", err)
	}
	byName := map[string]model.ApplicationServiceConfig{}
	for _, config := range configs {
		byName[config.ServiceName] = config
	}
	resp := make([]ApplicationServiceConfigView, 0, len(services))
	for name, raw := range services {
		var config *model.ApplicationServiceConfig
		if value, ok := byName[name]; ok {
			config = &value
		}
		resp = append(resp, s.applicationServiceConfigView(app, name, raw, config))
	}
	return resp, nil
}

func (s Service) applicationServiceConfigView(app model.Application, serviceName string, raw any, config *model.ApplicationServiceConfig) ApplicationServiceConfigView {
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
	if serviceName == "" || !validApplicationRouteDomain(domain) || port < 1 || port > 65535 {
		return "", "", 0, apperror.New(apperror.KindValidation, "Invalid application route fields")
	}
	return serviceName, strings.ToLower(domain), port, nil
}

func validApplicationRouteDomain(domain string) bool {
	if domain == "" || len(domain) > 253 || strings.ContainsAny(domain, " `\t\r\n") {
		return false
	}
	return applicationRouteDomainPattern.MatchString(domain)
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

func (s Service) renderApplicationTemplate(ctx context.Context, content string, appCode string) (string, error) {
	physicalDir, err := s.workspace.PhysicalDir(ctx)
	if err != nil {
		return "", err
	}
	physicalAppDir, err := s.workspace.PhysicalAppDir(ctx, appCode)
	if err != nil {
		return "", err
	}
	context := map[string]any{
		"app": map[string]any{
			"code":             appCode,
			"physical_dir":     physicalDir,
			"physical_app_dir": physicalAppDir,
		},
		"config": map[string]any{"domain_suffix": s.cfg.Traefik.DomainSuffix},
		"cert": map[string]any{"letsencrypt": map[string]any{
			"enabled":      s.cfg.Cert.LetsEncrypt.Enabled,
			"email":        s.cfg.Cert.LetsEncrypt.Email,
			"challenge":    s.cfg.Cert.LetsEncrypt.Challenge,
			"dns_provider": s.cfg.Cert.LetsEncrypt.DNSProvider,
		}},
	}
	return templatex.Render(content, context)
}

func applyApplicationServiceConfigs(compose string, configs []model.ApplicationServiceConfig) string {
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

func injectApplicationRouteLabels(compose string, routes []model.ApplicationRoute, letsEncrypt bool) (string, error) {
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
	for serviceName, group := range groupApplicationRoutes(routes) {
		service, ok := services[serviceName].(map[string]any)
		if !ok {
			return "", errors.New("service " + serviceName + " not found in docker-compose.yml")
		}
		labels, err := applicationRouteLabels(serviceName, group, letsEncrypt)
		if err != nil {
			return "", err
		}
		service["labels"] = labels
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func groupApplicationRoutes(routes []model.ApplicationRoute) map[string][]model.ApplicationRoute {
	groups := make(map[string][]model.ApplicationRoute)
	for _, route := range routes {
		groups[route.ServiceName] = append(groups[route.ServiceName], route)
	}
	return groups
}

func applicationRouteLabels(serviceName string, routes []model.ApplicationRoute, letsEncrypt bool) ([]string, error) {
	if len(routes) == 0 {
		return nil, errors.New("application route group is empty")
	}
	port := routes[0].Port
	hosts := make([]string, 0, len(routes))
	seen := map[string]struct{}{}
	for _, route := range routes {
		if route.Port != port {
			return nil, errors.New("service " + serviceName + " has routes with different ports")
		}
		domain := strings.ToLower(strings.TrimSpace(route.Domain))
		if !validApplicationRouteDomain(domain) {
			return nil, errors.New("service " + serviceName + " has invalid route domain")
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		hosts = append(hosts, "Host(`"+domain+"`)")
	}
	labels := []string{
		"traefik.enable=true",
		"traefik.http.routers." + serviceName + ".rule=" + strings.Join(hosts, " || "),
		"traefik.http.services." + serviceName + ".loadbalancer.server.port=" + strconv.Itoa(port),
	}
	if letsEncrypt {
		labels = append(labels, "traefik.http.routers."+serviceName+".entrypoints=websecure", "traefik.http.routers."+serviceName+".tls=true", "traefik.http.routers."+serviceName+".tls.certresolver=letsencrypt")
	} else {
		labels = append(labels, "traefik.http.routers."+serviceName+".entrypoints=web")
	}
	return labels, nil
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
