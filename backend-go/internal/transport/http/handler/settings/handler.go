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

type ConfigItemResp = settingssvc.ConfigItem
type SystemConfigResp = settingssvc.SystemConfig

type systemConfigUpdateReq struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type systemConfigResetReq struct {
	Keys []string `json:"keys"`
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
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req systemConfigUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	resp, err := h.service.Update(r.Context(), req.Key, req.Value)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) resetConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req systemConfigResetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	resp, err := h.service.Reset(r.Context(), req.Keys)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("settings request failed", "error", err)
	}
	transportresponse.JSON(w, apperror.StatusCode(err), map[string]string{"detail": err.Error()})
}
