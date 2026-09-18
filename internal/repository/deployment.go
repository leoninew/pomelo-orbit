package repository

import (
	"context"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// DeploymentStore persists deployment executions.
type DeploymentStore interface {
	CreateDeployment(ctx context.Context, projectId string, deployment model.Deployment) error
	BeginDeployment(ctx context.Context, projectId string, id string) (bool, error)
	CompleteDeployment(ctx context.Context, projectId string, id string, status string, message string) (bool, error)
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.Deployment], error)
	Deployment(ctx context.Context, projectId string, id string) (model.Deployment, error)
	DeleteDeployment(ctx context.Context, projectId string, id string) error
	ClearProjectDeploymentHistory(ctx context.Context, projectId string) error
	CancelDeployment(ctx context.Context, projectId string, id string) (bool, error)
	HasActiveDeployment(ctx context.Context, projectId string, serviceId string) (bool, error)
	LatestSuccessfulDeploymentPlanHash(ctx context.Context, projectId string, serviceId string) (*string, error)
}
