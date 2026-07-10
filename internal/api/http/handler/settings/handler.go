package settingshandler

import (
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	settingsdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/dto"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

type Handler struct {
	logger        *slog.Logger
	service       settingssvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service settingssvc.Service, authenticator authz.Authenticator) Handler {
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
	var req pomeloorbit.SystemConfigUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	resp, err := h.service.Update(c.Request.Context(), req.Key, transportresponse.NativeValue(req.Value))
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
	var req pomeloorbit.SystemConfigResetReq
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

func systemConfigResponse(config settingsdto.SystemConfig) pomeloorbit.SystemConfigResp {
	items := make([]pomeloorbit.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return pomeloorbit.SystemConfigResp{Items: transportresponse.Ptrs(items)}
}

func configItemResponse(item settingsdto.ConfigItem) pomeloorbit.ConfigItemResp {
	return pomeloorbit.ConfigItemResp{Key: item.Key, Value: transportresponse.ProtoValue(item.Value), Default: transportresponse.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
