package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// ApplicationReader provides the application lookup required by downstream domains.
type ApplicationReader interface {
	Application(ctx context.Context, id string) (model.Application, error)
}

// ApplicationStore persists applications, versions, and component specifications.
type ApplicationStore interface {
	ApplicationReader
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (Page[model.Application], error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error
	ListVersions(ctx context.Context, applicationId string) ([]model.Version, error)
	ListVersionsPage(ctx context.Context, applicationId string, page int, perPage int, search string) (Page[model.Version], error)
	Version(ctx context.Context, id string) (model.Version, error)
	CreateVersion(ctx context.Context, version model.Version) error
	UpdateVersion(ctx context.Context, version model.Version) error
	DeleteVersion(ctx context.Context, id string) error
	CountVersionRuntimeRefs(ctx context.Context, versionId string) (int, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	VersionComponent(ctx context.Context, id string) (model.VersionComponent, error)
	ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error
	CreateVersionComponent(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentBasic(ctx context.Context, component model.VersionComponent, oldName string) error
	UpdateVersionComponentRuntime(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentPorts(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentEnv(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentMounts(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentDependencies(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentAdvanced(ctx context.Context, component model.VersionComponent) error
	UpdateVersionComponentDevices(ctx context.Context, component model.VersionComponent) error
	DeleteVersionComponent(ctx context.Context, component model.VersionComponent) error
	CreateVersionWithVersionComponents(ctx context.Context, version model.Version, components []model.VersionComponent) error
}
