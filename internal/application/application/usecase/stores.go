package applicationsvc

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// stores composes only the application family's persistence ports.
type stores struct {
	project     repository.ProjectReader
	application repository.ApplicationStore
	service     repository.ServiceReader
}

func (s stores) ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListServicesByApplication(ctx, applicationId)
}

func (s stores) CountServiceExposesByVersionComponent(ctx context.Context, versionId string, componentName string) (int, error) {
	if s.service == nil {
		return 0, nil
	}
	return s.service.CountServiceExposesByVersionComponent(ctx, versionId, componentName)
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

func (s stores) VersionComponent(ctx context.Context, id string) (model.VersionComponent, error) {
	return s.application.VersionComponent(ctx, id)
}

func (s stores) ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error {
	return s.application.ReplaceVersionComponents(ctx, versionId, components)
}

func (s stores) CreateVersionComponent(ctx context.Context, component model.VersionComponent) error {
	return s.application.CreateVersionComponent(ctx, component)
}

func (s stores) UpdateVersionComponentBasic(ctx context.Context, component model.VersionComponent, oldName string) error {
	return s.application.UpdateVersionComponentBasic(ctx, component, oldName)
}

func (s stores) UpdateVersionComponentRuntime(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentRuntime(ctx, component)
}

func (s stores) UpdateVersionComponentPorts(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentPorts(ctx, component)
}

func (s stores) UpdateVersionComponentEnv(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentEnv(ctx, component)
}

func (s stores) UpdateVersionComponentMounts(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentMounts(ctx, component)
}

func (s stores) UpdateVersionComponentDependencies(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentDependencies(ctx, component)
}

func (s stores) UpdateVersionComponentAdvanced(ctx context.Context, component model.VersionComponent) error {
	return s.application.UpdateVersionComponentAdvanced(ctx, component)
}

func (s stores) DeleteVersionComponent(ctx context.Context, component model.VersionComponent) error {
	return s.application.DeleteVersionComponent(ctx, component)
}

func (s stores) CreateVersionWithVersionComponents(ctx context.Context, version model.Version, components []model.VersionComponent) error {
	return s.application.CreateVersionWithVersionComponents(ctx, version, components)
}
