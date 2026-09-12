package projectinitializationsvc

import (
	"context"
	"strings"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
	initdto "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/dto"
	initport "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type Service struct {
	projects     initport.ProjectService
	environments initport.EnvironmentService
	gateways     initport.GatewayService
	defaults     config.ProjectInitializationConfig
	localDisplay environmentdto.LocalDisplaySnapshot
}

func New(projects initport.ProjectService, environments initport.EnvironmentService, gateways initport.GatewayService, defaults config.ProjectInitializationConfig, localDisplay environmentdto.LocalDisplaySnapshot) Service {
	return Service{projects: projects, environments: environments, gateways: gateways, defaults: defaults, localDisplay: localDisplay}
}

func (s Service) Status(ctx context.Context, userID string, projectID string) (initdto.StatusView, error) {
	if _, err := s.projects.LoadForUser(ctx, projectID, userID); err != nil {
		return initdto.StatusView{}, err
	}
	view := initdto.StatusView{Defaults: s.defaultValues()}
	environment, err := s.environments.EnvironmentForUser(ctx, userID, projectID)
	if err != nil {
		if apperror.IsKind(err, apperror.KindNotFound) {
			view.Status = initdto.StatusNeedsEnvironment
			return view, nil
		}
		return initdto.StatusView{}, err
	}
	view.Environment = &environment
	if environment.State != model.EnvironmentStateActive || !hasFreshSuccessfulProbe(environment) {
		view.Status = initdto.StatusNeedsProbe
		return view, nil
	}
	gateways, err := s.gateways.ListGateways(ctx, userID, projectID, 1, 1, "")
	if err != nil {
		return initdto.StatusView{}, err
	}
	if gateways.Total == 0 || len(gateways.Items) == 0 || gateways.Items[0].DefaultService == nil {
		view.Status = initdto.StatusNeedsGateway
		return view, nil
	}
	gateway := gateways.Items[0]
	view.Gateway = &gateway
	view.Status = initdto.StatusReady
	return view, nil
}

func (s Service) TestEnvironment(ctx context.Context, userID string, projectID string, input initdto.SaveEnvironmentInput) error {
	if _, err := s.projects.LoadForUser(ctx, projectID, userID); err != nil {
		return err
	}
	if strings.TrimSpace(input.TargetType) != model.EnvironmentTargetTypeSSH || input.SSH == nil {
		return apperror.New(apperror.KindValidation, "SSH reachability test is only available for remote SSH targets")
	}
	return s.environments.TestSSHReachability(ctx, userID, projectID, *input.SSH)
}

func (s Service) SaveEnvironment(ctx context.Context, userID string, projectID string, input initdto.SaveEnvironmentInput) (initdto.StatusView, error) {
	status, err := s.Status(ctx, userID, projectID)
	if err != nil {
		return initdto.StatusView{}, err
	}
	if status.Status == initdto.StatusReady {
		return initdto.StatusView{}, apperror.New(apperror.KindConflict, "Project environment is already ready")
	}
	targetType := strings.TrimSpace(input.TargetType)
	if _, err := s.environments.SaveInitialization(ctx, userID, projectID, environmentdto.UpdateInput{
		TargetType: &targetType, Local: input.Local, SSH: input.SSH,
	}); err != nil {
		return initdto.StatusView{}, err
	}
	return s.Status(ctx, userID, projectID)
}

func (s Service) BootstrapEnvironment(ctx context.Context, userID string, projectID string, input initdto.BootstrapEnvironmentInput) (initdto.StatusView, error) {
	status, err := s.Status(ctx, userID, projectID)
	if err != nil {
		return initdto.StatusView{}, err
	}
	if status.Status == initdto.StatusReady || status.Status == initdto.StatusNeedsGateway {
		return initdto.StatusView{}, apperror.New(apperror.KindConflict, "Project environment is already ready")
	}
	if status.Status == initdto.StatusNeedsEnvironment {
		return initdto.StatusView{}, apperror.New(apperror.KindValidation, "Project environment must be saved before it can be initialized")
	}
	if _, err := s.environments.InitializeForUser(ctx, userID, projectID, environmentdto.InitializeInput{
		Username: input.Username, Password: input.Password,
		PrivateKey: input.PrivateKey, PrivateKeyPassphrase: input.PrivateKeyPassphrase,
	}); err != nil {
		return initdto.StatusView{}, err
	}
	return s.Status(ctx, userID, projectID)
}

