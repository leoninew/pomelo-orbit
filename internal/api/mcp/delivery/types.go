// Package delivery exposes Pomelo Orbit deployment use cases as MCP tools.
package delivery

import (
	"context"
	"time"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	gatewaydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/dto"
	gatewaysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/usecase"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// Dependencies is the MCP adapter's complete application boundary. The
// concrete services are composed in bootstrap; tool handlers never see stores
// or infrastructure implementations.
type Dependencies struct {
	ActorUserId     string
	ActorAuthorizer ActorAuthorizer
	Project         ProjectService
	Application     ApplicationService
	Service         ServiceService
	Deployment      DeploymentService
	Gateway         GatewayService
}

// ActorAuthorizer binds a stdio MCP session to an Orbit user on its first
// tool invocation. HTTP MCP supplies ActorUserId during connection setup.
type ActorAuthorizer func(context.Context) (string, error)

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
	DeleteApplication(context.Context, string, string, bool) error
	PreviewService(context.Context, string, string) (string, error)
	DeployService(context.Context, string, string, deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error)
	StopApplication(context.Context, string, string, deploymentdto.ServiceTargetInput) (string, error)
	RestartApplication(context.Context, string, string, deploymentdto.ServiceTargetInput) (string, error)
	DeploymentForUser(context.Context, string, string) (model.Deployment, error)
	DeploymentLog(context.Context, string, string, int) (deploymentdto.DeploymentLog, error)
	WaitDeployment(context.Context, string, string, *time.Duration) (deploymentdto.DeploymentWaitResult, error)
	ResolveRuntimeTarget(context.Context, string, string, string, bool) (deploymentdto.RuntimeTarget, error)
	RuntimeDoctor(context.Context, *deploymentdto.RuntimeTarget, string) (map[string]any, error)
	RuntimeComposeConfig(context.Context, deploymentdto.RuntimeTarget) (deploymentdto.RuntimeTextResult, error)
	RuntimeComposePS(context.Context, deploymentdto.RuntimeTarget) (deploymentdto.RuntimeComposePSResult, error)
	RuntimeComposeLogs(context.Context, deploymentdto.RuntimeTarget, int, string, []string) (deploymentdto.RuntimeTextResult, error)
	RuntimeContainerInspect(context.Context, deploymentdto.RuntimeTarget, string) (deploymentdto.RuntimeInspectResult, error)
	RuntimeNetworkInspect(context.Context, deploymentdto.RuntimeTarget, string) (deploymentdto.RuntimeInspectResult, error)
	RuntimeHTTPProbe(context.Context, deploymentdto.RuntimeTarget, string, int, string) (deploymentdto.RuntimeTextResult, error)
	VerifyDeployment(context.Context, string, string, string, deploymentdto.DeploymentVerificationInput) (deploymentdto.DeploymentVerificationResult, error)
}

type GatewayService interface {
	ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error)
	CreateGateway(context.Context, string, gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error)
	GatewayForUser(context.Context, string, string) (gatewaydto.GatewayView, error)
	UpdateGateway(context.Context, string, string, gatewaydto.GatewayUpdateInput) (gatewaydto.GatewayView, error)
	ProvisionGateway(context.Context, string, gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error)
}

var (
	_ ProjectService     = projectsvc.Service{}
	_ ApplicationService = applicationsvc.Service{}
	_ ServiceService     = servicesvc.Service{}
	_ DeploymentService  = deploymentsvc.Service{}
	_ GatewayService     = gatewaysvc.Service{}
)
