package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

type ApplicationCreateInput struct {
	ProjectId string
	Name      string
	Code      string
	Kind      string
}

type ApplicationUpdateInput struct {
	Name *string
	Code *string
}

// VersionCreateInput creates an unpublished version with component specifications.
type VersionCreateInput struct {
	ApplicationId string
	Label         string
	Note          *string
	Components    []VersionComponentInput
}

// VersionUpdateInput updates version metadata.
type VersionUpdateInput struct {
	Label *string
	Note  *string
}

type VersionComponentInput struct {
	Name          string
	Image         string
	Entrypoint    string
	Command       string
	Env           []model.VersionComponentEnv
	Endpoints     []model.VersionComponentEndpoint
	Mounts        []model.VersionComponentMount
	Dependencies  []model.VersionComponentDependency
	Healthcheck   *VersionComponentHealthcheckInput
	Resources     *model.VersionComponentResources
	PullPolicy    string
	RestartPolicy *string
	Tmpfs         []model.VersionComponentTmpfs
	Ulimits       []model.VersionComponentUlimit
	Devices       []model.VersionComponentDeviceRequest
}

type VersionComponentBasicUpdateInput struct {
	Name          string
	Image         string
	Entrypoint    string
	Command       string
	PullPolicy    string
	RestartPolicy *string
}

type VersionComponentRuntimeUpdateInput struct {
	Healthcheck *VersionComponentHealthcheckInput
}

type VersionComponentHealthcheckInput struct {
	TestMode      string
	Test          string
	Interval      *string
	Timeout       *string
	Retries       *int
	StartPeriod   *string
	StartInterval *string
	Disabled      bool
}

type VersionComponentEndpointsUpdateInput struct {
	Endpoints []model.VersionComponentEndpoint
}

type VersionComponentEnvUpdateInput struct {
	Env []model.VersionComponentEnv
}

type VersionComponentMountsUpdateInput struct {
	Mounts []model.VersionComponentMount
}

type VersionComponentDependenciesUpdateInput struct {
	Dependencies []model.VersionComponentDependency
}

type VersionComponentAdvancedUpdateInput struct {
	Resources *model.VersionComponentResources
	Tmpfs     []model.VersionComponentTmpfs
	Ulimits   []model.VersionComponentUlimit
}

type VersionComponentDevicesUpdateInput struct {
	Devices []model.VersionComponentDeviceRequest
}

// VersionView is a version with optional component details for API responses.
type VersionView struct {
	Version    model.Version
	Components []model.VersionComponent
}

// ApplicationImportInput imports an application with an initial version.
type ApplicationImportInput struct {
	ProjectId    string
	Name         string
	Code         string
	Kind         string
	VersionLabel string
	VersionNote  *string
	Components   []VersionComponentInput
}

// ApplicationExport bundles application/version data. Runtime services are
// attached by the HTTP adapter through the service domain.
type ApplicationExport struct {
	Application model.Application
	Versions    []VersionView
	Services    []model.Service
}
