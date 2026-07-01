package settingshandler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"backend/internal/apperror"
	settingssvc "backend/internal/service/settings"
	"backend/internal/transport/http/handler/authz"
	transportresponse "backend/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

type Handler struct {
	logger        *slog.Logger
	service       settingssvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service settingssvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) Register(r router) {
	r.Get("/api/settings/config", h.getConfig)
	r.Put("/api/settings/config", h.updateConfig)
	r.Delete("/api/settings/config", h.resetConfig)
}

func (h Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:read"); !ok {
		return
	}
	resp, err := h.service.Config(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, systemConfigResponse(resp))
}

func (h Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req SystemConfigUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(r.Context(), req.Key, req.Value)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, systemConfigResponse(resp))
}

func (h Handler) resetConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req SystemConfigResetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Reset(r.Context(), req.Keys)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, systemConfigResponse(resp))
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("settings request failed", "error", err)
	}
	transportresponse.Error(h.logger, w, apperror.StatusCode(err), err.Error())
}

func systemConfigResponse(config settingssvc.SystemConfig) SystemConfigResp {
	items := make([]ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return SystemConfigResp{Items: items}
}

func configItemResponse(item settingssvc.ConfigItem) ConfigItemResp {
	return ConfigItemResp{Key: item.Key, Value: item.Value, Default: item.Default, IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
