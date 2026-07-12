package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type ApplicationCreateInput struct {
	ProjectId       string
	Name            string
	Code            string
	ImagePullPolicy string
	RouteManaged    bool
}

type ApplicationUpdateInput struct {
	Name            *string
	Code            *string
	ImagePullPolicy *string
	RouteManaged    *bool
}

type ApplicationDeleteInput struct {
	RemoveDir bool
}

type ApplicationDeployInput struct {
	ForceRecreate bool
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

// ConfigFileInput stores an application config file payload.
type ConfigFileInput struct {
	Path    string
	Content string
}

// ApplicationRouteInput stores an application route payload.
type ApplicationRouteInput struct {
	ServiceName string
	Domain      string
	Port        int
}

// ApplicationServiceConfigImportInput stores import payload for service config.
type ApplicationServiceConfigImportInput struct {
	ServiceName string
	Image       *string
	Environment *string
	Volumes     *string
}

// ApplicationImportInput imports an application bundle.
type ApplicationImportInput struct {
	ProjectId         string
	Version           string
	Name              string
	Code              string
	ImagePullPolicy   string
	RouteManaged      bool
	ConfigFiles       []ConfigFileInput
	ServiceConfigs    []ApplicationServiceConfigImportInput
	ApplicationRoutes []ApplicationRouteInput
}

// ApplicationExport bundles application data for handler response.
type ApplicationExport struct {
	Application    model.Application
	ConfigFiles    []model.ApplicationConfigFile
	ServiceConfigs []model.ApplicationServiceConfig
	Routes         []model.ApplicationRoute
}

// ApplicationServiceConfigView is the service-config view used by HTTP responses.
type ApplicationServiceConfigView struct {
	ServiceName   string  `json:"service_name"`
	DefaultDomain string  `json:"default_domain"`
	DefaultPort   int     `json:"default_port"`
	BaseImage     *string `json:"base_image"`
	Image         *string `json:"image"`
	ConfigId      *string `json:"config_id"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
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
