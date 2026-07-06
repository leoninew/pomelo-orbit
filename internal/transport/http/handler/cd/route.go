package cdhandler

import (
	apiv1 "backend/internal/gen/orbit/api/v1"
	"bytes"
	"encoding/pem"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository/model"
	cdsvc "backend/internal/service/cd"
	transportresponse "backend/internal/transport/http/response"
)

func (h Handler) RegisterRouteRoutes(r router) {
	r.Get("/api/cd/route", h.listRoutes)
	r.Post("/api/cd/route", h.createRoute)
	r.Post("/api/cd/route/sync", h.syncRoutes)
	r.Get("/api/cd/route/{route_id}", h.getRoute)
	r.Put("/api/cd/route/{route_id}", h.updateRoute)
	r.Delete("/api/cd/route/{route_id}", h.deleteRoute)
	r.Post("/api/cd/route/{route_id}/enable", h.enableRoute)
	r.Post("/api/cd/route/{route_id}/disable", h.disableRoute)
	r.Post("/api/cd/route/{route_id}/cert", h.uploadRouteCert)
	r.Delete("/api/cd/route/{route_id}/https", h.disableRouteHTTPS)
	r.Post("/api/cd/route/{route_id}/letsencrypt", h.enableRouteLetsEncrypt)
	r.Post("/api/cd/route/{route_id}/mkcert", h.enableRouteMkcert)
}

func (h Handler) RegisterTraefikRouteRoutes(r router) {
	r.Get("/api/cd/traefik-route/config", h.getTraefikRouteConfig)
	r.Get("/api/cd/traefik-route", h.listTraefikRoutes)
}

func (h Handler) listRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRoutes(r.Context(), current.Id, r.URL.Query().Get("project_id"), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, routeResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RoutePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.CreateRoute(r.Context(), current.Id, r.URL.Query().Get("project_id"), cdsvc.RouteCreateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetUrl, Enabled: req.Enabled})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	route, err := h.service.RouteForUser(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.UpdateRoute(r.Context(), current.Id, chi.URLParam(r, "route_id"), cdsvc.RouteUpdateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetUrl, Enabled: req.Enabled})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteRoute(r.Context(), current.Id, chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) enableRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteEnableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if _, err := h.service.EnableRoute(r.Context(), current.Id, chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RouteEnableResp{Message: "Route enabled successfully"})
}

func (h Handler) disableRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteDisableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if _, err := h.service.DisableRoute(r.Context(), current.Id, chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RouteDisableResp{Message: "Route disabled successfully"})
}

func (h Handler) syncRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteSyncReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.SyncRoutes(r.Context(), current.Id, r.URL.Query().Get("project_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RouteSyncResp{Message: "Routes synced successfully"})
}

func (h Handler) uploadRouteCert(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	file, _, err := r.FormFile("pem")
	if err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "pem is required")
		return
	}
	defer func() { _ = file.Close() }()
	var content bytes.Buffer
	if _, err := content.ReadFrom(file); err != nil {
		h.logger.Error("read certificate upload failed", "route_id", chi.URLParam(r, "route_id"), "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to read certificate")
		return
	}
	certPEM, certKey, ok := splitPEM(content.Bytes())
	if !ok {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid PEM certificate")
		return
	}
	route, err := h.service.UploadRouteCert(r.Context(), current.Id, chi.URLParam(r, "route_id"), certPEM, certKey)
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) disableRouteHTTPS(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	route, err := h.service.DisableRouteHTTPS(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) enableRouteLetsEncrypt(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteLetsEncryptEnableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.EnableRouteLetsEncrypt(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) enableRouteMkcert(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RouteMkcertEnableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.EnableRouteMkcert(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := routeResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) getTraefikRouteConfig(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	config, err := h.service.TraefikRouteConfig(r.Context(), current.Id, r.URL.Query().Get("project_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := traefikConfigResponse(config)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) listTraefikRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListTraefikRoutes(r.Context(), current.Id, r.URL.Query().Get("project_id"))
	if err != nil {
		if strings.HasPrefix(err.Error(), "无法连接到 Traefik:") {
			transportresponse.Error(h.logger, w, http.StatusServiceUnavailable, err.Error())
			return
		}
		h.writeError(w, err)
		return
	}
	resp := traefikRouteListResponse(items)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func routeResponse(route model.Route) apiv1.RouteResp {
	return apiv1.RouteResp{Id: route.Id, Name: route.Name, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetUrl: route.TargetURL, Enabled: route.Enabled, HttpsEnabled: route.HTTPSEnabled, CertType: route.CertType, CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

func traefikConfigResponse(config cdsvc.TraefikConfigResp) apiv1.TraefikConfigResp {
	return apiv1.TraefikConfigResp{DashboardDomain: config.DashboardDomain, HttpsEnabled: config.HTTPSEnabled}
}

func traefikRouteListResponse(resp cdsvc.TraefikRouteListResp) apiv1.TraefikRouteListResp {
	items := make([]apiv1.TraefikRouterResp, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, traefikRouterResponse(item))
	}
	return apiv1.TraefikRouteListResp{Items: transportresponse.Ptrs(items), Total: int32(resp.Total)}
}

func traefikRouterResponse(router cdsvc.TraefikRouterResp) apiv1.TraefikRouterResp {
	return apiv1.TraefikRouterResp{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: router.Entrypoints, Tls: router.TLS}
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
