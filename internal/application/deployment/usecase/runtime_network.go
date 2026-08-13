package deploymentsvc

import (
	"context"
	"encoding/json"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

const managedGatewayNetwork = "traefik"

// ExternalNetworkInspect reads the only external network owned by an Orbit
// Gateway. Callers cannot choose arbitrary Docker network names.
func (s Service) ExternalNetworkInspect(ctx context.Context, networkName string) (deploymentdto.RuntimeNetwork, error) {
	if strings.TrimSpace(networkName) != managedGatewayNetwork {
		return deploymentdto.RuntimeNetwork{}, apperror.New(apperror.KindValidation, "network_name must be traefik")
	}
	if s.queryRunner == nil {
		return deploymentdto.RuntimeNetwork{}, apperror.New(apperror.KindInternal, "deployment query runner is not configured")
	}
	output, err := s.queryRunner.Run(ctx, "", "docker", "network", "inspect", managedGatewayNetwork)
	if err != nil {
		return deploymentdto.RuntimeNetwork{}, apperror.New(apperror.KindUnavailable, outputOrError(output, err))
	}
	var networks []struct {
		Id     string `json:"Id"`
		Name   string `json:"Name"`
		Driver string `json:"Driver"`
	}
	if err := json.Unmarshal([]byte(output), &networks); err != nil || len(networks) != 1 {
		return deploymentdto.RuntimeNetwork{}, apperror.New(apperror.KindInternal, "invalid traefik network inspect output")
	}
	network := networks[0]
	if network.Name != managedGatewayNetwork {
		return deploymentdto.RuntimeNetwork{}, apperror.New(apperror.KindInternal, "traefik network inspect returned an unexpected network")
	}
	return deploymentdto.RuntimeNetwork{Id: network.Id, Name: network.Name, Driver: network.Driver}, nil
}
