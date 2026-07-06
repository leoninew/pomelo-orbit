package cdhandler

import apiv1 "backend/internal/transport/http/dto/proto/orbit/api/v1"

type ApplicationResp = apiv1.ApplicationResp
type ApplicationCreateReq = apiv1.ApplicationCreateReq
type ApplicationUpdateReq = apiv1.ApplicationUpdateReq
type ApplicationDeployReq = apiv1.ApplicationDeployReq
type DeploymentActionResp = apiv1.DeploymentActionResp
type DeploymentLogsResp = apiv1.DeploymentLogsResp
type DeploymentContainerLogsResp = apiv1.DeploymentContainerLogsResp
type DeploymentResp = apiv1.DeploymentResp
type RouteResp = apiv1.RouteResp
type RouteCreateReq = apiv1.RouteCreateReq
type RouteUpdateReq = apiv1.RouteUpdateReq
type RouteEnableResp = apiv1.RouteEnableResp
type RouteDisableResp = apiv1.RouteDisableResp
type RouteSyncResp = apiv1.RouteSyncResp
type TraefikRouterResp = apiv1.TraefikRouterResp
type TraefikConfigResp = apiv1.TraefikConfigResp
type TraefikRouteListResp = apiv1.TraefikRouteListResp
type ConfigFileReq = apiv1.ConfigFileReq
type ApplicationExportConfigFileResp = apiv1.ApplicationExportConfigFileResp
type ApplicationStopReq = apiv1.ApplicationStopReq
type ApplicationFileContentResp = apiv1.ApplicationFileContentResp
type ApplicationStatusResp = apiv1.ApplicationStatusResp
type ApplicationLogsResp = apiv1.ApplicationLogsResp
type ApplicationComposePreviewResp = apiv1.ApplicationComposePreviewResp
type ConfigFileResp = apiv1.ConfigFileResp
type ApplicationRouteReq = apiv1.ApplicationRouteReq
type ApplicationRouteResp = apiv1.ApplicationRouteResp
type ApplicationExportRouteResp = apiv1.ApplicationExportRouteResp
type ApplicationServiceConfigUpdateReq = apiv1.ApplicationServiceConfigUpdateReq
type ComposeServiceResp = apiv1.ComposeServiceResp
type ConfigFileListResp = apiv1.ConfigFileListResp
type ApplicationRouteListResp = apiv1.ApplicationRouteListResp
type ComposeServiceListResp = apiv1.ComposeServiceListResp
type ApplicationServiceConfigResp = apiv1.ApplicationServiceConfigResp
type ApplicationServiceConfigListResp = apiv1.ApplicationServiceConfigListResp
type ApplicationExportResp = apiv1.ApplicationExportResp
type ApplicationImportReq = apiv1.ApplicationImportReq
type ApplicationServiceConfigImportReq = apiv1.ApplicationServiceConfigImportReq
type ApplicationServiceConfigExportResp = apiv1.ApplicationServiceConfigExportResp
type ApplicationPaginatedResp = apiv1.ApplicationPaginatedResp
type DeploymentPaginatedResp = apiv1.DeploymentPaginatedResp
type RoutePaginatedResp = apiv1.RoutePaginatedResp
