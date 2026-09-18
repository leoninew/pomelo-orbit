package routehandler

import (
	"bytes"
	"encoding/pem"
	"net/http"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	routev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/route"
)

func (h Handler) ListRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRoutes(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponses(items.Items)
	transport.WriteProtoJSON(c, http.StatusOK, &routev1.RoutePaginatedResp{Items: transport.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transport.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.CreateRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), routeCreateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	route, err := h.service.RouteForUser(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.UpdateRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"), routeUpdateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) EnableRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteEnableReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if _, err := h.service.EnableRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &routev1.RouteEnableResp{Message: "Route enabled successfully"})
}

func (h Handler) DisableRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteDisableReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if _, err := h.service.DisableRoute(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &routev1.RouteDisableResp{Message: "Route disabled successfully"})
}

func (h Handler) PreviewRouteSync(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteSyncPreviewReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	preview, err := h.service.PreviewRouteSync(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), routeSyncChanges(req.Changes))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	response := routeSyncPreviewResponse(preview)
	transport.WriteProtoJSON(c, http.StatusOK, &response)
}

func (h Handler) ConfirmRouteSync(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteSyncConfirmReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.ConfirmRouteSync(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), routedto.RouteSyncConfirmInput{
		Changes:      routeSyncChanges(req.Changes),
		BusinessHash: req.BusinessHash,
		TraefikHash:  req.TraefikHash,
	}); err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &routev1.RouteSyncConfirmResp{Message: "Routes synced successfully"})
}

func (h Handler) UploadRouteCert(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	file, _, err := c.Request.FormFile("pem")
	if err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "pem is required")
		return
	}
	defer func() { _ = file.Close() }()
	var content bytes.Buffer
	if _, err := content.ReadFrom(file); err != nil {
		h.logger.Error("read certificate upload failed", "route_id", c.Param("route_id"), "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	certPEM, certKey, ok := splitPEM(content.Bytes())
	if !ok {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid PEM certificate")
		return
	}
	route, err := h.service.UploadRouteCert(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"), certPEM, certKey)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DisableRouteHTTPS(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	route, err := h.service.DisableRouteHTTPS(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) EnableRouteLetsEncrypt(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteLetsEncryptEnableReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.EnableRouteLetsEncrypt(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"), req.Challenge)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) EnableRouteMkcert(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req routev1.RouteMkcertEnableReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.EnableRouteMkcert(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("route_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := routeResponse(route)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetTraefikRouteConfig(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	config, err := h.service.TraefikRouteConfig(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := traefikConfigResponse(config)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ListTraefikRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListTraefikRoutes(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := traefikRouteListResponse(items)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
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
