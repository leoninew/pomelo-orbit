package settingshandler

import (
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	settingsv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/settings"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
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
		h.writeError(c, err)
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
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(c.Request.Context(), req.Key, configUpdateValue(&req))
	if err != nil {
		h.writeError(c, err)
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
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Reset(c.Request.Context(), req.Keys)
	if err != nil {
		h.writeError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.ProtoJSON(c, http.StatusOK, &body)
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("settings request failed", "error", err)
	}
	transportresponse.Error(c, apperror.StatusCode(err), err.Error())
}
