package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// DeploymentStore persists deployment executions.
type DeploymentStore interface {
	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	BeginDeployment(ctx context.Context, id string) (bool, error)
	CompleteDeployment(ctx context.Context, id string, status string, message string) (bool, error)
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.Deployment], error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	CancelDeployment(ctx context.Context, id string) (bool, error)
	HasActiveDeployment(ctx context.Context, serviceId string) (bool, error)
	LatestSuccessfulDeploymentPlanHash(ctx context.Context, serviceId string) (*string, error)
}
