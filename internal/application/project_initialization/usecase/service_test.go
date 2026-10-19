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

func TestProbeResultsDoNotChangeInitializationProgress(t *testing.T) {
	service, environments := newInitializationService(t)
	view, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit"))
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsGateway || view.Environment == nil || view.Environment.Local.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("after save = %#v", view)
	}

	environments.failNextProbe = true
	view, err = service.ProbeEnvironment(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusNeedsGateway || view.Environment == nil || view.Environment.LastProbeStatus == nil || *view.Environment.LastProbeStatus != model.EnvironmentProbeStatusFailed {
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

func TestCreateGatewayRequiresEnvironmentAndReachesReadyWithoutProbe(t *testing.T) {
	service, _ := newInitializationService(t)
	if _, err := service.CreateGateway(context.Background(), "user-1", "project-1", testGatewayInput()); !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("create before environment error = %v", err)
	}
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit")); err != nil {
		t.Fatal(err)
	}
	view, err := service.CreateGateway(context.Background(), "user-1", "project-1", testGatewayInput())
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != initdto.StatusReady || view.Gateway == nil || view.Gateway.Service == nil {
		t.Fatalf("ready status = %#v", view)
	}
	if view.Gateway.Application.Code != "traefik" || view.Gateway.Application.Name != "Traefik" || view.Gateway.Service.Code != "traefik-default" {
		t.Fatalf("gateway = app %q %q service %q", view.Gateway.Application.Code, view.Gateway.Application.Name, view.Gateway.Service.Code)
	}
	if _, err := service.SaveEnvironment(context.Background(), "user-1", "project-1", localSaveInput("/srv/orbit-next")); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("save after ready error = %v", err)
	}
}

