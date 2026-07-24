package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// ApplicationReader provides the application lookup required by downstream domains.
type ApplicationReader interface {
	Application(ctx context.Context, id string) (model.Application, error)
}

// ApplicationStore persists applications, versions, components, and exposes.
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
	VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error)
	ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error
	ReplaceVersionExposes(ctx context.Context, versionId string, exposes []model.VersionExpose) error
	CreateVersionWithVersionComponentsAndExposes(ctx context.Context, version model.Version, components []model.VersionComponent, exposes []model.VersionExpose) error
}
