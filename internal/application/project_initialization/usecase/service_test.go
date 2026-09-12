package projectinitializationsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	initdto "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestStatusStartsAtNeedsEnvironment(t *testing.T) {
	service, _ := newInitializationService(t)
	view, err := service.Status(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsEnvironment || view.Environment != nil || view.Gateway != nil {
		t.Fatalf("status = %#v", view)
	}
	if view.Defaults.LocalWorkspaceRoot != "~/.pomelo-orbit" || view.Defaults.Image != "traefik:3.6" {
		t.Fatalf("defaults = %#v", view.Defaults)
	}
	if view.Defaults.LocalPlatform != "windows" || view.Defaults.LocalHost != "orbit-host" || view.Defaults.LocalUsername != "orbit" {
		t.Fatalf("local display defaults = %#v", view.Defaults)
	}
}

func TestSaveEnvironmentAndFailedProbeStayAtNeedsProbe(t *testing.T) {
	service, environments := newInitializationService(t)
	view, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit"))
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsProbe || view.Environment == nil || view.Environment.Local.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("after save = %#v", view)
	}

	environments.failNextProbe = true
	view, err = service.ProbeEnvironment(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsProbe || view.Environment == nil || view.Environment.LastProbeStatus == nil || *view.Environment.LastProbeStatus != model.EnvironmentProbeStatusFailed {
		t.Fatalf("after failed probe = %#v", view)
	}

	view, err = service.ProbeEnvironment(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsGateway || view.Gateway != nil {
		t.Fatalf("after successful probe = %#v", view)
	}
}

func TestCreateGatewayRequiresSuccessfulProbeAndReachesReady(t *testing.T) {
	service, _ := newInitializationService(t)
	if _, err := service.CreateGateway(context.Background(), "user-1", "project-1", testGatewayInput()); !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("create before environment error = %v", err)
	}
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateGateway(context.Background(), "user-1", "project-1", testGatewayInput()); !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("create before probe error = %v", err)
	}
	if _, err := service.ProbeEnvironment(context.Background(), "user-1", "project-1"); err != nil {
		t.Fatal(err)
	}
	view, err := service.CreateGateway(context.Background(), "user-1", "project-1", testGatewayInput())
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusReady || view.Gateway == nil || view.Gateway.DefaultService == nil {
		t.Fatalf("ready status = %#v", view)
	}
	if view.Gateway.Application.Code != "traefik" || view.Gateway.Application.Name != "Traefik" || view.Gateway.DefaultService.Code != "traefik-default" {
		t.Fatalf("gateway = app %q %q service %q", view.Gateway.Application.Code, view.Gateway.Application.Name, view.Gateway.DefaultService.Code)
	}
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit-next")); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("save after ready error = %v", err)
	}
}

func TestSaveEnvironmentAfterSuccessfulProbeAllowsWizardRollback(t *testing.T) {
	service, _ := newInitializationService(t)
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProbeEnvironment(context.Background(), "user-1", "project-1"); err != nil {
		t.Fatal(err)
	}

	view, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit-next"))
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsProbe || view.Environment == nil || view.Environment.Local.WorkspaceRoot != "/srv/orbit-next" {
		t.Fatalf("rollback save = %#v", view)
	}
}

func TestSaveEnvironmentPersistsSSHTarget(t *testing.T) {
	service, _ := newInitializationService(t)
	view, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", initdto.SaveEnvironmentInput{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
			Username: "deploy", WorkspaceRoot: "/srv/orbit",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsProbe || view.Environment == nil || view.Environment.TargetType != model.EnvironmentTargetTypeSSH {
		t.Fatalf("after ssh save = %#v", view)
	}
	if view.Environment.SSH == nil || view.Environment.SSH.Host != "192.0.2.10" || view.Environment.SSH.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("ssh target = %#v", view.Environment.SSH)
	}
}

func TestDeploymentSSHPublicKeyIsAvailableBeforeEnvironmentSave(t *testing.T) {
	service, environments := newInitializationService(t)
	environments.publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbitDeploymentKey"
	publicKey, err := service.DeploymentSSHPublicKey(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if publicKey != environments.publicKey {
		t.Fatalf("public key before save = %q", publicKey)
	}
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", initdto.SaveEnvironmentInput{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22,
			Username: "orbit", WorkspaceRoot: `C:\\orbit`,
		},
	}); err != nil {
		t.Fatal(err)
	}
	publicKey, err = service.DeploymentSSHPublicKey(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if publicKey != environments.publicKey {
		t.Fatalf("public key = %q", publicKey)
	}
}

func localSaveInput(workspaceRoot string) initdto.SaveEnvironmentInput {
	return initdto.SaveEnvironmentInput{
		TargetType: model.EnvironmentTargetTypeLocal,
		Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: workspaceRoot},
	}
}

func testGatewayInput() initdto.CreateGatewayInput {
	return initdto.CreateGatewayInput{
		Image: "traefik:3.6", RestApiUrl: "http://localhost:8080", RestReadyTimeoutSeconds: 20,
		BaseDomain: "lvh.me", DefaultEntrypoint: "web", TLSMode: "none",
	}
}

