package projectinitializationhandler

import (
	"net/http"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	initdto "github.com/leoninew/pomelo-orbit/internal/application/project_initialization/dto"
	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
	initv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/project_initialization"
)

func (h Handler) GetStatus(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.Status(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) SaveEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationEnvironmentReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.SaveEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")), saveEnvironmentInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) PrepareSSHEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationEnvironmentReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	result, err := h.service.PrepareSSHEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")), saveEnvironmentInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &initv1.ProjectInitializationSSHCommandResp{
		Status:    initializationResponse(result.Status),
		PublicKey: result.PublicKey,
	})
}

func (h Handler) ProbeEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.ProbeEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) CreateGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationGatewayReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.CreateGateway(c.Request.Context(), current.Id, strings.TrimSpace(c.Query("project_id")), initdto.CreateGatewayInput{
		Image: req.Image, RestApiUrl: req.RestApiUrl, RestApiHostUrl: req.RestApiHostUrl, RestReadyTimeoutSeconds: int(req.RestReadyTimeoutSeconds),
		BaseDomain: req.BaseDomain, DefaultEntrypoint: req.DefaultEntrypoint, TLSMode: req.TlsMode,
		AcmeProfile: req.AcmeProfile, AcmeEmail: req.AcmeEmail, DNSApiToken: req.DnsApiToken,
	})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusCreated, initializationResponse(view))
}

func initializationResponse(view initdto.StatusView) *initv1.ProjectInitializationStatusResp {
	resp := &initv1.ProjectInitializationStatusResp{
		Status: view.Status,
		Defaults: &initv1.ProjectInitializationDefaults{
			LocalWorkspaceRoot: view.Defaults.LocalWorkspaceRoot, Image: view.Defaults.Image,
			RestApiUrl: view.Defaults.RestApiUrl, RestApiHostUrl: view.Defaults.RestApiHostUrl, RestReadyTimeoutSeconds: int32(view.Defaults.RestReadyTimeoutSeconds),
			BaseDomain: view.Defaults.BaseDomain, DefaultEntrypoint: view.Defaults.DefaultEntrypoint,
			TlsMode: view.Defaults.TLSMode, AcmeProfile: view.Defaults.AcmeProfile,
			AcmeEmail: view.Defaults.AcmeEmail, DnsApiToken: view.Defaults.DNSApiToken,
			LocalPlatform: view.Defaults.LocalPlatform, LocalHost: view.Defaults.LocalHost,
			LocalUsername: view.Defaults.LocalUsername,
		},
	}
	if view.Environment != nil {
		workspaceRoot := ""
		if view.Environment.Local != nil {
			workspaceRoot = view.Environment.Local.WorkspaceRoot
		} else if view.Environment.SSH != nil {
			workspaceRoot = view.Environment.SSH.WorkspaceRoot
		}
		resp.Environment = &initv1.ProjectInitializationEnvironmentSnapshot{
			Id: view.Environment.Id, TargetType: view.Environment.TargetType, WorkspaceRoot: workspaceRoot,
			TargetRevision:    view.Environment.TargetRevision,
			LastProbeRevision: view.Environment.LastProbeRevision, LastProbeStatus: view.Environment.LastProbeStatus,
			LastProbeAt:         transport.FormatOptionalTime(view.Environment.LastProbeAt),
			LastProbeDiagnostic: view.Environment.LastProbeDiagnostic,
			Local:               localTargetResponse(view.Environment.Local),
			Ssh:                 sshTargetResponse(view.Environment.SSH),
		}
	}
	if view.Gateway != nil {
		status := ""
		if view.Gateway.Service != nil {
			status = view.Gateway.Service.Status
		}
		resp.Gateway = &initv1.ProjectInitializationGatewaySnapshot{
			Id: view.Gateway.Application.Id, Code: view.Gateway.Application.Code, Name: view.Gateway.Application.Name,
			RestApiUrl: view.Gateway.Config.RestApiUrl, RestApiHostUrl: view.Gateway.Config.RestApiHostUrl, BaseDomain: view.Gateway.Config.BaseDomain,
			ServiceStatus: status,
		}
	}
	return resp
}

func saveEnvironmentInput(req *initv1.ProjectInitializationEnvironmentReq) initdto.SaveEnvironmentInput {
	input := initdto.SaveEnvironmentInput{TargetType: req.TargetType}
	if req.Local != nil {
		input.Local = &environmentdto.LocalTargetInput{WorkspaceRoot: req.Local.WorkspaceRoot}
	}
	if req.Ssh != nil {
		input.SSH = &environmentdto.SSHTargetInput{
			Platform: req.Ssh.Platform, Host: req.Ssh.Host, Port: int(req.Ssh.Port),
			Username: req.Ssh.Username, WorkspaceRoot: req.Ssh.WorkspaceRoot,
		}
	}
	return input
}

func localTargetResponse(item *environmentdto.LocalTargetView) *environmentv1.EnvironmentLocalTargetResp {
	if item == nil {
		return nil
	}
	return &environmentv1.EnvironmentLocalTargetResp{
		WorkspaceRoot: item.WorkspaceRoot, Platform: item.Platform, Host: item.Host, Username: item.Username,
	}
}

func sshTargetResponse(item *environmentdto.SSHTargetView) *environmentv1.EnvironmentSSHTargetResp {
	if item == nil {
		return nil
	}
	return &environmentv1.EnvironmentSSHTargetResp{
		Platform: item.Platform, Host: item.Host, Port: int32(item.Port),
		Username: item.Username, WorkspaceRoot: item.WorkspaceRoot, HostKeyFingerprint: item.HostKeyFingerprint,
	}
}
