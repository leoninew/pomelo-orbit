package deploymentsvc

import (
	"context"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// stores composes deployment-side domain stores. It is not a repository interface.
type stores struct {
	project     repository.ProjectReader
	application repository.ApplicationStore
	service     repository.ServiceStore
	deployment  repository.DeploymentStore
	gateway     repository.GatewayStore
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}

func (s stores) ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	return s.application.ListApplications(ctx, projectId, page, perPage, search, kind)
}
func (s stores) Application(ctx context.Context, id string) (model.Application, error) {
	return s.application.Application(ctx, id)
}
func (s stores) ApplicationByName(ctx context.Context, name string) (model.Application, error) {
	return s.application.ApplicationByName(ctx, name)
}
func (s stores) ApplicationByCode(ctx context.Context, code string) (model.Application, error) {
	return s.application.ApplicationByCode(ctx, code)
}
func (s stores) CreateApplication(ctx context.Context, app model.Application) error {
	return s.application.CreateApplication(ctx, app)
}
func (s stores) UpdateApplication(ctx context.Context, app model.Application) error {
	return s.application.UpdateApplication(ctx, app)
}
func (s stores) DeleteApplication(ctx context.Context, id string) error {
	return s.application.DeleteApplication(ctx, id)
}
func (s stores) ListVersions(ctx context.Context, applicationId string) ([]model.Version, error) {
	return s.application.ListVersions(ctx, applicationId)
}
func (s stores) ListVersionsPage(ctx context.Context, applicationId string, page int, perPage int, search string) (repository.Page[model.Version], error) {
	return s.application.ListVersionsPage(ctx, applicationId, page, perPage, search)
}
func (s stores) Version(ctx context.Context, id string) (model.Version, error) {
	return s.application.Version(ctx, id)
}
func (s stores) CreateVersion(ctx context.Context, version model.Version) error {
	return s.application.CreateVersion(ctx, version)
}
func (s stores) UpdateVersion(ctx context.Context, version model.Version) error {
	return s.application.UpdateVersion(ctx, version)
}
func (s stores) DeleteVersion(ctx context.Context, id string) error {
	return s.application.DeleteVersion(ctx, id)
}
func (s stores) CountVersionRuntimeRefs(ctx context.Context, versionId string) (int, error) {
	return s.application.CountVersionRuntimeRefs(ctx, versionId)
}
func (s stores) VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, versionId)
}
func (s stores) ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error {
	return s.application.ReplaceVersionComponents(ctx, versionId, components)
}

func (s stores) ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error) {
	return s.service.ListServicesByApplication(ctx, applicationId)
}
func (s stores) ListServicesByProject(ctx context.Context, projectId string, applicationId string, status string, search string, page int, perPage int) (repository.Page[model.ServiceListItem], error) {
	return s.service.ListServicesByProject(ctx, projectId, applicationId, status, search, page, perPage)
}
func (s stores) ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error) {
	return s.service.ServiceListItem(ctx, id)
}
func (s stores) ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error) {
	return s.service.ServiceByKey(ctx, applicationId, instanceKey)
}
func (s stores) Service(ctx context.Context, id string) (model.Service, error) {
	return s.service.Service(ctx, id)
}
func (s stores) ServiceEnvByService(ctx context.Context, serviceId string) ([]model.ServiceEnv, error) {
	return s.service.ServiceEnvByService(ctx, serviceId)
}
func (s stores) ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error) {
	return s.service.ServiceComponentsByService(ctx, serviceId)
}
func (s stores) UpsertService(ctx context.Context, svc model.Service) error {
	return s.service.UpsertService(ctx, svc)
}
func (s stores) UpdateServiceStatus(ctx context.Context, id string, status string) error {
	return s.service.UpdateServiceStatus(ctx, id, status)
}
func (s stores) UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string) error {
	return s.service.UpdateServiceAfterDeploy(ctx, id, status, versionId)
}

func (s stores) CreateDeployment(ctx context.Context, deployment model.Deployment) error {
	return s.deployment.CreateDeployment(ctx, deployment)
}
func (s stores) CompleteDeployment(ctx context.Context, id string, status string, message string) (bool, error) {
	return s.deployment.CompleteDeployment(ctx, id, status, message)
}
func (s stores) ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.Deployment], error) {
	return s.deployment.ListDeployments(ctx, projectId, applicationId, status, search, dateFrom, dateTo, page, perPage)
}
func (s stores) Deployment(ctx context.Context, id string) (model.Deployment, error) {
	return s.deployment.Deployment(ctx, id)
}
func (s stores) CancelDeployment(ctx context.Context, id string) (bool, error) {
	return s.deployment.CancelDeployment(ctx, id)
}
func (s stores) BeginDeployment(ctx context.Context, id string) (bool, error) {
	return s.deployment.BeginDeployment(ctx, id)
}
func (s stores) HasActiveDeployment(ctx context.Context, serviceID string) (bool, error) {
	return s.deployment.HasActiveDeployment(ctx, serviceID)
}

func (s stores) HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error) {
	return s.gateway.HasActiveGatewayService(ctx, excludeApplicationId)
}
func (s stores) GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error) {
	return s.gateway.GatewayConfig(ctx, applicationId)
}
func (s stores) ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error) {
	return s.gateway.ResolveActiveGatewayConfig(ctx)
}
func (s stores) ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error) {
	return s.gateway.ListGatewayApplications(ctx, projectId)
}
func (s stores) UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error {
	return s.gateway.UpsertGatewayConfig(ctx, cfg)
}
