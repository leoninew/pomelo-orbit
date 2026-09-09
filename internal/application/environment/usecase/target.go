package environmentsvc

import (
	"context"
	"errors"
	"strings"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type TargetResolver struct {
	environments repository.EnvironmentStore
	credentials  deploymentCredentialReader
}

func NewTargetResolver(environments repository.EnvironmentStore, credentials deploymentCredentialReader) TargetResolver {
	return TargetResolver{environments: environments, credentials: credentials}
}

func (r TargetResolver) ResolveProjectTarget(ctx context.Context, projectID string) (environmentport.Target, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "project_id is required for deployment target")
	}
	environment, err := r.environments.EnvironmentByProject(ctx, projectID)
	if errors.Is(err, repository.ErrNotFound) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment is not configured")
	}
	if err != nil {
		return environmentport.Target{}, apperror.Wrap(apperror.KindInternal, "Failed to resolve project environment", err)
	}
	if !environment.IsActive() {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment is disabled")
	}
	if !environment.HasFreshSuccessfulProbe() {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment must pass probe after its latest configuration change")
	}
	if environment.IsLocal() {
		return environmentport.Target{Environment: environment}, nil
	}
	if !environment.IsSSH() {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment target type is invalid")
	}
	if r.credentials == nil {
		return environmentport.Target{}, apperror.New(apperror.KindInternal, "deployment SSH credential reader is not configured")
	}
	credential, privateKey, err := r.credentials.DeploymentSSHCredential(ctx, environment.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(environment, credential) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment deployment SSH credential binding is invalid")
	}
	return environmentport.Target{Environment: environment, PrivateKey: &privateKey}, nil
}
