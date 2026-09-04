package gatewaysvc

import (
	"context"
	"strings"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

// ProvisionGateway is an idempotent resource-preparation operation. It never
// publishes a Version, creates a Deployment, waits for runtime state, or
// changes an existing Service binding.
func (s Service) ProvisionGateway(ctx context.Context, userId string, input gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error) {
	projectID := strings.TrimSpace(input.ProjectId)
	if projectID == "" {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if instanceKey := strings.TrimSpace(input.InstanceKey); instanceKey != "" && instanceKey != "default" {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "a project environment supports only the default gateway service")
	}

	gateways, err := s.ListGateways(ctx, userId, projectID, 1, 1, "")
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	result := gatewaydto.ProvisionGatewayResult{}
	if gateways.Total == 0 {
		project, err := s.project.Project(ctx, projectID)
		if err != nil {
			return gatewaydto.ProvisionGatewayResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
		}
		defaults := s.CreateDefaults()
		image, entrypoint, tlsMode := defaults.InitialComponentImage, defaults.DefaultEntrypoint, defaults.TLSMode
		created, err := s.CreateGateway(ctx, userId, gatewaydto.GatewayCreateInput{
			ProjectId:                  projectID,
			Code:                       managedGatewayCodeForProject(project),
			Name:                       managedGatewayNameForProject(project),
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
		result.Gateway = gateways.Items[0]
		result.Steps = append(result.Steps, "Reused Gateway resources")
	}
	if result.Gateway.DefaultService == nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindInternal, "Gateway default service is missing")
	}
	result.Service = *result.Gateway.DefaultService
	result.Steps = append(result.Steps, "Reused Gateway Service")
	return result, nil
}
