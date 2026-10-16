package deploymentsvc

import (
	"errors"
	"fmt"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var errRuntimeDirectoryNotReady = errors.New("service has no bound runtime directory")

// Directory revisions record provenance; the selected target resolves the saved path.
func runtimeServiceLocation(target environmentport.Target, service model.Service) (deploymentport.ServiceLocation, error) {
	if service.RuntimeDirectory == "" {
		cause := fmt.Errorf("%w (project_id=%s application_id=%s service_id=%s service_code=%s service_status=%s environment_id=%s environment_target_revision=%d runtime_target_revision=%d)",
			errRuntimeDirectoryNotReady, service.ProjectId, service.ApplicationId, service.Id, service.Code, service.Status,
			target.Environment.Id, target.Environment.TargetRevision, service.RuntimeTargetRevision)
		return deploymentport.ServiceLocation{}, apperror.WrapWithCode(apperror.KindConflict, "service_runtime_directory_missing", "Service runtime directory is not recorded. Confirm the service directory for this environment.", cause)
	}
	return deploymentport.ServiceLocation{Code: service.Code, Directory: service.RuntimeDirectory}, nil
}

func lifecycleServiceLocation(target environmentport.Target, service model.Service) (deploymentport.ServiceLocation, error) {
	if service.RuntimeDirectory != "" {
		return runtimeServiceLocation(target, service)
	}
	if service.DeploymentDirectory != "" {
		return plannedServiceLocation(service), nil
	}
	return runtimeServiceLocation(target, service)
}

func plannedServiceLocation(service model.Service) deploymentport.ServiceLocation {
	return deploymentport.ServiceLocation{Code: service.Code, Directory: service.DeploymentDirectory}
}
