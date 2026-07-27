package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// ServiceReader provides runtime service binding queries to other domains.
type ServiceReader interface {
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	ListServicesByProject(ctx context.Context, projectId string, applicationId string, status string, search string, page int, perPage int) (Page[model.ServiceListItem], error)
	ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error)
	ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
}

// ServiceStore persists runtime service bindings.
type ServiceStore interface {
	ServiceReader
	UpsertService(ctx context.Context, svc model.Service) error
	DeleteService(ctx context.Context, id string) error
	UpdateServiceRuntimeConfig(ctx context.Context, id string, runtimeConfig map[string]string) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error
}
