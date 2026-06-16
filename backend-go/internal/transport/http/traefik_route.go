package transporthttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const traefikRouteClientTimeout = 10 * time.Second

type traefikConfigResp struct {
	DashboardDomain string `json:"dashboard_domain"`
	HTTPSEnabled    bool   `json:"https_enabled"`
}

type traefikRouteListResp struct {
	Items []traefikRouterResp `json:"items"`
	Total int                 `json:"total"`
}

type traefikRouterResp struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Entrypoints []string `json:"entrypoints"`
	TLS         bool     `json:"tls"`
}

type traefikRouterAPIResp struct {
	Name        string           `json:"name"`
	Provider    string           `json:"provider"`
	Status      string           `json:"status"`
	Rule        string           `json:"rule"`
	Service     string           `json:"service"`
	Entrypoints []string         `json:"entryPoints"`
	TLS         *json.RawMessage `json:"tls"`
}

func (s Server) getTraefikRouteConfig(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}

	dashboardDomain := fmt.Sprintf("traefik.%s", s.appCfg.Traefik.DomainSuffix)
	route, err := s.store.RouteByDomain(r.Context(), dashboardDomain)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("load traefik route config failed", "domain", dashboardDomain, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load Traefik route config"})
		return
	}

	writeJSON(w, http.StatusOK, traefikConfigResp{
		DashboardDomain: dashboardDomain,
		HTTPSEnabled:    err == nil && route.Enabled && route.HTTPSEnabled,
	})
}

func (s Server) listTraefikRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}

	items, err := fetchTraefikRouters(r.Context(), s.appCfg.Traefik.APIURL)
	if err != nil {
		if isTraefikConnectionError(err) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": fmt.Sprintf("无法连接到 Traefik: %v", err)})
			return
		}
		s.logger.Error("list traefik routes failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list Traefik routes"})
		return
	}

	writeJSON(w, http.StatusOK, traefikRouteListResp{Items: items, Total: len(items)})
}

func fetchTraefikRouters(ctx context.Context, apiURL string) ([]traefikRouterResp, error) {
	url := strings.TrimRight(strings.TrimSpace(apiURL), "/") + "/api/http/routers"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create traefik request: %w", err)
	}
	client := http.Client{Timeout: traefikRouteClientTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request traefik routers: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("traefik returned status %d", response.StatusCode)
	}

	var routers []traefikRouterAPIResp
	if err := json.NewDecoder(response.Body).Decode(&routers); err != nil {
		return nil, fmt.Errorf("decode traefik routers: %w", err)
	}
	items := make([]traefikRouterResp, 0, len(routers))
	for _, router := range routers {
		items = append(items, traefikRouterResp{
			Name:        router.Name,
			Provider:    router.Provider,
			Status:      router.Status,
			Rule:        router.Rule,
			Service:     router.Service,
			Entrypoints: router.Entrypoints,
			TLS:         router.TLS != nil && string(*router.TLS) != "null",
		})
	}
	return items, nil
}

func isTraefikConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}
