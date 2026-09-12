package projectinitializationhandler

import (
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	initdto "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestInitializationResponseMapsLocalWorkspaceAndDefaults(t *testing.T) {
	revision := int64(1)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	view := initdto.StatusView{
		Status: initdto.StatusNeedsGateway,
		Defaults: initdto.Defaults{
			LocalWorkspaceRoot: "/srv/orbit", Image: "traefik:3.6", RestApiUrl: "http://localhost:8080",
			RestReadyTimeoutSeconds: 20, BaseDomain: "lvh.me", DefaultEntrypoint: "web", TLSMode: "none",
			LocalPlatform: "windows", LocalHost: "orbit-host", LocalUsername: "orbit",
		},
		Environment: &environmentdto.View{
			Id: "environment-1", TargetType: model.EnvironmentTargetTypeLocal, State: model.EnvironmentStateActive,
			TargetRevision: 1, LastProbeRevision: &revision, LastProbeStatus: &probeStatus,
			Local: &environmentdto.LocalTargetView{WorkspaceRoot: "/srv/orbit"},
		},
	}
	resp := initializationResponse(view)
	if resp.Status != initdto.StatusNeedsGateway || resp.Defaults == nil || resp.Defaults.LocalWorkspaceRoot != "/srv/orbit" || resp.Defaults.Image != "traefik:3.6" {
		t.Fatalf("defaults response = %#v", resp)
	}
	if resp.Defaults.LocalPlatform != "windows" || resp.Defaults.LocalHost != "orbit-host" || resp.Defaults.LocalUsername != "orbit" {
		t.Fatalf("local display defaults = %#v", resp.Defaults)
	}
	if resp.Environment == nil || resp.Environment.WorkspaceRoot != "/srv/orbit" || resp.Environment.TargetType != model.EnvironmentTargetTypeLocal {
		t.Fatalf("environment snapshot = %#v", resp.Environment)
	}
	if resp.Environment.Local == nil || resp.Environment.Local.WorkspaceRoot != "/srv/orbit" || resp.Environment.Ssh != nil {
		t.Fatalf("local snapshot = %#v", resp.Environment.Local)
	}
	if resp.Gateway != nil {
		t.Fatalf("gateway snapshot = %#v", resp.Gateway)
	}
}

func TestInitializationResponseMapsSSHTarget(t *testing.T) {
	view := initdto.StatusView{
		Status: initdto.StatusNeedsProbe,
		Environment: &environmentdto.View{
			Id: "environment-1", TargetType: model.EnvironmentTargetTypeSSH, State: model.EnvironmentStateActive,
			TargetRevision: 1,
			SSH: &environmentdto.SSHTargetView{
				Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
				Username: "deploy", WorkspaceRoot: "/srv/orbit",
			},
		},
	}
	resp := initializationResponse(view)
	if resp.Environment == nil || resp.Environment.TargetType != model.EnvironmentTargetTypeSSH || resp.Environment.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("ssh snapshot = %#v", resp.Environment)
	}
	if resp.Environment.Ssh == nil || resp.Environment.Ssh.Host != "192.0.2.10" || resp.Environment.Local != nil {
		t.Fatalf("ssh target = %#v", resp.Environment.Ssh)
	}
}

func TestInitializationResponseMapsReadyGateway(t *testing.T) {
	projectID := "project-1"
	service := model.Service{Status: "stopped"}
	view := initdto.StatusView{
		Status:   initdto.StatusReady,
		Defaults: initdto.Defaults{Image: "traefik:3.6"},
		Gateway: &gatewaydto.GatewayView{
			Application:    model.Application{Id: "gateway-1", ProjectId: &projectID, Code: "traefik", Name: "Traefik"},
			Config:         model.GatewayConfig{RestApiUrl: "http://localhost:8080", BaseDomain: "lvh.me"},
			DefaultService: &service,
		},
	}
	resp := initializationResponse(view)
	if resp.Status != initdto.StatusReady || resp.Gateway == nil || resp.Gateway.Id != "gateway-1" || resp.Gateway.DefaultServiceStatus != "stopped" {
		t.Fatalf("ready response = %#v", resp)
	}
}
