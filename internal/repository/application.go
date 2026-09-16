package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// ApplicationReader provides the application lookup required by downstream domains.
type ApplicationReader interface {
	Application(ctx context.Context, projectId string, id string) (model.Application, error)
}

// ApplicationStore persists applications, versions, and component specifications.
type ApplicationStore interface {
	ApplicationReader
	ListApplications(ctx context.Context, projectId string, page int, perPage int, search string, kind string) (Page[model.Application], error)
	ApplicationByName(ctx context.Context, projectId string, name string) (model.Application, error)
	ApplicationByProjectAndName(ctx context.Context, projectId string, name string) (model.Application, error)
	ApplicationByProjectAndCode(ctx context.Context, projectId string, code string) (model.Application, error)
	ApplicationByCode(ctx context.Context, projectId string, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, projectId string, app model.Application) error
	DeleteApplication(ctx context.Context, projectId string, id string) error
	DeleteGatewayApplication(ctx context.Context, projectId string, id string) error
	ListVersions(ctx context.Context, projectId string, applicationId string) ([]model.Version, error)
	LatestVersionByApplication(ctx context.Context, projectId string, applicationId string) (model.Version, error)
	ListVersionsPage(ctx context.Context, projectId string, applicationId string, page int, perPage int, search string) (Page[model.Version], error)
	Version(ctx context.Context, projectId string, id string) (model.Version, error)
	CreateVersion(ctx context.Context, projectId string, version model.Version) error
	UpdateVersion(ctx context.Context, projectId string, version model.Version) error
	DeleteVersion(ctx context.Context, projectId string, id string) error
	CountVersionRuntimeRefs(ctx context.Context, projectId string, versionId string) (int, error)
	VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error)
	VersionComponent(ctx context.Context, projectId string, id string) (model.VersionComponent, error)
	ReplaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error
	CreateVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentBasic(ctx context.Context, projectId string, component model.VersionComponent, oldName string) error
	UpdateVersionComponentRuntime(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentEndpoints(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentEnv(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentMounts(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentDependencies(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentAdvanced(ctx context.Context, projectId string, component model.VersionComponent) error
	UpdateVersionComponentDevices(ctx context.Context, projectId string, component model.VersionComponent) error
	DeleteVersionComponent(ctx context.Context, projectId string, component model.VersionComponent) error
	CreateVersionWithVersionComponents(ctx context.Context, projectId string, version model.Version, components []model.VersionComponent) error
}
