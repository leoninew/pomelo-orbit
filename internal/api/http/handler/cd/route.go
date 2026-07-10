package cdhandler

import (
	"bytes"
	"encoding/pem"
	"net/http"
	"strings"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (h Handler) RegisterRouteRoutes(r router) {
	r.GET("/api/cd/route", h.listRoutes)
	r.POST("/api/cd/route", h.createRoute)
	r.POST("/api/cd/route/sync", h.syncRoutes)
	r.GET("/api/cd/route/:route_id", h.getRoute)
	r.PUT("/api/cd/route/:route_id", h.updateRoute)
	r.DELETE("/api/cd/route/:route_id", h.deleteRoute)
	r.POST("/api/cd/route/:route_id/enable", h.enableRoute)
	r.POST("/api/cd/route/:route_id/disable", h.disableRoute)
	r.POST("/api/cd/route/:route_id/cert", h.uploadRouteCert)
	r.DELETE("/api/cd/route/:route_id/https", h.disableRouteHTTPS)
	r.POST("/api/cd/route/:route_id/letsencrypt", h.enableRouteLetsEncrypt)
	r.POST("/api/cd/route/:route_id/mkcert", h.enableRouteMkcert)
}

func (h Handler) RegisterTraefikRouteRoutes(r router) {
	r.GET("/api/cd/traefik-route/config", h.getTraefikRouteConfig)
	r.GET("/api/cd/traefik-route", h.listTraefikRoutes)
}

func (h Handler) listRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRoutes(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, routeResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RoutePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.CreateRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), cddto.RouteCreateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetUrl, Enabled: req.Enabled})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	route, err := h.service.RouteForUser(c.Request.Context(), current.Id, c.Param("route_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.UpdateRoute(c.Request.Context(), current.Id, c.Param("route_id"), cddto.RouteUpdateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetUrl, Enabled: req.Enabled})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteRoute(c.Request.Context(), current.Id, c.Param("route_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) enableRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteEnableReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if _, err := h.service.EnableRoute(c.Request.Context(), current.Id, c.Param("route_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RouteEnableResp{Message: "Route enabled successfully"}})
}

func (h Handler) disableRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteDisableReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if _, err := h.service.DisableRoute(c.Request.Context(), current.Id, c.Param("route_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RouteDisableResp{Message: "Route disabled successfully"}})
}

func (h Handler) syncRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteSyncReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if err := h.service.SyncRoutes(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RouteSyncResp{Message: "Routes synced successfully"}})
}

func (h Handler) uploadRouteCert(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	file, _, err := c.Request.FormFile("pem")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "pem is required"})
		return
	}
	defer func() { _ = file.Close() }()
	var content bytes.Buffer
	if _, err := content.ReadFrom(file); err != nil {
		h.logger.Error("read certificate upload failed", "route_id", c.Param("route_id"), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to read certificate"})
		return
	}
	certPEM, certKey, ok := splitPEM(content.Bytes())
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid PEM certificate"})
		return
	}
	route, err := h.service.UploadRouteCert(c.Request.Context(), current.Id, c.Param("route_id"), certPEM, certKey)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) disableRouteHTTPS(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	route, err := h.service.DisableRouteHTTPS(c.Request.Context(), current.Id, c.Param("route_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) enableRouteLetsEncrypt(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteLetsEncryptEnableReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.EnableRouteLetsEncrypt(c.Request.Context(), current.Id, c.Param("route_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) enableRouteMkcert(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RouteMkcertEnableReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.EnableRouteMkcert(c.Request.Context(), current.Id, c.Param("route_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := routeResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getTraefikRouteConfig(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	config, err := h.service.TraefikRouteConfig(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := traefikConfigResponse(config)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) listTraefikRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListTraefikRoutes(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"))
	if err != nil {
		if strings.HasPrefix(err.Error(), "无法连接到 Traefik:") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"detail": err.Error()})
			return
		}
		h.writeError(c, err)
		return
	}
	resp := traefikRouteListResponse(items)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func routeResponse(route model.Route) pomeloorbit.RouteResp {
	return pomeloorbit.RouteResp{Id: route.Id, Name: route.Name, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetUrl: route.TargetURL, Enabled: route.Enabled, HttpsEnabled: route.HTTPSEnabled, CertType: route.CertType, CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

func traefikConfigResponse(config cddto.TraefikConfigResp) pomeloorbit.TraefikConfigResp {
	return pomeloorbit.TraefikConfigResp{DashboardDomain: config.DashboardDomain, HttpsEnabled: config.HTTPSEnabled}
}

func traefikRouteListResponse(resp cddto.TraefikRouteListResp) pomeloorbit.TraefikRouteListResp {
	items := make([]pomeloorbit.TraefikRouterResp, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, traefikRouterResponse(item))
	}
	return pomeloorbit.TraefikRouteListResp{Items: transportresponse.Ptrs(items), Total: int32(resp.Total)}
}

func traefikRouterResponse(router cddto.TraefikRouterResp) pomeloorbit.TraefikRouterResp {
	return pomeloorbit.TraefikRouterResp{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: append([]string(nil), router.Entrypoints...), Tls: router.TLS}
}

func splitPEM(content []byte) (string, string, bool) {
	var cert bytes.Buffer
	var key bytes.Buffer
	remaining := content
	for {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}
		encoded := pem.EncodeToMemory(block)
		if block.Type == "CERTIFICATE" {
			cert.Write(encoded)
		} else if strings.Contains(block.Type, "PRIVATE KEY") {
			key.Write(encoded)
		}
		if len(rest) == len(remaining) {
			break
		}
		remaining = rest
	}
	if cert.Len() == 0 || key.Len() == 0 {
		return "", "", false
	}
	return cert.String(), key.String(), true
}
