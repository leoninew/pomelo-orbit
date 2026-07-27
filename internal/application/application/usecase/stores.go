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

func (s stores) ListServicesByApplication(ctx context.Context, applicationID string) ([]model.Service, error) {
	if s.service == nil {
		return nil, nil
	}
	return s.service.ListServicesByApplication(ctx, applicationID)
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}

func (s stores) IsProjectMember(ctx context.Context, projectID string, userID string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectID, userID)
}

func (s stores) ListApplications(ctx context.Context, projectID *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	return s.application.ListApplications(ctx, projectID, page, perPage, search, kind)
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

func (s stores) ListVersions(ctx context.Context, applicationID string) ([]model.Version, error) {
	return s.application.ListVersions(ctx, applicationID)
}

func (s stores) ListVersionsPage(ctx context.Context, applicationID string, page int, perPage int, search string) (repository.Page[model.Version], error) {
	return s.application.ListVersionsPage(ctx, applicationID, page, perPage, search)
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

func (s stores) CountVersionRuntimeRefs(ctx context.Context, versionID string) (int, error) {
	return s.application.CountVersionRuntimeRefs(ctx, versionID)
}

func (s stores) VersionComponentsByVersion(ctx context.Context, versionID string) ([]model.VersionComponent, error) {
	return s.application.VersionComponentsByVersion(ctx, versionID)
}

func (s stores) VersionExposesByVersion(ctx context.Context, versionID string) ([]model.VersionExpose, error) {
	return s.application.VersionExposesByVersion(ctx, versionID)
}

func (s stores) ReplaceVersionComponents(ctx context.Context, versionID string, components []model.VersionComponent) error {
	return s.application.ReplaceVersionComponents(ctx, versionID, components)
}

func (s stores) ReplaceVersionExposes(ctx context.Context, versionID string, exposes []model.VersionExpose) error {
	return s.application.ReplaceVersionExposes(ctx, versionID, exposes)
}

func (s stores) CreateVersionWithVersionComponentsAndExposes(ctx context.Context, version model.Version, components []model.VersionComponent, exposes []model.VersionExpose) error {
	return s.application.CreateVersionWithVersionComponentsAndExposes(ctx, version, components, exposes)
}
