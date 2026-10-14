package gatewaysvc

import (
	"context"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

// ProvisionGateway returns existing Gateway resources and their Service binding.
// Missing resources produce gateway_not_ready.
func (s Service) ProvisionGateway(ctx context.Context, userId string, input gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error) {
	projectId := input.ProjectId
	if projectId == "" {
		return gatewaydto.ProvisionGatewayResult{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	gateways, err := s.ListGateways(ctx, userId, projectId, 1, 1, "")
	if err != nil {
		return gatewaydto.ProvisionGatewayResult{}, err
	}
	if gateways.Total == 0 {
		return gatewaydto.ProvisionGatewayResult{}, apperror.NewWithCode(apperror.KindValidation, "gateway_not_ready", "Gateway is not ready")
	}
	result := gatewaydto.ProvisionGatewayResult{Gateway: gateways.Items[0], Steps: []string{"Reused Gateway resources"}}
	if result.Gateway.Service == nil {
		return gatewaydto.ProvisionGatewayResult{}, apperror.NewWithCode(apperror.KindValidation, "gateway_not_ready", "Gateway service is not ready")
	}
	result.Service = *result.Gateway.Service
	result.Steps = append(result.Steps, "Reused Gateway Service")
	return result, nil
}
