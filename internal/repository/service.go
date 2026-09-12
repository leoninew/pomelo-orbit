package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// ServiceReader provides runtime service binding queries to other domains.
type ServiceReader interface {
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	ListServicesByProject(ctx context.Context, projectId string, applicationId string, status string, search string, page int, perPage int) (Page[model.ServiceListItem], error)
	ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error)
	ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error)
	ServiceByProjectAndCode(ctx context.Context, projectId string, code string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	ServiceEnvByService(ctx context.Context, serviceId string) ([]model.ServiceEnv, error)
	ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error)
	ServiceComponent(ctx context.Context, id string) (model.ServiceComponent, error)
}

// ServiceStore persists runtime service bindings.
type ServiceStore interface {
	ServiceReader
	UpsertService(ctx context.Context, svc model.Service) error
	CreateServiceWithComponents(ctx context.Context, svc model.Service, components []model.ServiceComponent) error
	UpdateServiceConfiguration(ctx context.Context, svc model.Service, components []model.ServiceComponent) error
	ReplaceServiceEnv(ctx context.Context, serviceId string, env []model.ServiceEnv) error
	UpdateServiceComponentOverlay(ctx context.Context, component model.ServiceComponent) error
	DeleteService(ctx context.Context, id string) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string) error
}
