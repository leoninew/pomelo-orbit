package cdhandler

import transportresponse "backend/internal/transport/http/response"

type ApplicationResp struct {
	Id              string  `json:"id"`
	ProjectId       *string `json:"project_id"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	ImagePullPolicy string  `json:"image_pull_policy"`
	Status          string  `json:"status"`
	RouteManaged    bool    `json:"route_managed"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type ApplicationCreateReq struct {
	Name            string `json:"name"`
	Code            string `json:"code"`
	ImagePullPolicy string `json:"image_pull_policy"`
	RouteManaged    bool   `json:"route_managed"`
}

type ApplicationUpdateReq struct {
	Name            *string `json:"name"`
	Code            *string `json:"code"`
	ImagePullPolicy *string `json:"image_pull_policy"`
	RouteManaged    *bool   `json:"route_managed"`
}

type ApplicationDeployReq struct {
	ForceRecreate bool `json:"force_recreate"`
}

type DeploymentActionResp struct {
	DeploymentId string `json:"deployment_id"`
}

type DeploymentLogsResp struct {
	Logs       string `json:"logs"`
	Offset     int    `json:"offset"`
	IsComplete bool   `json:"is_complete"`
	Status     string `json:"status"`
}

type DeploymentContainerLogsResp struct {
	Logs                string `json:"logs"`
	Source              string `json:"source"`
	IsRealtimeSupported bool   `json:"is_realtime_supported"`
}

type DeploymentResp struct {
	Id                       string  `json:"id"`
	ProjectId                *string `json:"project_id"`
	ApplicationId            *string `json:"application_id"`
	ApplicationName          string  `json:"application_name"`
	OperationType            string  `json:"operation_type"`
	TriggerType              string  `json:"trigger_type"`
	CommandText              string  `json:"command_text"`
	Status                   string  `json:"status"`
	StartedAt                string  `json:"started_at"`
	FinishedAt               *string `json:"finished_at"`
	DurationMs               *int    `json:"duration_ms"`
	LogText                  *string `json:"log_text"`
	ErrorMessage             *string `json:"error_message"`
	IsRollback               bool    `json:"is_rollback"`
	RollbackFromDeploymentId *string `json:"rollback_from_deployment_id"`
}

type RouteResp struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Domain       string `json:"domain"`
	PathPrefix   string `json:"path_prefix"`
	TargetURL    string `json:"target_url"`
	Enabled      bool   `json:"enabled"`
	HTTPSEnabled bool   `json:"https_enabled"`
	CertType     string `json:"cert_type"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type RouteCreateReq struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	PathPrefix string `json:"path_prefix"`
	TargetURL  string `json:"target_url"`
	Enabled    bool   `json:"enabled"`
}

type RouteUpdateReq struct {
	Name       *string `json:"name"`
	Domain     *string `json:"domain"`
	PathPrefix *string `json:"path_prefix"`
	TargetURL  *string `json:"target_url"`
	Enabled    *bool   `json:"enabled"`
}

type RouteEnableResp struct {
	Message string `json:"message"`
}

type RouteDisableResp struct {
	Message string `json:"message"`
}

type RouteSyncResp struct {
	Message string `json:"message"`
}

type TraefikRouterResp struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Entrypoints []string `json:"entrypoints"`
	TLS         bool     `json:"tls"`
}

type TraefikConfigResp struct {
	DashboardDomain string `json:"dashboard_domain"`
	HTTPSEnabled    bool   `json:"https_enabled"`
}

type TraefikRouteListResp struct {
	Items []TraefikRouterResp `json:"items"`
	Total int                 `json:"total"`
}

type ConfigFileReq struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ApplicationExportConfigFileResp struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ApplicationStopReq struct {
	RemoveVolumes bool `json:"remove_volumes"`
}

type ApplicationFileContentResp struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

type ApplicationStatusResp struct {
	Status string `json:"status"`
}

type ApplicationLogsResp struct {
	Logs string `json:"logs"`
}

type ApplicationComposePreviewResp struct {
	ComposeYAML string `json:"compose_yaml"`
}

type ConfigFileResp struct {
	Id        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type ApplicationRouteReq struct {
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
}

type ApplicationRouteResp struct {
	Id          string `json:"id"`
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ApplicationExportRouteResp struct {
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
}

type ApplicationServiceConfigUpdateReq struct {
	Image *string `json:"image"`
}

type ComposeServiceResp struct {
	ServiceName   string `json:"service_name"`
	DefaultDomain string `json:"default_domain"`
	DefaultPort   int    `json:"default_port"`
}

type ConfigFileListResp = transportresponse.ListResp[ConfigFileResp]
type ApplicationRouteListResp = transportresponse.ListResp[ApplicationRouteResp]
type ComposeServiceListResp = transportresponse.ListResp[ComposeServiceResp]

type ApplicationServiceConfigResp struct {
	ServiceName   string  `json:"service_name"`
	DefaultDomain string  `json:"default_domain"`
	DefaultPort   int     `json:"default_port"`
	BaseImage     *string `json:"base_image"`
	Image         *string `json:"image"`
	ConfigId      *string `json:"config_id"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
}

type ApplicationServiceConfigListResp = transportresponse.ListResp[ApplicationServiceConfigResp]

type ApplicationExportResp struct {
	Version         string                               `json:"version"`
	Name            string                               `json:"name"`
	Code            string                               `json:"code"`
	ImagePullPolicy string                               `json:"image_pull_policy"`
	RouteManaged    bool                                 `json:"route_managed"`
	ConfigFiles     []ApplicationExportConfigFileResp    `json:"config_files"`
	ServiceConfigs  []ApplicationServiceConfigExportResp `json:"service_configs"`
	Routes          []ApplicationExportRouteResp         `json:"routes"`
}

type ApplicationImportReq struct {
	Version         string                              `json:"version"`
	Name            string                              `json:"name"`
	Code            string                              `json:"code"`
	ImagePullPolicy string                              `json:"image_pull_policy"`
	RouteManaged    bool                                `json:"route_managed"`
	ConfigFiles     []ConfigFileReq                     `json:"config_files"`
	ServiceConfigs  []ApplicationServiceConfigImportReq `json:"service_configs"`
	Routes          []ApplicationRouteReq               `json:"routes"`
}

type ApplicationServiceConfigImportReq struct {
	ServiceName string  `json:"service_name"`
	Image       *string `json:"image"`
	Environment *string `json:"environment"`
	Volumes     *string `json:"volumes"`
}

type ApplicationServiceConfigExportResp struct {
	ServiceName string  `json:"service_name"`
	Image       *string `json:"image"`
	Environment *string `json:"environment"`
	Volumes     *string `json:"volumes"`
}
