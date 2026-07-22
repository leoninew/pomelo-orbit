package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type ApplicationCreateInput struct {
	ProjectId       string
	Name            string
	Code            string
	ImagePullPolicy string
}

type ApplicationUpdateInput struct {
	Name            *string
	Code            *string
	ImagePullPolicy *string
}

type ApplicationDeleteInput struct {
	RemoveDir bool
}

type ApplicationDeployInput struct {
	VersionId     string
	EnvironmentId string
	InstanceKey   string
	AttachIngress *bool
	ForceRecreate bool
}

type ApplicationServiceTargetInput struct {
	EnvironmentId string
	InstanceKey   string
	ServiceId     string
	RemoveVolumes bool
}

// VersionCreateInput creates an unpublished version with components and exposes.
type VersionCreateInput struct {
	ApplicationId string
	Label         string
	EnvJSON       *string
	Note          *string
	Components    []ComponentInput
	Exposes       []ExposeInput
}

// VersionUpdateInput updates an unpublished version metadata and optionally components/exposes.
type VersionUpdateInput struct {
	Label      *string
	EnvJSON    *string
	Note       *string
	Components *[]ComponentInput
	Exposes    *[]ExposeInput
}

// ComponentInput is the write payload for a version component.
type ComponentInput struct {
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

// ExposeInput is the write payload for a version expose.
type ExposeInput struct {
	ComponentName string
	Protocol      string
	ContainerPort int
	PathPrefix    *string
}

// VersionView is version plus its components and exposes for API responses.
type VersionView struct {
	Version    model.Version
	Components []model.Component
	Exposes    []model.Expose
}

// ServiceView is the runtime binding for an application instance.
type ServiceView struct {
	Service model.Service
}

type EnvironmentCreateInput struct {
	ProjectId   string
	Code        string
	Name        string
	Description *string
}

type EnvironmentUpdateInput struct {
	Name        *string
	Description *string
}

type EnvironmentBindingInput struct {
	ComponentName string
	Protocol      string
	ContainerPort int
	Domains       []string
	Entrypoint    string
	TLSMode       string
	SNIHost       *string
	Note          *string
}

type EnvironmentView struct {
	Environment model.Environment
	Bindings    []model.EnvironmentBinding
}

type ApplicationDeployDispatchInput struct {
	ApplicationID string
	DeploymentID  string
	ForceRecreate bool
}

type ApplicationRestartDispatchInput struct {
	ApplicationID string
	DeploymentID  string
}

type ApplicationStopDispatchInput struct {
	ApplicationID string
	DeploymentID  string
	RemoveVolumes bool
}

type DeploymentListInput struct {
	ProjectId     string
	ApplicationId string
	Status        string
	Search        string
	DateFrom      string
	DateTo        string
	Page          int
	PerPage       int
}

type DeploymentLog struct {
	Logs       string
	Offset     int
	IsComplete bool
	Status     string
}

type DeploymentContainerLog struct {
	Logs                string
	Source              string
	IsRealtimeSupported bool
}

// ApplicationImportInput imports an application with an initial version.
type ApplicationImportInput struct {
	ProjectId       string
	Name            string
	Code            string
	ImagePullPolicy string
	VersionLabel    string
	VersionEnvJSON  *string
	VersionNote     *string
	Components      []ComponentInput
	Exposes         []ExposeInput
}

// ApplicationExport bundles application data for handler response.
type ApplicationExport struct {
	Application model.Application
	Versions    []VersionView
	Services    []model.Service
}

// RouteCreateInput creates a project route.
type RouteCreateInput struct {
	Name       string
	Domain     string
	PathPrefix string
	TargetURL  string
	Enabled    bool
}

// RouteUpdateInput updates a project route.
type RouteUpdateInput struct {
	Name       *string
	Domain     *string
	PathPrefix *string
	TargetURL  *string
	Enabled    *bool
}

// TraefikRouterResp describes a router returned by the Traefik integration.
type TraefikRouterResp struct {
	Name        string
	Provider    string
	Status      string
	Rule        string
	Service     string
	Entrypoints []string
	TLS         bool
}

// TraefikConfigResp reports the dashboard route state.
type TraefikConfigResp struct {
	DashboardDomain string `json:"dashboard_domain"`
	HTTPSEnabled    bool   `json:"https_enabled"`
}

// TraefikRouteListResp wraps the Traefik router list.
type TraefikRouteListResp struct {
	Items []TraefikRouterResp `json:"items"`
	Total int                 `json:"total"`
}

// DeployOptionsJSON is stored on Deployment.options_json.
type DeployOptionsJSON struct {
	ForceRecreate bool   `json:"force_recreate,omitempty"`
	InstanceKey   string `json:"instance_key,omitempty"`
	AttachIngress bool   `json:"attach_ingress,omitempty"`
	RemoveVolumes bool   `json:"remove_volumes,omitempty"`
}
