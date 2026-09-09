package environmenthandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
)

func (h Handler) GetProjectEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	item, err := h.service.EnvironmentForUser(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := environmentResponse(item)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}

func (h Handler) UpdateProjectEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req environmentv1.ProjectEnvironmentUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	item, err := h.service.UpdateForUser(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), projectEnvironmentUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := environmentResponse(item)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}

func (h Handler) ProbeProjectEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	item, err := h.service.ProbeForUser(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := environmentResponse(item)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}

func (h Handler) InitializeProjectEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req environmentv1.ProjectEnvironmentInitializeReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	item, err := h.service.InitializeForUser(c.Request.Context(), current.Id, strings.TrimSpace(c.Param("project_id")), projectEnvironmentInitializeInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := environmentResponse(item)
	transportresponse.ProtoJSON(c, http.StatusOK, response)
}

func projectEnvironmentUpdateInput(req *environmentv1.ProjectEnvironmentUpdateReq) environmentdto.UpdateInput {
	input := environmentdto.UpdateInput{
		State:      req.State,
		TargetType: req.TargetType,
	}
	if req.Ssh != nil {
		input.SSH = &environmentdto.SSHTargetInput{
			Platform: req.Ssh.Platform, Host: req.Ssh.Host, Port: int(req.Ssh.Port),
			Username: req.Ssh.Username, WorkspaceRoot: req.Ssh.WorkspaceRoot,
		}
	}
	return input
}

func projectEnvironmentInitializeInput(req *environmentv1.ProjectEnvironmentInitializeReq) environmentdto.InitializeInput {
	return environmentdto.InitializeInput{
		Username:             req.Username,
		Password:             req.Password,
		PrivateKey:           req.PrivateKey,
		PrivateKeyPassphrase: req.PrivateKeyPassphrase,
	}
}

func environmentResponse(item environmentdto.View) *environmentv1.EnvironmentResp {
	response := &environmentv1.EnvironmentResp{
		Id:                   item.Id,
		ProjectId:            item.ProjectId,
		Code:                 item.Code,
		State:                item.State,
		TargetType:           item.TargetType,
		TargetRevision:       item.TargetRevision,
		LastProbeRevision:    item.LastProbeRevision,
		LastProbeStatus:      item.LastProbeStatus,
		LastProbeAt:          transportresponse.FormatOptionalTime(item.LastProbeAt),
		LastProbeDiagnostic:  item.LastProbeDiagnostic,
		GatewayApplicationId: item.GatewayApplicationId,
		CreatedAt:            transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:            transportresponse.FormatTime(item.UpdatedAt),
	}
	if item.SSH != nil {
		response.Ssh = &environmentv1.EnvironmentSSHTargetResp{
			Platform: item.SSH.Platform, Host: item.SSH.Host, Port: int32(item.SSH.Port),
			Username: item.SSH.Username, WorkspaceRoot: item.SSH.WorkspaceRoot,
			HostKeyFingerprint: item.SSH.HostKeyFingerprint,
		}
	}
	if item.Local != nil {
		response.Local = &environmentv1.EnvironmentLocalTargetResp{
			WorkspaceRoot: item.Local.WorkspaceRoot,
			Platform:      item.Local.Platform,
			Host:          item.Local.Host,
			Username:      item.Local.Username,
		}
	}
	return response
}
