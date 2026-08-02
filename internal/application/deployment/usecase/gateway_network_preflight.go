package deploymentsvc

import (
	"context"
	"encoding/json"
	"strings"

	deploymentport "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const gatewayComposeNetworkLabel = "com.docker.compose.network"

type inspectedGatewayNetwork struct {
	Name   string            `json:"Name"`
	Labels map[string]string `json:"Labels"`
}

// preflightGatewayNetwork rejects a pre-existing network that Compose cannot
// safely manage. A missing network is valid because the first Gateway deploy
// creates it from the rendered Compose file.
func preflightGatewayNetwork(ctx context.Context, runner deploymentport.CommandQueryRunner, app model.Application) error {
	if app.Kind != status.ApplicationKindGateway {
		return nil
	}
	if runner == nil {
		return apperror.New(apperror.KindInternal, "Gateway network inspection is not configured")
	}
	output, err := runner.Run(ctx, "", "docker", "network", "inspect", defaultGatewayNetworkName)
	if err != nil {
		if dockerNetworkNotFound(output) {
			return nil
		}
		return apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	var networks []inspectedGatewayNetwork
	if err := json.Unmarshal([]byte(output), &networks); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to inspect Gateway network", err)
	}
	if len(networks) != 1 || networks[0].Name != defaultGatewayNetworkName {
		return apperror.New(apperror.KindInternal, "Gateway network inspection returned an unexpected result")
	}
	if networks[0].Labels[gatewayComposeNetworkLabel] != gatewayNetworkKey {
		return apperror.New(apperror.KindConflict, "Gateway network conflicts with the current Compose configuration. Remove the existing traefik network before deploying the Gateway.")
	}
	return nil
}

func dockerNetworkNotFound(output string) bool {
	value := strings.ToLower(output)
	return strings.Contains(value, "no such network") || strings.Contains(value, "network "+defaultGatewayNetworkName+" not found")
}
