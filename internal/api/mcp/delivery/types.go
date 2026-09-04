// Package delivery exposes Pomelo Orbit deployment use cases as MCP tools.
package delivery

import (
	"context"
	"time"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
	projectsvc "github.com/leoninew/pomelo-orbit/internal/application/project/usecase"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routesvc "github.com/leoninew/pomelo-orbit/internal/application/route/usecase"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// Dependencies is the MCP adapter's complete application boundary. The
// concrete services are composed in bootstrap; tool handlers never see stores
// or infrastructure implementations.
type Dependencies struct {
	ActorUserId        string
	ActorAuthenticator ActorAuthenticator
	Project            ProjectService
	Application        ApplicationService
	Service            ServiceService
	Deployment         DeploymentService
	Environment        EnvironmentService
	Gateway            GatewayService
	Route              RouteService
}

// ActorAuthenticator validates the configured stdio credential before every
// tool invocation. Fixed in-process callers supply ActorUserId directly.
type ActorAuthenticator func(context.Context) (string, error)

type ProjectService interface {
	ListByMember(context.Context, string) ([]model.Project, error)
}

type ApplicationService interface {
	ListApplications(context.Context, string, *string, int, int, string, string) (repository.Page[model.Application], error)
	CreateApplication(context.Context, string, applicationdto.ApplicationCreateInput) (model.Application, error)
	ApplicationForUser(context.Context, string, string) (model.Application, error)
	ListVersions(context.Context, string, string) ([]applicationdto.VersionView, error)
	VersionForUser(context.Context, string, string) (applicationdto.VersionView, error)
	CreateVersion(context.Context, string, applicationdto.VersionCreateInput) (applicationdto.VersionView, error)
	UpdateVersion(context.Context, string, string, applicationdto.VersionUpdateInput) (applicationdto.VersionView, error)
	VersionComponentForUser(context.Context, string, string, string) (model.VersionComponent, error)
	CreateVersionComponent(context.Context, string, string, applicationdto.VersionComponentInput) (model.VersionComponent, error)
	UpdateVersionComponentBasic(context.Context, string, string, string, applicationdto.VersionComponentBasicUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentRuntime(context.Context, string, string, string, applicationdto.VersionComponentRuntimeUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentEndpoints(context.Context, string, string, string, applicationdto.VersionComponentEndpointsUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentEnv(context.Context, string, string, string, applicationdto.VersionComponentEnvUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentMounts(context.Context, string, string, string, applicationdto.VersionComponentMountsUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentDependencies(context.Context, string, string, string, applicationdto.VersionComponentDependenciesUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentAdvanced(context.Context, string, string, string, applicationdto.VersionComponentAdvancedUpdateInput) (model.VersionComponent, error)
	UpdateVersionComponentDevices(context.Context, string, string, string, applicationdto.VersionComponentDevicesUpdateInput) (model.VersionComponent, error)
	PublishVersion(context.Context, string, string) (applicationdto.VersionView, error)
	DeleteVersion(context.Context, string, string) error
}

type ServiceService interface {
	ListServicesByApplication(context.Context, string, string) ([]model.Service, error)
	CreateService(context.Context, string, servicedto.ServiceCreateInput) (servicedto.ServiceView, error)
	UpdateServiceComponentOverlay(context.Context, string, string, string, servicedto.ServiceComponentOverlayInput) (model.ServiceComponent, error)
	UpdateServiceEnv(context.Context, string, string, servicedto.ServiceEnvUpdateInput) (servicedto.ServiceView, error)
	UpdateServiceBasic(context.Context, string, string, servicedto.ServiceBasicUpdateInput) (servicedto.ServiceView, error)
}

type DeploymentService interface {
	DeleteApplication(context.Context, string, string) error
	PreviewService(context.Context, string, string, deploymentdto.PreviewComposeInput) (string, error)
	DeployService(context.Context, string, string, deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error)
	StopApplication(context.Context, string, string, deploymentdto.ServiceTargetInput) (string, error)
	RestartApplication(context.Context, string, string, deploymentdto.ServiceTargetInput) (string, error)
	DeploymentForUser(context.Context, string, string) (model.Deployment, error)
	DeploymentLog(context.Context, string, string, int) (deploymentdto.DeploymentLog, error)
	WaitDeployment(context.Context, string, string, *time.Duration) (deploymentdto.DeploymentWaitResult, error)
	ResolveRuntimeTarget(context.Context, string, string, string, bool) (deploymentdto.RuntimeTarget, error)
	RuntimeDoctor(context.Context, *deploymentdto.RuntimeTarget) (map[string]any, error)
	RuntimeComposeConfig(context.Context, deploymentdto.RuntimeTarget) (deploymentdto.RuntimeTextResult, error)
	RuntimeComposePS(context.Context, deploymentdto.RuntimeTarget) (deploymentdto.RuntimeComposePSResult, error)
	RuntimeComposeLogs(context.Context, deploymentdto.RuntimeTarget, int, string, []string) (deploymentdto.RuntimeTextResult, error)
	RuntimeContainerInspect(context.Context, deploymentdto.RuntimeTarget, string) (deploymentdto.RuntimeInspectResult, error)
	RuntimeNetworkInspect(context.Context, deploymentdto.RuntimeTarget, string) (deploymentdto.RuntimeInspectResult, error)
	RuntimeHTTPProbe(context.Context, deploymentdto.RuntimeTarget, string, int, string) (deploymentdto.RuntimeTextResult, error)
	VerifyDeployment(context.Context, string, string, string, deploymentdto.DeploymentVerificationInput) (deploymentdto.DeploymentVerificationResult, error)
}

type EnvironmentService interface {
	EnvironmentForUser(context.Context, string, string) (model.Environment, error)
	UpdateForUser(context.Context, string, string, environmentdto.UpdateInput) (model.Environment, error)
	ProbeForUser(context.Context, string, string) (model.Environment, error)
}

type GatewayService interface {
	ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error)
	CreateGateway(context.Context, string, gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error)
	GatewayForUser(context.Context, string, string) (gatewaydto.GatewayView, error)
	UpdateGateway(context.Context, string, string, gatewaydto.GatewayUpdateInput) (gatewaydto.GatewayView, error)
	ProvisionGateway(context.Context, string, gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error)
}

type RouteService interface {
	ListRoutes(context.Context, string, string, int, int, string) (repository.Page[model.Route], error)
	CreateRoute(context.Context, string, string, routedto.RouteCreateInput) (model.Route, error)
	RouteForUser(context.Context, string, string) (model.Route, error)
	UpdateRoute(context.Context, string, string, routedto.RouteUpdateInput) (model.Route, error)
	EnableRoute(context.Context, string, string) (model.Route, error)
	DisableRoute(context.Context, string, string) (model.Route, error)
	PreviewRouteSync(context.Context, string, string, []routedto.RouteSyncChange) (routedto.RouteSyncPreview, error)
	ConfirmRouteSync(context.Context, string, string, routedto.RouteSyncConfirmInput) error
}

var (
	_ ProjectService     = projectsvc.Service{}
	_ ApplicationService = applicationsvc.Service{}
	_ ServiceService     = servicesvc.Service{}
	_ DeploymentService  = deploymentsvc.Service{}
	_ EnvironmentService = environmentsvc.Service{}
	_ GatewayService     = gatewaysvc.Service{}
	_ RouteService       = routesvc.Service{}
)
