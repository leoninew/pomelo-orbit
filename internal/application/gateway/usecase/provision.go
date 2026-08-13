package gatewaysvc

import (
	"context"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	managedGatewayCode = "traefik"
	managedNetworkName = "traefik"
)

// ProvisionGateway owns the high-level managed Gateway workflow. The MCP
// adapter calls this one use case rather than sequencing lower-level tools.
func (s Service) ProvisionGateway(ctx context.Context, userId string, input gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error) {
	if s.deployer == nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindInternal, "gateway deployment service is not configured")
	}
	projectId := strings.TrimSpace(input.ProjectId)
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if projectId == "" || instanceKey == "" {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "project_id and instance_key are required")
	}

	gateways, err := s.ListGateways(ctx, userId, projectId, 1, 10000, "")
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	matches := make([]gatewaydto.GatewayView, 0, 1)
	for _, item := range gateways.Items {
		if item.Application.Code == managedGatewayCode {
			matches = append(matches, item)
		}
	}
	if len(matches) > 1 {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "multiple traefik Gateways exist in the Project")
	}

	result := gatewaydto.ProvisionGatewayResult{}
	if len(matches) == 1 {
		result.Gateway = matches[0]
		result.Steps = append(result.Steps, "Reused Gateway Application")
	} else {
		image := "traefik:3.6"
		gateway, err := s.CreateGateway(ctx, userId, gatewaydto.GatewayCreateInput{ProjectId: projectId, Code: managedGatewayCode, Name: "Traefik", RestApiUrl: "http://localhost:8080", BaseDomain: "lvh.me", InitialComponentImage: &image, InitialComponentPullPolicy: "missing"})
		if err != nil {
			return gatewaydto.ProvisionGatewayResult{}, err
		}
		result.Gateway, result.GatewayCreated = gateway, true
		result.Steps = append(result.Steps, "Created Gateway Application through Orbit")
	}

	services, err := s.service.ListServicesByApplication(ctx, result.Gateway.Application.Id)
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway services", err)
	}
	serviceMatches := make([]model.Service, 0, 1)
	for _, item := range services {
		if item.InstanceKey == instanceKey {
			serviceMatches = append(serviceMatches, item)
		}
	}
	if len(serviceMatches) > 1 {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "multiple Gateway Services use the requested instance_key")
	}
	var existing *model.Service
	if len(serviceMatches) == 1 {
		existing = &serviceMatches[0]
	}

	versions, err := s.application.ListVersions(ctx, result.Gateway.Application.Id)
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.Wrap(apperror.KindInternal, "Failed to list gateway versions", err)
	}
	version, err := selectGatewayVersion(versions, existing)
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	if version.Status != status.VersionStatusPublished {
		version.Status = status.VersionStatusPublished
		if err := s.application.UpdateVersion(ctx, version); err != nil {
			return gatewaydto.ProvisionGatewayResult{}, apperror.Wrap(apperror.KindInternal, "Failed to publish Gateway Version", err)
		}
		result.VersionPublished = true
		result.Steps = append(result.Steps, "Published Gateway Version")
	}
	result.Version = version

	if existing == nil {
		created, err := s.serviceCommands.CreateService(ctx, userId, servicedto.ServiceCreateInput{ApplicationId: result.Gateway.Application.Id, VersionId: version.Id, InstanceKey: instanceKey, Code: result.Gateway.Application.Code + "-" + instanceKey})
		if err != nil {
			return gatewaydto.ProvisionGatewayResult{}, err
		}
		result.Service, result.ServiceCreated = created.Service, true
		result.Steps = append(result.Steps, "Created Gateway Service")
	} else {
		result.Service = *existing
		if existing.VersionId != version.Id {
			updated, err := s.serviceCommands.UpdateServiceBasic(ctx, userId, existing.Id, servicedto.ServiceBasicUpdateInput{VersionId: version.Id, InstanceKey: instanceKey})
			if err != nil {
				return gatewaydto.ProvisionGatewayResult{}, err
			}
			result.Service = updated.Service
		}
		result.Steps = append(result.Steps, "Reused Gateway Service")
	}

	deployment, err := s.deployer.DeployService(ctx, userId, result.Service.Id, deploymentdto.DeployServiceInput{ForceRecreate: input.ForceRecreate})
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	result.Steps = append(result.Steps, "Created Gateway Deployment through Orbit")
	waited, err := s.deployer.WaitDeployment(ctx, userId, deployment.DeploymentId, input.Timeout())
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	result.Deployment, result.TimedOut = waited.Deployment, waited.TimedOut
	if waited.TimedOut || waited.Deployment.Status != status.WorkStatusRanToCompletion {
		result.Network = gatewaydto.GatewayNetwork{Name: managedNetworkName, Ready: false, Status: "not_checked"}
		result.Steps = append(result.Steps, "Gateway Deployment did not reach a successful terminal state")
		return result, nil
	}
	result.Steps = append(result.Steps, "Gateway Deployment reached a successful terminal state")

	network, err := s.deployer.ExternalNetworkInspect(ctx, managedNetworkName)
	if err != nil {
		result.Network = gatewaydto.GatewayNetwork{Name: managedNetworkName, Ready: false, Status: "unavailable"}
		result.Steps = append(result.Steps, "External traefik network is unavailable")
		return result, nil
	}
	result.Network = gatewaydto.GatewayNetwork{Name: network.Name, Driver: network.Driver, Ready: network.Driver == "bridge"}
	if result.Network.Ready {
		result.Network.Status = "ready"
		result.Steps = append(result.Steps, "Confirmed external traefik bridge network")
	} else {
		result.Network.Status = "unsupported_driver"
		result.Steps = append(result.Steps, "External traefik network is not a bridge network")
	}
	result.Ready = result.Network.Ready
	return result, nil
}

func selectGatewayVersion(versions []model.Version, service *model.Service) (model.Version, error) {
	unpublished := make([]model.Version, 0, 1)
	for _, version := range versions {
		if version.Status == status.VersionStatusUnpublished {
			unpublished = append(unpublished, version)
		}
	}
	if len(unpublished) == 1 {
		return unpublished[0], nil
	}
	if len(unpublished) > 1 {
		return model.Version{}, apperror.New(apperror.KindValidation, "multiple unpublished Gateway Versions exist")
	}
	if service != nil {
		for _, version := range versions {
			if version.Id == service.VersionId {
				return version, nil
			}
		}
		return model.Version{}, apperror.New(apperror.KindValidation, "existing Gateway Service does not reference a listed Version")
	}
	if len(versions) == 1 {
		return versions[0], nil
	}
	return model.Version{}, apperror.New(apperror.KindValidation, "unable to select a unique Gateway Version")
}
