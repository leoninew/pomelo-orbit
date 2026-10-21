package environmentsvc

import (
	"context"
	"errors"
	"strings"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type TargetResolver struct {
	environments           repository.EnvironmentStore
	environmentCredentials repository.EnvironmentCredentialStore
	secretKey              string
}

func NewTargetResolver(environments repository.EnvironmentStore, environmentCredentials repository.EnvironmentCredentialStore, secretKey string) TargetResolver {
	return TargetResolver{environments: environments, environmentCredentials: environmentCredentials, secretKey: secretKey}
}

func (r TargetResolver) ResolveProjectTarget(ctx context.Context, projectId string) (environmentport.Target, error) {
	if projectId == "" {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "project_id is required for deployment target")
	}
	environment, err := r.environments.EnvironmentByProject(ctx, projectId)
	if errors.Is(err, repository.ErrNotFound) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment is not configured")
	}
	if err != nil {
		return environmentport.Target{}, apperror.Wrap(apperror.KindInternal, "Failed to resolve project environment", err)
	}
	if strings.TrimSpace(environment.WorkspaceRoot) == "" {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment workspace_root must be configured before deployment")
	}
	if environment.IsLocal() {
		if !validWorkspaceRoot(model.EnvironmentPlatformLinux, environment.WorkspaceRoot) && !validWorkspaceRoot(model.EnvironmentPlatformWindows, environment.WorkspaceRoot) {
			return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment workspace_root is invalid")
		}
		return environmentport.Target{Environment: environment}, nil
	}
	if !environment.IsSSH() {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment target type is invalid")
	}
	if !validWorkspaceRoot(environment.SSH.Platform, environment.WorkspaceRoot) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment workspace_root is invalid")
	}
	if !environment.HasHostKeyFingerprint() {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment SSH host key fingerprint is not pinned")
	}
	if r.environmentCredentials == nil {
		return environmentport.Target{}, apperror.New(apperror.KindInternal, "environment credential store is not configured")
	}
	credential, err := r.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, environment.Id)
	if err != nil || !matchesEnvironmentCredential(environment, credential) {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment deployment SSH credential binding is invalid")
	}
	privateKey, err := decryptEnvironmentPrivateKey(r.secretKey, credential)
	if err != nil {
		return environmentport.Target{}, apperror.New(apperror.KindValidation, "Project environment deployment SSH credential binding is invalid")
	}
	return environmentport.Target{Environment: environment, PrivateKey: &privateKey}, nil
}
