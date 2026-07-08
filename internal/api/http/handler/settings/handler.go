package settingshandler

import (
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
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
	r.GET("/api/settings/config", h.getConfig)
	r.PUT("/api/settings/config", h.updateConfig)
	r.DELETE("/api/settings/config", h.resetConfig)
}

func (h Handler) getConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:read"); !ok {
		return
	}
	resp, err := h.service.Config(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &body})
}

func (h Handler) updateConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req pomeloorbit.SystemConfigUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	resp, err := h.service.Update(c.Request.Context(), req.Key, transportresponse.NativeValue(req.Value))
	if err != nil {
		h.writeError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &body})
}

func (h Handler) resetConfig(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "setting:write"); !ok {
		return
	}
	var req pomeloorbit.SystemConfigResetReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	resp, err := h.service.Reset(c.Request.Context(), req.Keys)
	if err != nil {
		h.writeError(c, err)
		return
	}
	body := systemConfigResponse(resp)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &body})
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("settings request failed", "error", err)
	}
	c.JSON(apperror.StatusCode(err), gin.H{"detail": err.Error()})
}

func systemConfigResponse(config settingssvc.SystemConfig) pomeloorbit.SystemConfigResp {
	items := make([]pomeloorbit.ConfigItemResp, 0, len(config.Items))
	for _, item := range config.Items {
		items = append(items, configItemResponse(item))
	}
	return pomeloorbit.SystemConfigResp{Items: transportresponse.Ptrs(items)}
}

func configItemResponse(item settingssvc.ConfigItem) pomeloorbit.ConfigItemResp {
	return pomeloorbit.ConfigItemResp{Key: item.Key, Value: transportresponse.ProtoValue(item.Value), Default: transportresponse.ProtoValue(item.Default), IsOverridden: item.IsOverridden, Secret: item.Secret, Description: item.Description}
}
