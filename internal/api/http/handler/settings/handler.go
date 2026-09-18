package settingshandler

import (
	"log/slog"
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	settingsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/settings"

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
		transport.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transport.WriteProtoJSON(c, http.StatusOK, &body)
}

func (h Handler) UpdateConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req settingsv1.SystemConfigUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(c.Request.Context(), req.Key, configUpdateValue(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transport.WriteProtoJSON(c, http.StatusOK, &body)
}

func (h Handler) ResetConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req settingsv1.SystemConfigResetReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Reset(c.Request.Context(), req.Keys)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transport.WriteProtoJSON(c, http.StatusOK, &body)
}