func (s Service) DeploymentSSHPublicKey(ctx context.Context, userID string, projectID string) (string, error) {
	if _, err := s.projects.LoadForUser(ctx, projectID, userID); err != nil {
		return "", err
	}
	return s.environments.DeploymentSSHPublicKeyForProject(ctx, userID, projectID)
}

func (s Service) ProbeEnvironment(ctx context.Context, userID string, projectID string) (initdto.StatusView, error) {
	if _, err := s.projects.LoadForUser(ctx, projectID, userID); err != nil {
		return initdto.StatusView{}, err
	}
	if _, err := s.environments.ProbeForUser(ctx, userID, projectID); err != nil {
		return initdto.StatusView{}, err
	}
	return s.Status(ctx, userID, projectID)
}

func (s Service) CreateGateway(ctx context.Context, userID string, projectID string, input initdto.CreateGatewayInput) (initdto.StatusView, error) {
	status, err := s.Status(ctx, userID, projectID)
	if err != nil {
		return initdto.StatusView{}, err
	}
	if status.Status != initdto.StatusNeedsGateway {
		return initdto.StatusView{}, apperror.New(apperror.KindValidation, "Project environment must pass probe before creating a gateway")
	}
	project, err := s.projects.LoadForUser(ctx, projectID, userID)
	if err != nil {
		return initdto.StatusView{}, err
	}
	timeout := input.RestReadyTimeoutSeconds
	image := strings.TrimSpace(input.Image)
	entrypoint := strings.TrimSpace(input.DefaultEntrypoint)
	tlsMode := strings.TrimSpace(input.TLSMode)
	acmeProfile := strings.TrimSpace(input.AcmeProfile)
	acmeEmail := strings.TrimSpace(input.AcmeEmail)
	dnsToken := strings.TrimSpace(input.DNSApiToken)
	if _, err := s.gateways.CreateGateway(ctx, userID, gatewaydto.GatewayCreateInput{
		ProjectId:               project.Id,
		Code:                    gatewaysvc.ManagedGatewayCode(),
		Name:                    gatewaysvc.ManagedGatewayName(),
		RestApiUrl:              input.RestApiUrl,
		RestReadyTimeoutSeconds: &timeout,
		BaseDomain:              input.BaseDomain,
		InitialComponentImage:   &image,
		DefaultEntrypoint:       &entrypoint,
		TLSMode:                 &tlsMode,
		AcmeProfile:             &acmeProfile,
		AcmeEmail:               &acmeEmail,
		DNSApiToken:             &dnsToken,
	}); err != nil {
		return initdto.StatusView{}, err
	}
	return s.Status(ctx, userID, projectID)
}

func (s Service) defaultValues() initdto.Defaults {
	return initdto.Defaults{
		LocalWorkspaceRoot:      s.defaults.Environment.LocalWorkspaceRoot,
		Image:                   s.defaults.Gateway.Image,
		RestApiUrl:              s.defaults.Gateway.RestApiUrl,
		RestReadyTimeoutSeconds: s.defaults.Gateway.RestReadyTimeoutSeconds(),
		BaseDomain:              s.defaults.Gateway.BaseDomain,
		DefaultEntrypoint:       s.defaults.Gateway.DefaultEntrypoint,
		TLSMode:                 s.defaults.Gateway.TLSMode,
		AcmeProfile:             s.defaults.Gateway.AcmeProfile,
		AcmeEmail:               s.defaults.Gateway.AcmeEmail,
		DNSApiToken:             s.defaults.Gateway.DNSApiToken,
		LocalPlatform:           s.localDisplay.Platform,
		LocalHost:               s.localDisplay.Host,
		LocalUsername:           s.localDisplay.Username,
	}
}

func hasFreshSuccessfulProbe(view environmentdto.View) bool {
	if view.LastProbeRevision == nil || *view.LastProbeRevision != view.TargetRevision || view.LastProbeStatus == nil || *view.LastProbeStatus != model.EnvironmentProbeStatusSucceeded {
		return false
	}
	if view.TargetType == model.EnvironmentTargetTypeLocal {
		return true
	}
	if view.SSH == nil {
		return false
	}
	fingerprint := strings.TrimSpace(view.SSH.HostKeyFingerprint)
	return strings.HasPrefix(fingerprint, "SHA256:") && len(fingerprint) > len("SHA256:")
}
