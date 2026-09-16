package applicationsvc

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// stores composes only the application family's persistence ports.
type stores struct {
	project     repository.ProjectReader
	application repository.ApplicationStore
	service     repository.ServiceReader
}

func (s stores) ListServicesByApplication(ctx context.Context, projectId, applicationId string) ([]model.Service, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListServicesByApplication(ctx, projectId, applicationId)
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

func (s stores) ApplicationByProjectAndName(ctx context.Context, projectId string, name string) (model.Application, error) {
	return s.application.ApplicationByProjectAndName(ctx, projectId, name)
}

func (s stores) ApplicationByProjectAndCode(ctx context.Context, projectId string, code string) (model.Application, error) {
	return s.application.ApplicationByProjectAndCode(ctx, projectId, code)
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

func (s stores) VersionComponent(ctx context.Context, projectId string, id string) (model.VersionComponent, error) {
	return s.application.VersionComponent(ctx, projectId, id)
}

func (s stores) ReplaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error {
	return s.application.ReplaceVersionComponents(ctx, projectId, versionId, components)
}

func (s stores) CreateVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.CreateVersionComponent(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentBasic(ctx context.Context, projectId string, component model.VersionComponent, oldName string) error {
	return s.application.UpdateVersionComponentBasic(ctx, projectId, component, oldName)
}

func (s stores) UpdateVersionComponentRuntime(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentRuntime(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentEndpoints(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentEndpoints(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentEnv(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentEnv(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentMounts(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentMounts(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentDependencies(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentDependencies(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentAdvanced(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentAdvanced(ctx, projectId, component)
}

func (s stores) UpdateVersionComponentDevices(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentDevices(ctx, projectId, component)
}

func (s stores) DeleteVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error {
	return s.application.DeleteVersionComponent(ctx, projectId, component)
}

func (s stores) CreateVersionWithVersionComponents(ctx context.Context, projectId string, version model.Version, components []model.VersionComponent) error {
	return s.application.CreateVersionWithVersionComponents(ctx, projectId, version, components)
}
