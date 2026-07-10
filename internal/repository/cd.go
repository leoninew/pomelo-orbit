package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// CDStore persists application deployment configuration and routes.
type CDStore interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[model.Application], error)
	Application(ctx context.Context, id string) (model.Application, error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]model.ApplicationConfigFile, error)
	ConfigFile(ctx context.Context, id string) (model.ApplicationConfigFile, error)
	CreateConfigFile(ctx context.Context, file model.ApplicationConfigFile) error
	UpdateConfigFile(ctx context.Context, file model.ApplicationConfigFile) error
	DeleteConfigFile(ctx context.Context, id string) error
	Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error)
	ApplicationRoute(ctx context.Context, id string) (model.ApplicationRoute, error)
	CreateApplicationRoute(ctx context.Context, route model.ApplicationRoute) error
	UpdateApplicationRoute(ctx context.Context, route model.ApplicationRoute) error
	DeleteApplicationRoute(ctx context.Context, id string) error
	ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error)
	ApplicationServiceConfig(ctx context.Context, applicationId string, serviceName string) (model.ApplicationServiceConfig, error)
	UpsertApplicationServiceConfig(ctx context.Context, config model.ApplicationServiceConfig) error
	CreateApplication(ctx context.Context, app model.Application) error
	CreateApplicationBundle(ctx context.Context, app model.Application, files []model.ApplicationConfigFile, serviceConfigs []model.ApplicationServiceConfig, routes []model.ApplicationRoute) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error
	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	MarkApplicationStatus(ctx context.Context, id string, status string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.Deployment], error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	CancelDeployment(ctx context.Context, id string) error
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Route], error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
	Route(ctx context.Context, id string) (model.Route, error)
	RouteByDomain(ctx context.Context, domain string) (model.Route, error)
	CreateRoute(ctx context.Context, route model.Route) error
	UpdateRoute(ctx context.Context, route model.Route) error
	DeleteRoute(ctx context.Context, id string) error
}

// DeploymentExecutionStore persists state transitions and inputs of a deployment execution.
type DeploymentExecutionStore interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]model.ApplicationConfigFile, error)
	ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error)
	Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error)
	MarkApplicationStatus(ctx context.Context, id string, status string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
}
