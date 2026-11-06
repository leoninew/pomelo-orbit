package servicehandler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	servicev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/service"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type serviceCopier interface {
	Copy(context.Context, string, string, string, servicedto.ServiceCopyInput) (model.Service, error)
}

type CopyHandler struct {
	service       serviceCopier
	authenticator security.Authenticator
}

func NewCopy(service serviceCopier, authenticator security.Authenticator) CopyHandler {
	return CopyHandler{service: service, authenticator: authenticator}
}

func (h CopyHandler) CopyService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceCopyReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	service, err := h.service.Copy(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("service_id"), servicedto.ServiceCopyInput{
		TargetProjectId: req.TargetProjectId, CopyAllVersions: req.CopyAllVersions,
		DeploymentDirectory: req.DeploymentDirectory, EnvironmentTargetRevision: req.EnvironmentTargetRevision,
	})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusCreated, &servicev1.ServiceCopyResp{Id: service.Id})
}
