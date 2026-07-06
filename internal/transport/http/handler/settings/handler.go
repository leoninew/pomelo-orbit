package settingshandler

import (
	apiv1 "backend/internal/gen/orbit/api/v1"
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
	body := systemConfigResponse(resp)
	transportresponse.JSON(h.logger, w, http.StatusOK, &body)
}

func (h Handler) updateConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req apiv1.SystemConfigUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(r.Context(), req.Key, transportresponse.NativeValue(req.Value))
	if err != nil {
		h.writeError(w, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.JSON(h.logger, w, http.StatusOK, &body)
}

func (h Handler) resetConfig(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "setting:write"); !ok {
		return
	}
	var req apiv1.SystemConfigResetReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Reset(r.Context(), req.Keys)
	if err != nil {
		h.writeError(w, err)
		return
	}
	body := systemConfigResponse(resp)
	transportresponse.JSON(h.logger, w, http.StatusOK, &body)
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("settings request failed", "error", err)
	}
	transportresponse.Error(h.logger, w, apperror.StatusCode(err), err.Error())
}

func systemConfigResponse(config settingssvc.SystemConfig) apiv1.SystemConfigResp {
	items := make([]apiv1.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return apiv1.SystemConfigResp{Items: transportresponse.Ptrs(items)}
}

func configItemResponse(item settingssvc.ConfigItem) apiv1.ConfigItemResp {
	return apiv1.ConfigItemResp{Key: item.Key, Value: transportresponse.ProtoValue(item.Value), Default: transportresponse.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
