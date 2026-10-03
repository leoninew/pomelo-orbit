package deploymentsvc

import (
	"errors"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var errRuntimeLocationChanged = errors.New("service runtime directory changed")

func runtimeServiceLocation(target environmentport.Target, service model.Service) (deploymentport.ServiceLocation, error) {
	if service.RuntimeDirectory == "" {
		return deploymentport.ServiceLocation{}, deploymentport.ErrLogNotReady
	}
	if service.RuntimeTargetRevision != target.Environment.TargetRevision {
		return deploymentport.ServiceLocation{}, apperror.New(apperror.KindConflict, "Service runtime directory belongs to a previous Environment revision. Deploy again.")
	}
	return deploymentport.ServiceLocation{Code: service.Code, Directory: service.RuntimeDirectory}, nil
}

func plannedServiceLocation(service model.Service) deploymentport.ServiceLocation {
	return deploymentport.ServiceLocation{Code: service.Code, Directory: service.DeploymentDirectory}
}