func newInitializationService(t *testing.T) (Service, *fakeInitEnvironment) {
	t.Helper()
	projectID := "project-1"
	environments := &fakeInitEnvironment{projectID: projectID}
	return New(
		fakeInitProject{project: model.Project{Id: projectID, Code: "demo", Name: "Demo"}},
		environments,
		&fakeInitGateway{projectID: projectID},
		config.ProjectInitializationConfig{
			Environment: config.ProjectInitializationEnvironmentConfig{LocalWorkspaceRoot: "~/.pomelo-orbit"},
			Gateway: config.ProjectInitializationGatewayConfig{
				Image: "traefik:3.6", RestApiUrl: "http://localhost:8080", BaseDomain: "lvh.me",
				RestReadyTimeout: 20 * time.Second, DefaultEntrypoint: "web", TLSMode: "none",
			},
		},
		environmentdto.LocalDisplaySnapshot{Platform: "windows", Host: "orbit-host", Username: "orbit"},
	), environments
}

type fakeInitProject struct {
	project model.Project
}

func (f fakeInitProject) LoadForUser(context.Context, string, string) (model.Project, error) {
	return f.project, nil
}

type fakeInitEnvironment struct {
	projectID     string
	item          *environmentdto.View
	failNextProbe bool
	publicKey     string
}

func (f *fakeInitEnvironment) EnvironmentForUser(context.Context, string, string) (environmentdto.View, error) {
	if f.item == nil {
		return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	return *f.item, nil
}

func (f *fakeInitEnvironment) TestSSHReachability(context.Context, string, string, environmentdto.SSHTargetInput) error {
	return nil
}

func (f *fakeInitEnvironment) DeploymentSSHPublicKeyForProject(context.Context, string, string) (string, error) {
	return f.publicKey, nil
}

func (f *fakeInitEnvironment) SaveInitialization(_ context.Context, _ string, projectID string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if f.item != nil && f.item.GatewayApplicationId != nil {
		return environmentdto.View{}, apperror.New(apperror.KindConflict, "Project environment is already bound to a gateway")
	}
	targetType := ""
	if input.TargetType != nil {
		targetType = *input.TargetType
	}
	view := environmentdto.View{
		Id: "environment-1", ProjectId: projectID, Code: "demo", State: model.EnvironmentStateActive,
		TargetType: targetType, TargetRevision: 1,
	}
	if targetType == model.EnvironmentTargetTypeLocal && input.Local != nil {
		view.Local = &environmentdto.LocalTargetView{WorkspaceRoot: input.Local.WorkspaceRoot}
	}
	if targetType == model.EnvironmentTargetTypeSSH && input.SSH != nil {
		view.SSH = &environmentdto.SSHTargetView{
			Platform: input.SSH.Platform, Host: input.SSH.Host, Port: input.SSH.Port,
			Username: input.SSH.Username, WorkspaceRoot: input.SSH.WorkspaceRoot,
		}
	}
	f.item = &view
	return view, nil
}

func (f *fakeInitEnvironment) InitializeForUser(context.Context, string, string, environmentdto.InitializeInput) (environmentdto.View, error) {
	if f.item == nil {
		return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	revision := f.item.TargetRevision
	status := model.EnvironmentProbeStatusSucceeded
	f.item.LastProbeRevision = &revision
	f.item.LastProbeStatus = &status
	return *f.item, nil
}

func (f *fakeInitEnvironment) ProbeForUser(context.Context, string, string) (environmentdto.View, error) {
	if f.item == nil {
		return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	revision := f.item.TargetRevision
	f.item.LastProbeRevision = &revision
	if f.failNextProbe {
		f.failNextProbe = false
		status := model.EnvironmentProbeStatusFailed
		diagnostic := "Local Docker and Docker Compose prerequisites failed."
		f.item.LastProbeStatus = &status
		f.item.LastProbeDiagnostic = &diagnostic
		return *f.item, nil
	}
	status := model.EnvironmentProbeStatusSucceeded
	f.item.LastProbeStatus = &status
	return *f.item, nil
}

type fakeInitGateway struct {
	projectID string
	item      *gatewaydto.GatewayView
}

func (f *fakeInitGateway) ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error) {
	if f.item == nil {
		return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Page: 1, PerPage: 1}, nil
	}
	return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{*f.item}, Total: 1, Page: 1, PerPage: 1}, nil
}

func (f *fakeInitGateway) CreateGateway(_ context.Context, _ string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error) {
	if f.item != nil {
		return gatewaydto.GatewayView{}, errors.New("gateway already created")
	}
	projectID := input.ProjectId
	service := model.Service{Id: "service-1", InstanceKey: "default", Code: input.Code + "-default", Status: "stopped"}
	view := gatewaydto.GatewayView{
		Application:    model.Application{Id: "gateway-1", ProjectId: &projectID, Code: input.Code, Name: input.Name},
		Config:         model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: input.RestApiUrl, BaseDomain: input.BaseDomain},
		DefaultService: &service,
	}
	f.item = &view
	return view, nil
}
