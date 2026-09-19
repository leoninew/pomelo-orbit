package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// TargetDefinitionForUser returns the stored target configuration without
// probing or resolving a deployment target.
func (s Service) TargetDefinitionForUser(ctx context.Context, userId string, projectId string) (environmentdto.TargetDefinition, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	environment, err := s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	return s.configurationFromEnvironment(ctx, environment)
}

// SaveTargetDefinitionForUser persists an already-known deployment target. It
// performs model and target-exclusivity validation but deliberately does not
// invoke the SSH reachability probe or any runtime integration.
func (s Service) SaveTargetDefinitionForUser(ctx context.Context, userId string, projectId string, input environmentdto.TargetDefinition) (environmentdto.TargetDefinition, error) {
	return s.saveTargetDefinitionForUser(ctx, userId, projectId, input, false, false)
}

// SaveTargetDefinitionForHandover restores the package value exactly. Import
// persists configuration only; deployment-target availability is checked when
// the target is subsequently used, not while a package is being restored.
func (s Service) SaveTargetDefinitionForHandover(ctx context.Context, userId string, projectId string, input environmentdto.TargetDefinition) (environmentdto.TargetDefinition, error) {
	return s.saveTargetDefinitionForUser(ctx, userId, projectId, input, true, true)
}

func (s Service) saveTargetDefinitionForUser(ctx context.Context, userId string, projectId string, input environmentdto.TargetDefinition, allowLocalWorkspacePlatformMismatch bool, skipTargetAvailabilityCheck bool) (environmentdto.TargetDefinition, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	project, err := s.projects.Project(ctx, strings.TrimSpace(projectId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return environmentdto.TargetDefinition{}, apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	existing, err := s.environments.EnvironmentByProject(ctx, project.Id)
	creating := errors.Is(err, repository.ErrNotFound)
	if err != nil && !creating {
		return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	previousCredentialId := ""
	if !creating && existing.IsSSH() {
		previousCredentialId = existing.SSH.CredentialId
	}
	credential, credentialCreating, err := s.configurationCredential(ctx, project.Id, existing, creating, input)
	if err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	environment, err := environmentFromConfiguration(project, existing, creating, input, credential)
	if err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	if err := validateEnvironment(environment, s.localDisplay.Platform, false, allowLocalWorkspacePlatformMismatch); err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	if !skipTargetAvailabilityCheck {
		if err := s.ensureTargetAvailable(ctx, environment); err != nil {
			return environmentdto.TargetDefinition{}, err
		}
	}
	if credential != nil {
		if credentialCreating {
			if err := s.environmentCredentials.CreateEnvironmentCredential(ctx, *credential); err != nil {
				return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to create environment credential", err)
			}
		} else if err := s.environmentCredentials.UpdateEnvironmentCredential(ctx, *credential); err != nil {
			return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to update environment credential", err)
		}
	}
	if creating {
		if err := s.environments.CreateEnvironment(ctx, environment); err != nil {
			return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to create project environment", err)
		}
	} else if err := s.environments.UpdateEnvironment(ctx, environment); err != nil {
		return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	if previousCredentialId != "" && (credential == nil || credential.Id != previousCredentialId) {
		if err := s.environmentCredentials.DeleteEnvironmentCredential(ctx, previousCredentialId); err != nil {
			return environmentdto.TargetDefinition{}, apperror.Wrap(apperror.KindInternal, "Failed to remove replaced environment SSH credential", err)
		}
	}
	return s.TargetDefinitionForUser(ctx, userId, project.Id)
}

func (s Service) configurationFromEnvironment(ctx context.Context, environment model.Environment) (environmentdto.TargetDefinition, error) {
	configuration := environmentdto.TargetDefinition{
		TargetType:          environment.TargetType,
		WorkspaceRoot:       environment.WorkspaceRoot,
		TargetRevision:      environment.TargetRevision,
		LastProbeRevision:   environment.LastProbeRevision,
		LastProbeStatus:     environment.LastProbeStatus,
		LastProbeAt:         environment.LastProbeAt,
		LastProbeDiagnostic: environment.LastProbeDiagnostic,
	}
	if !environment.IsSSH() {
		return configuration, nil
	}
	configuration.SSH = &environmentdto.SSHDefinition{
		Platform: environment.SSH.Platform, Host: environment.SSH.Host, Port: environment.SSH.Port,
		Username: environment.SSH.Username, CredentialRevision: environment.SSH.CredentialRevision,
		HostKeyFingerprint: environment.SSH.HostKeyFingerprint,
	}
	credential, err := s.environmentCredential(ctx, environment.SSH.CredentialId)
	if err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	if !matchesEnvironmentCredential(environment, credential) {
		return environmentdto.TargetDefinition{}, apperror.New(apperror.KindValidation, "Project environment deployment SSH credential binding is invalid")
	}
	privateKey, err := decryptEnvironmentPrivateKey(s.secretKey, credential)
	if err != nil {
		return environmentdto.TargetDefinition{}, err
	}
	configuration.Credential = &environmentdto.SSHCredentialDefinition{
		PublicKey: credential.PublicKey, PrivateKey: privateKey.PrivateKey, Revision: credential.Revision,
	}
	return configuration, nil
}

func (s Service) configurationCredential(ctx context.Context, projectId string, existing model.Environment, creating bool, input environmentdto.TargetDefinition) (*model.EnvironmentCredential, bool, error) {
	if input.TargetType != model.EnvironmentTargetTypeSSH {
		if input.Credential != nil {
			return nil, false, apperror.New(apperror.KindValidation, "Local environment must not include an SSH credential")
		}
		return nil, false, nil
	}
	if input.SSH == nil || input.Credential == nil {
		return nil, false, apperror.New(apperror.KindValidation, "SSH environment credential is required")
	}
	if s.environmentCredentials == nil {
		return nil, false, apperror.New(apperror.KindInternal, "environment credential store is not configured")
	}
	if input.Credential.Revision < 1 || strings.TrimSpace(input.Credential.PublicKey) == "" || strings.TrimSpace(input.Credential.PrivateKey) == "" {
		return nil, false, apperror.New(apperror.KindValidation, "Environment SSH credential is invalid")
	}
	if input.SSH.CredentialRevision != input.Credential.Revision {
		return nil, false, apperror.New(apperror.KindValidation, "Environment SSH credential revision does not match its target binding")
	}
	if strings.TrimSpace(s.secretKey) == "" {
		return nil, false, apperror.New(apperror.KindInternal, "credential encryption key is not configured")
	}
	item := model.EnvironmentCredential{Id: idutil.NewId(), ProjectId: projectId, CreatedAt: time.Now().UTC()}
	credentialCreating := true
	if !creating && existing.IsSSH() && strings.TrimSpace(existing.SSH.CredentialId) != "" {
		loaded, err := s.environmentCredential(ctx, existing.SSH.CredentialId)
		if err != nil {
			return nil, false, err
		}
		if loaded.ProjectId != projectId {
			return nil, false, apperror.New(apperror.KindValidation, "Environment SSH credential does not belong to the project")
		}
		item = loaded
		credentialCreating = false
	}
	encrypted, err := security.EncryptString(s.secretKey, input.Credential.PrivateKey)
	if err != nil {
		return nil, false, apperror.Wrap(apperror.KindInternal, "Failed to encrypt deployment SSH private key", err)
	}
	item.PublicKey = input.Credential.PublicKey
	item.EncryptedPrivateKey = encrypted
	item.Revision = input.Credential.Revision
	return &item, credentialCreating, nil
}

func environmentFromConfiguration(project model.Project, existing model.Environment, creating bool, input environmentdto.TargetDefinition, credential *model.EnvironmentCredential) (model.Environment, error) {
	if input.TargetRevision < 1 {
		return model.Environment{}, apperror.New(apperror.KindValidation, "Environment target_revision must be positive")
	}
	if err := validateProbeState(input); err != nil {
		return model.Environment{}, err
	}
	environment := model.Environment{
		ProjectId: project.Id, Code: project.Code, TargetType: strings.TrimSpace(input.TargetType), WorkspaceRoot: strings.TrimSpace(input.WorkspaceRoot),
		TargetRevision: input.TargetRevision, LastProbeRevision: input.LastProbeRevision, LastProbeStatus: input.LastProbeStatus,
		LastProbeAt: input.LastProbeAt, LastProbeDiagnostic: input.LastProbeDiagnostic,
	}
	if creating {
		environment.Id = idutil.NewId()
		environment.CreatedAt = time.Now().UTC()
	} else {
		environment.Id = existing.Id
		environment.CreatedAt = existing.CreatedAt
		environment.GatewayApplicationId = existing.GatewayApplicationId
	}
	if input.SSH != nil {
		if credential == nil {
			return model.Environment{}, apperror.New(apperror.KindValidation, "SSH environment credential is required")
		}
		environment.SSH = &model.EnvironmentSSHTarget{
			Platform: strings.TrimSpace(input.SSH.Platform), Host: strings.TrimSpace(input.SSH.Host), Port: input.SSH.Port,
			Username: strings.TrimSpace(input.SSH.Username), CredentialId: credential.Id,
			CredentialRevision: input.SSH.CredentialRevision, HostKeyFingerprint: strings.TrimSpace(input.SSH.HostKeyFingerprint),
		}
	}
	return environment, nil
}

func validateProbeState(input environmentdto.TargetDefinition) error {
	if input.LastProbeRevision == nil {
		if input.LastProbeStatus != nil || input.LastProbeAt != nil || input.LastProbeDiagnostic != nil {
			return apperror.New(apperror.KindValidation, "Environment probe state is invalid")
		}
		return nil
	}
	if *input.LastProbeRevision < 1 || *input.LastProbeRevision > input.TargetRevision || input.LastProbeStatus == nil || input.LastProbeAt == nil {
		return apperror.New(apperror.KindValidation, "Environment probe state is invalid")
	}
	if *input.LastProbeStatus != model.EnvironmentProbeStatusSucceeded && *input.LastProbeStatus != model.EnvironmentProbeStatusFailed {
		return apperror.New(apperror.KindValidation, "Environment probe status is invalid")
	}
	return nil
}

func (s Service) ensureTargetAvailable(ctx context.Context, environment model.Environment) error {
	reader, ok := s.environments.(environmentTargetReader)
	if !ok {
		return nil
	}
	host, port := "", 0
	if environment.SSH != nil {
		host, port = environment.SSH.Host, environment.SSH.Port
	}
	if _, err := reader.EnvironmentByTarget(ctx, environment.ProjectId, environment.TargetType, host, port); err == nil {
		return apperror.New(apperror.KindConflict, "The Docker target is already bound to another project")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check environment target", err)
	}
	return nil
}
