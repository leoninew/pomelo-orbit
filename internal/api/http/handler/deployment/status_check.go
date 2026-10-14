package deploymenthandler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	servicev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/service"
)

func (h Handler) PreviewServiceStatusCheck(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	preview, err := h.service.PreviewServiceStatusCheck(c.Request.Context(), current.Id, c.Query("project_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	items := make([]*servicev1.ServiceStatusCheckItemResp, 0, len(preview.Items))
	for _, item := range preview.Items {
		items = append(items, &servicev1.ServiceStatusCheckItemResp{
			ServiceId: item.ServiceId, ServiceCode: item.ServiceCode, RecordedStatus: item.RecordedStatus, ObservedStatus: item.ObservedStatus,
		})
	}
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServiceStatusCheckPreviewResp{Token: preview.Token, ObservedAt: preview.ObservedAt.Format(time.RFC3339Nano), Items: items})
}

func (h Handler) ConfirmServiceStatusCheck(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceStatusCheckConfirmReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	state, err := h.service.ConfirmServiceStatusCheck(c.Request.Context(), current.Id, c.Query("project_id"), req.Token, req.ServiceId)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServiceStatusCheckConfirmResp{ServiceId: req.ServiceId, Status: state})
}
