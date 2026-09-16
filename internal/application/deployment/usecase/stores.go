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

func (s stores) ListApplications(ctx context.Context, projectId string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	return s.application.ListApplications(ctx, projectId, page, perPage, search, kind)
}
func (s stores) Application(ctx context.Context, projectId string, id string) (model.Application, error) {
	return s.application.Application(ctx, projectId, id)
}
func (s stores) ApplicationByName(ctx context.Context, projectId string, name string) (model.Application, error) {
	return s.application.ApplicationByName(ctx, projectId, name)
}
func (s stores) ApplicationByCode(ctx context.Context, projectId string, code string) (model.Application, error) {
	return s.application.ApplicationByCode(ctx, projectId, code)
}
func (s stores) CreateApplication(ctx context.Context, app model.Application) error {
	return s.application.CreateApplication(ctx, app)
}
func (s stores) UpdateApplication(ctx context.Context, projectId string, app model.Application) error {
	return s.application.UpdateApplication(ctx, projectId, app)
}
func (s stores) DeleteApplication(ctx context.Context, projectId string, id string) error {
	return s.application.DeleteApplication(ctx, projectId, id)
}
func (s stores) ListVersions(ctx context.Context, projectId string, applicationId string) ([]model.Version, error) {
	return s.application.ListVersions(ctx, projectId, applicationId)
}
func (s stores) ListVersionsPage(ctx context.Context, projectId string, applicationId string, page int, perPage int, search string) (repository.Page[model.Version], error) {
	return s.application.ListVersionsPage(ctx, projectId, applicationId, page, perPage, search)
}
func (s stores) Version(ctx context.Context, projectId string, id string) (model.Version, error) {
	return s.application.Version(ctx, projectId, id)
}
func (s stores) CreateVersion(ctx context.Context, projectId string, version model.Version) error {
	return s.application.CreateVersion(ctx, projectId, version)
}
func (s stores) UpdateVersion(ctx context.Context, projectId string, version model.Version) error {
	return s.application.UpdateVersion(ctx, projectId, version)
}
func (s stores) DeleteVersion(ctx context.Context, projectId string, id string) error {
	return s.application.DeleteVersion(ctx, projectId, id)
}
func (s stores) CountVersionRuntimeRefs(ctx context.Context, projectId string, versionId string) (int, error) {
	return s.application.CountVersionRuntimeRefs(ctx, projectId, versionId)
}
func (s stores) VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, projectId, versionId)
}
func (s stores) ReplaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error {
	return s.application.ReplaceVersionComponents(ctx, projectId, versionId, components)
}

func (s stores) ListServicesByApplication(ctx context.Context, projectId, applicationId string) ([]model.Service, error) {
	return s.service.ListServicesByApplication(ctx, projectId, applicationId)
}
func (s stores) ListServicesByProject(ctx context.Context, projectId string, applicationId string, status string, search string, page int, perPage int) (repository.Page[model.ServiceListItem], error) {
	return s.service.ListServicesByProject(ctx, projectId, applicationId, status, search, page, perPage)
}
func (s stores) ServiceListItem(ctx context.Context, projectId, id string) (model.ServiceListItem, error) {
	return s.service.ServiceListItem(ctx, projectId, id)
}
func (s stores) ServiceByKey(ctx context.Context, projectId, applicationId, instanceKey string) (model.Service, error) {
	return s.service.ServiceByKey(ctx, projectId, applicationId, instanceKey)
}
func (s stores) Service(ctx context.Context, projectId, id string) (model.Service, error) {
	return s.service.Service(ctx, projectId, id)
}
func (s stores) ServiceEnvByService(ctx context.Context, projectId, serviceId string) ([]model.ServiceEnv, error) {
	return s.service.ServiceEnvByService(ctx, projectId, serviceId)
}
func (s stores) ServiceComponentsByService(ctx context.Context, projectId, serviceId string) ([]model.ServiceComponent, error) {
	return s.service.ServiceComponentsByService(ctx, projectId, serviceId)
}
func (s stores) UpsertService(ctx context.Context, projectId string, svc model.Service) error {
	return s.service.UpsertService(ctx, projectId, svc)
}
func (s stores) UpdateServiceStatus(ctx context.Context, projectId, id, status string) error {
	return s.service.UpdateServiceStatus(ctx, projectId, id, status)
}
func (s stores) UpdateServiceAfterDeploy(ctx context.Context, projectId, id, status, versionId string) error {
	return s.service.UpdateServiceAfterDeploy(ctx, projectId, id, status, versionId)
}

func (s stores) CreateDeployment(ctx context.Context, projectId string, deployment model.Deployment) error {
	return s.deployment.CreateDeployment(ctx, projectId, deployment)
}
func (s stores) CompleteDeployment(ctx context.Context, projectId string, id string, status string, message string) (bool, error) {
	return s.deployment.CompleteDeployment(ctx, projectId, id, status, message)
}
func (s stores) ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.Deployment], error) {
	return s.deployment.ListDeployments(ctx, projectId, applicationId, status, search, dateFrom, dateTo, page, perPage)
}
func (s stores) Deployment(ctx context.Context, projectId string, id string) (model.Deployment, error) {
	return s.deployment.Deployment(ctx, projectId, id)
}
func (s stores) CancelDeployment(ctx context.Context, projectId string, id string) (bool, error) {
	return s.deployment.CancelDeployment(ctx, projectId, id)
}
func (s stores) BeginDeployment(ctx context.Context, projectId string, id string) (bool, error) {
	return s.deployment.BeginDeployment(ctx, projectId, id)
}
func (s stores) HasActiveDeployment(ctx context.Context, projectId string, serviceId string) (bool, error) {
	return s.deployment.HasActiveDeployment(ctx, projectId, serviceId)
}

func (s stores) GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error) {
	return s.gateway.GatewayConfig(ctx, applicationId)
}

func (s stores) ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error) {
	return s.gateway.ListGatewayApplications(ctx, projectId)
}
func (s stores) UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error {
	return s.gateway.UpsertGatewayConfig(ctx, cfg)
}

func (s stores) ReplaceGatewayVersionBindings(ctx context.Context, applicationId string, bindings []model.GatewayVersionBinding) error {
	return s.gateway.ReplaceGatewayVersionBindings(ctx, applicationId, bindings)
}
