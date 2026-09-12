package gatewaysvc

import (
	"context"
	"strings"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

// ProvisionGateway is an idempotent resource-preparation operation. It never
// publishes a Version, creates a Deployment, waits for runtime state, or
// changes an existing Service binding. Missing Gateway resources are not created.
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
	if gateways.Total == 0 {
		return gatewaydto.ProvisionGatewayResult{}, apperror.NewWithCode(apperror.KindValidation, "gateway_not_ready", "Gateway is not ready")
	}
	result := gatewaydto.ProvisionGatewayResult{Gateway: gateways.Items[0], Steps: []string{"Reused Gateway resources"}}
	if result.Gateway.DefaultService == nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.NewWithCode(apperror.KindValidation, "gateway_not_ready", "Gateway default service is not ready")
	}
	result.Service = *result.Gateway.DefaultService
	result.Steps = append(result.Steps, "Reused Gateway Service")
	return result, nil
}
