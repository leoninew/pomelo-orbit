package projectinitializationhandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
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
	view, err := h.service.Status(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) TestEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationEnvironmentReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.TestEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), saveEnvironmentInput(&req)); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &initv1.ProjectInitializationEnvironmentTestResp{Ok: true})
}

func (h Handler) SaveEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationEnvironmentReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.SaveEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), saveEnvironmentInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) GetDeploymentPublicKey(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	publicKey, err := h.service.DeploymentSSHPublicKey(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &initv1.ProjectInitializationDeploymentPublicKeyResp{PublicKey: publicKey})
}

func (h Handler) BootstrapEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationBootstrapReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.BootstrapEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), initdto.BootstrapEnvironmentInput{
		Username: req.Username, Password: req.Password, PrivateKey: req.PrivateKey, PrivateKeyPassphrase: req.PrivateKeyPassphrase,
	})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) ProbeEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.ProbeEnvironment(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, initializationResponse(view))
}

func (h Handler) CreateGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req initv1.ProjectInitializationGatewayReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.CreateGateway(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), initdto.CreateGatewayInput{
		Image: req.Image, RestApiUrl: req.RestApiUrl, RestReadyTimeoutSeconds: int(req.RestReadyTimeoutSeconds),
		BaseDomain: req.BaseDomain, DefaultEntrypoint: req.DefaultEntrypoint, TLSMode: req.TlsMode,
		AcmeProfile: req.AcmeProfile, AcmeEmail: req.AcmeEmail, DNSApiToken: req.DnsApiToken,
	})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusCreated, initializationResponse(view))
}

func initializationResponse(view initdto.StatusView) *initv1.ProjectInitializationStatusResp {
	resp := &initv1.ProjectInitializationStatusResp{
		Status: view.Status,
		Defaults: &initv1.ProjectInitializationDefaults{
			LocalWorkspaceRoot: view.Defaults.LocalWorkspaceRoot, Image: view.Defaults.Image,
			RestApiUrl: view.Defaults.RestApiUrl, RestReadyTimeoutSeconds: int32(view.Defaults.RestReadyTimeoutSeconds),
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
			State: view.Environment.State, TargetRevision: view.Environment.TargetRevision,
			LastProbeRevision: view.Environment.LastProbeRevision, LastProbeStatus: view.Environment.LastProbeStatus,
			LastProbeAt:         transportresponse.FormatOptionalTime(view.Environment.LastProbeAt),
			LastProbeDiagnostic: view.Environment.LastProbeDiagnostic,
			Local:               localTargetResponse(view.Environment.Local),
			Ssh:                 sshTargetResponse(view.Environment.SSH),
		}
	}
	if view.Gateway != nil {
		status := ""
		if view.Gateway.DefaultService != nil {
			status = view.Gateway.DefaultService.Status
		}
		resp.Gateway = &initv1.ProjectInitializationGatewaySnapshot{
			Id: view.Gateway.Application.Id, Code: view.Gateway.Application.Code, Name: view.Gateway.Application.Name,
			RestApiUrl: view.Gateway.Config.RestApiUrl, BaseDomain: view.Gateway.Config.BaseDomain,
			DefaultServiceStatus: status,
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
