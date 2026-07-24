package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type ApplicationCreateInput struct {
	ProjectId       string
	Name            string
	Code            string
	Kind            string
	ImagePullPolicy string
}

type ApplicationUpdateInput struct {
	Name            *string
	Code            *string
	ImagePullPolicy *string
}

// VersionCreateInput creates an unpublished version with components and exposes.
type VersionCreateInput struct {
	ApplicationId string
	Label         string
	EnvJSON       *string
	Note          *string
	Components    []VersionComponentInput
	Exposes       []VersionExposeInput
}

// VersionUpdateInput updates an unpublished version metadata and optionally components/exposes.
type VersionUpdateInput struct {
	Label      *string
	EnvJSON    *string
	Note       *string
	Components *[]VersionComponentInput
	Exposes    *[]VersionExposeInput
}

type VersionComponentInput struct {
	Name            string
	Image           string
	CommandJSON     *string
	ArgsJSON        *string
	EnvJSON         *string
	PortsJSON       *string
	MountsJSON      *string
	NetworksJSON    *string
	DependsOnJSON   *string
	HealthcheckJSON *string
	ResourcesJSON   *string
	PullPolicy      *string
}

type VersionExposeInput struct {
	ComponentName string
	Protocol      string
	ContainerPort int
	PathPrefix    *string
	Access        string
	ListenPort    *int
}

// VersionView is version plus its components and exposes for API responses.
type VersionView struct {
	Version    model.Version
	Components []model.VersionComponent
	Exposes    []model.VersionExpose
}

// ApplicationImportInput imports an application with an initial version.
type ApplicationImportInput struct {
	ProjectId       string
	Name            string
	Code            string
	Kind            string
	ImagePullPolicy string
	VersionLabel    string
	VersionEnvJSON  *string
	VersionNote     *string
	Components      []VersionComponentInput
	Exposes         []VersionExposeInput
}

// ApplicationExport bundles application/version data. Runtime services are
// attached by the HTTP adapter through the service domain.
type ApplicationExport struct {
	Application model.Application
	Versions    []VersionView
	Services    []model.Service
}
