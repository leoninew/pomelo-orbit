package environmenthandler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/environment"
	"github.com/leoninew/pomelo-orbit/internal/model"
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
	transportresponse.ProtoJSON(c, http.StatusOK, &response)
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
	transportresponse.ProtoJSON(c, http.StatusOK, &response)
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
	transportresponse.ProtoJSON(c, http.StatusOK, &response)
}
func projectEnvironmentUpdateInput(req *environmentv1.ProjectEnvironmentUpdateReq) environmentdto.UpdateInput {
	input := environmentdto.UpdateInput{
		State:                      req.State,
		Platform:                   req.Platform,
		Host:                       req.Host,
		Username:                   req.Username,
		WorkspaceRoot:              req.WorkspaceRoot,
		DeploymentSSHPrivateKey:    req.DeploymentSshPrivateKey,
		DeploymentSSHKeyPassphrase: req.DeploymentSshKeyPassphrase,
		HostKeyFingerprint:         req.HostKeyFingerprint,
	}
	if req.Port != nil {
		value := int(*req.Port)
		input.Port = &value
	}
	return input
}

func environmentResponse(item model.Environment) environmentv1.EnvironmentResp {
	return environmentv1.EnvironmentResp{
		Id:                    item.Id,
		ProjectId:             item.ProjectId,
		Code:                  item.Code,
		State:                 item.State,
		Platform:              item.Platform,
		Host:                  item.Host,
		Port:                  int32(item.Port),
		Username:              item.Username,
		WorkspaceRoot:         item.WorkspaceRoot,
		SshCredentialId:       item.SSHCredentialId,
		SshCredentialRevision: item.SSHCredentialRevision,
		HostKeyFingerprint:    item.HostKeyFingerprint,
		TargetRevision:        item.TargetRevision,
		LastProbeRevision:     item.LastProbeRevision,
		LastProbeStatus:       item.LastProbeStatus,
		LastProbeAt:           transportresponse.FormatOptionalTime(item.LastProbeAt),
		LastProbeDiagnostic:   item.LastProbeDiagnostic,
		GatewayApplicationId:  item.GatewayApplicationId,
		CreatedAt:             transportresponse.FormatTime(item.CreatedAt),
		UpdatedAt:             transportresponse.FormatTime(item.UpdatedAt),
	}
}
