package settingshandler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	settingsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/settings"

	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	settingssvc "github.com/leoninew/pomelo-orbit/internal/application/settings/usecase"
)

type Handler struct {
	logger        *slog.Logger
	service       settingssvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service settingssvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) GetConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:read"); !ok {
		return
	}
	resp, err := h.service.Config(c.Request.Context())
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.ProtoJSON(c, http.StatusOK, &body)
}

func (h Handler) UpdateConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req settingsv1.SystemConfigUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(c.Request.Context(), req.Key, configUpdateValue(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.ProtoJSON(c, http.StatusOK, &body)
}

func (h Handler) ResetConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req settingsv1.SystemConfigResetReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Reset(c.Request.Context(), req.Keys)
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.ProtoJSON(c, http.StatusOK, &body)
}
