package gatewaysvc

import (
	"context"
	"errors"
	"strings"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// ProvisionGateway is an idempotent resource-preparation operation. It never
// publishes a Version, creates a Deployment, waits for runtime state, or
// changes an existing Service binding.
func (s Service) ProvisionGateway(ctx context.Context, userId string, input gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error) {
	projectID := strings.TrimSpace(input.ProjectId)
	instanceKey := strings.TrimSpace(input.InstanceKey)
	if projectID == "" || instanceKey == "" {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "project_id and instance_key are required")
	}

	gateways, err := s.ListGateways(ctx, userId, projectID, 1, 10000, "")
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
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "multiple managed Gateways exist in the Project")
	}

	result := gatewaydto.ProvisionGatewayResult{}
	if len(matches) == 0 {
		defaults := s.CreateDefaults()
		image, entrypoint, tlsMode := defaults.InitialComponentImage, defaults.DefaultEntrypoint, defaults.TLSMode
		created, err := s.CreateGateway(ctx, userId, gatewaydto.GatewayCreateInput{
			ProjectId:                  projectID,
			Code:                       managedGatewayCode,
			Name:                       managedGatewayName,
			RestApiUrl:                 defaults.RestApiUrl,
			BaseDomain:                 defaults.BaseDomain,
			InitialComponentImage:      &image,
			InitialComponentPullPolicy: defaults.InitialComponentPullPolicy,
			DefaultEntrypoint:          &entrypoint,
			TLSMode:                    &tlsMode,
		})
		if err != nil {
			return gatewaydto.ProvisionGatewayResult{}, err
		}
		result.Gateway, result.GatewayCreated = created, true
		result.Steps = append(result.Steps, "Created Gateway resources")
	} else {
		result.Gateway = matches[0]
		result.Steps = append(result.Steps, "Reused Gateway resources")
	}

	service, err := s.service.ServiceByKey(ctx, result.Gateway.Application.Id, instanceKey)
	if err == nil {
		result.Service = service
		result.Steps = append(result.Steps, "Reused Gateway Service")
		return result, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return gatewaydto.ProvisionGatewayResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load Gateway Service", err)
	}
	if result.Gateway.DefaultService == nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindInternal, "Gateway default service is missing")
	}
	created, err := s.serviceCommands.CreateService(ctx, userId, servicedto.ServiceCreateInput{
		ApplicationId: result.Gateway.Application.Id,
		VersionId:     result.Gateway.DefaultService.VersionId,
		InstanceKey:   instanceKey,
		Code:          result.Gateway.Application.Code + "-" + instanceKey,
	})
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	result.Service, result.ServiceCreated = created.Service, true
	result.Steps = append(result.Steps, "Created Gateway Service")
	return result, nil
}
