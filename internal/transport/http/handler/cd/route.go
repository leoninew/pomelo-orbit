package cdhandler

import (
	"bytes"
	"encoding/json"
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
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, routeResponse)))
}

func (h Handler) createRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req RouteCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.CreateRoute(r.Context(), current.Id, r.URL.Query().Get("project_id"), cdsvc.RouteCreateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetURL, Enabled: req.Enabled})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, routeResponse(route))
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
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
}

func (h Handler) updateRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req RouteUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.UpdateRoute(r.Context(), current.Id, chi.URLParam(r, "route_id"), cdsvc.RouteUpdateInput{Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetURL, Enabled: req.Enabled})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
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
	if _, err := h.service.EnableRoute(r.Context(), current.Id, chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, RouteEnableResp{Message: "Route enabled successfully"})
}

func (h Handler) disableRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if _, err := h.service.DisableRoute(r.Context(), current.Id, chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, RouteDisableResp{Message: "Route disabled successfully"})
}

func (h Handler) syncRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.SyncRoutes(r.Context(), current.Id, r.URL.Query().Get("project_id")); err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, RouteSyncResp{Message: "Routes synced successfully"})
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
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
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
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
}

func (h Handler) enableRouteLetsEncrypt(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	route, err := h.service.EnableRouteLetsEncrypt(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
}

func (h Handler) enableRouteMkcert(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	route, err := h.service.EnableRouteMkcert(r.Context(), current.Id, chi.URLParam(r, "route_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, routeResponse(route))
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
	transportresponse.JSON(h.logger, w, http.StatusOK, traefikConfigResponse(config))
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
	transportresponse.JSON(h.logger, w, http.StatusOK, traefikRouteListResponse(items))
}

func routeResponse(route model.Route) RouteResp {
	return RouteResp{Id: route.Id, Name: route.Name, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetURL: route.TargetURL, Enabled: route.Enabled, HTTPSEnabled: route.HTTPSEnabled, CertType: route.CertType, CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

func traefikConfigResponse(config cdsvc.TraefikConfigResp) TraefikConfigResp {
	return TraefikConfigResp{DashboardDomain: config.DashboardDomain, HTTPSEnabled: config.HTTPSEnabled}
}

func traefikRouteListResponse(resp cdsvc.TraefikRouteListResp) TraefikRouteListResp {
	items := make([]TraefikRouterResp, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, traefikRouterResponse(item))
	}
	return TraefikRouteListResp{Items: items, Total: resp.Total}
}

func traefikRouterResponse(router cdsvc.TraefikRouterResp) TraefikRouterResp {
	return TraefikRouterResp{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: router.Entrypoints, TLS: router.TLS}
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