func TestInitializedProjectStaysReadyAfterWorkspaceMigrationOrFailedProbe(t *testing.T) {
	service, environments := newInitializationService(t)
	ctx := context.Background()
	if _, err := service.SaveEnvironment(ctx, "user-1", "project-1", localSaveInput("D:/old")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ProbeEnvironment(ctx, "user-1", "project-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateGateway(ctx, "user-1", "project-1", testGatewayInput()); err != nil {
		t.Fatal(err)
	}
	environments.item.Local.WorkspaceRoot = "D:/Documents/orbit"
	environments.item.TargetRevision++
	failed, succeeded := model.EnvironmentProbeStatusFailed, model.EnvironmentProbeStatusSucceeded
	for _, test := range []struct {
		name   string
		status *string
	}{
		{name: "stale successful probe", status: &succeeded},
		{name: "failed probe", status: &failed},
		{name: "no probe history"},
	} {
		t.Run(test.name, func(t *testing.T) {
			environments.item.LastProbeStatus = test.status
			view, err := service.Status(ctx, "user-1", "project-1")
			if err != nil {
				t.Fatal(err)
			}
			if view.Status != initdto.StatusReady || view.Gateway == nil || view.Gateway.Service == nil || view.Environment.Local.WorkspaceRoot != "D:/Documents/orbit" || view.Environment.LastProbeStatus != test.status {
				t.Fatalf("initialized status = %#v", view)
			}
		})
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
	if view.Status != initdto.StatusNeedsGateway || view.Environment == nil || view.Environment.Local.WorkspaceRoot != "/srv/orbit-next" {
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
	if view.Status != initdto.StatusNeedsGateway || view.Environment == nil || view.Environment.TargetType != model.EnvironmentTargetTypeSSH {
		t.Fatalf("after ssh save = %#v", view)
	}
	if view.Environment.SSH == nil || view.Environment.SSH.Host != "192.0.2.10" || view.Environment.SSH.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("ssh target = %#v", view.Environment.SSH)
	}
}

func TestPrepareSSHEnvironmentPersistsTargetBeforeProbe(t *testing.T) {
	service, environments := newInitializationService(t)
	environments.publicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbitDeploymentKey"
	environments.keyCreatedAt = time.Date(2026, time.October, 15, 1, 2, 3, 0, time.UTC)
	environments.sourceHost = "orbit-control"
	result, err := service.PrepareSSHEnvironment(context.Background(), "user-1", "project-1", initdto.SaveEnvironmentInput{
		TargetType: model.EnvironmentTargetTypeSSH,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22,
			Username: "orbit", WorkspaceRoot: `C:\\orbit`,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PublicKey != environments.publicKey || !result.KeyCreatedAt.Equal(environments.keyCreatedAt) || result.SourceHost != environments.sourceHost || result.Status.Status != initdto.StatusNeedsGateway {
		t.Fatalf("prepare result = %#v", result)
	}
	if environments.item == nil || environments.item.SSH == nil || environments.item.SSH.Host != "192.0.2.10" {
		t.Fatalf("prepared environment = %#v", environments.item)
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
		Image: "traefik:3.6", RestApiUrl: model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl, RestReadyTimeoutSeconds: 20,
		InternalDomain: "lvh.me", DefaultEntrypoint: "web", TLSMode: "none",
	}
}

func newInitializationService(t *testing.T) (Service, *fakeInitEnvironment) {
	t.Helper()
	projectId := "project-1"
	environments := &fakeInitEnvironment{projectId: projectId}
	return New(
		fakeInitProject{project: model.Project{Id: projectId, Code: "demo", Name: "Demo"}},
		environments,
		&fakeInitGateway{projectId: projectId},
		config.ProjectInitializationConfig{
			Environment: config.ProjectInitializationEnvironmentConfig{LocalWorkspaceRoot: "~/.pomelo-orbit"},
			Gateway: config.ProjectInitializationGatewayConfig{
				Image: "traefik:3.6", RestApiUrl: model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl, InternalDomain: "lvh.me",
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
	projectId     string
	item          *environmentdto.View
	failNextProbe bool
	publicKey     string
	keyCreatedAt  time.Time
	sourceHost    string
}

func (f *fakeInitEnvironment) EnvironmentForUser(context.Context, string, string) (environmentdto.View, error) {
	if f.item == nil {
		return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	return *f.item, nil
}

func (f *fakeInitEnvironment) PrepareSSHEnvironment(_ context.Context, _ string, projectId string, input environmentdto.UpdateInput) (environmentdto.SSHCommandView, error) {
	view, err := f.SaveInitialization(context.Background(), "", projectId, input)
	if err != nil {
		return environmentdto.SSHCommandView{}, err
	}
	return environmentdto.SSHCommandView{Environment: view, PublicKey: f.publicKey, KeyCreatedAt: f.keyCreatedAt, SourceHost: f.sourceHost}, nil
}

func (f *fakeInitEnvironment) SaveInitialization(_ context.Context, _ string, projectId string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if f.item != nil && f.item.GatewayApplicationId != nil {
		return environmentdto.View{}, apperror.New(apperror.KindConflict, "Project environment is already bound to a gateway")
	}
	targetType := ""
	if input.TargetType != nil {
		targetType = *input.TargetType
	}
	view := environmentdto.View{
		Id: "environment-1", ProjectId: projectId, Code: "demo",
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
	projectId string
	item      *gatewaydto.GatewayView
}

func (f *fakeInitGateway) ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error) {
	if f.item == nil {
		return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{}, Page: 1, PerPage: 1}, nil
	}
	return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{*f.item}, Total: 1, Page: 1, PerPage: 1}, nil
}

func (f *fakeInitGateway) CreateGateway(_ context.Context, _, projectId string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error) {
	if f.item != nil {
		return gatewaydto.GatewayView{}, errors.New("gateway already created")
	}
	service := model.Service{Id: "service-1", Code: input.Code + "-default", Status: "stopped"}
	view := gatewaydto.GatewayView{
		Application: model.Application{Id: "gateway-1", ProjectId: &projectId, Code: input.Code, Name: input.Name},
		Config:      model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: input.RestApiUrl, RestApiHostUrl: input.RestApiHostUrl, InternalDomain: input.InternalDomain},
		Service:     &service,
	}
	f.item = &view
	return view, nil
}
