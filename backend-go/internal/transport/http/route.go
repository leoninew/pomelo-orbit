package transporthttp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"backend/internal/repository"
	"gopkg.in/yaml.v3"
)

const (
	certTypeManual      = "manual"
	certTypeLetsEncrypt = "letsencrypt"
	certTypeMkcert      = "mkcert"
)

var routeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
var routeTargetURLPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+:\d+$`)

type routeResp struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	Domain       string `json:"domain"`
	PathPrefix   string `json:"path_prefix"`
	TargetURL    string `json:"target_url"`
	Enabled      bool   `json:"enabled"`
	HTTPSEnabled bool   `json:"https_enabled"`
	CertType     string `json:"cert_type"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type routeCreateReq struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	PathPrefix string `json:"path_prefix"`
	TargetURL  string `json:"target_url"`
	Enabled    bool   `json:"enabled"`
}

type routeUpdateReq struct {
	Name       *string `json:"name"`
	Domain     *string `json:"domain"`
	PathPrefix *string `json:"path_prefix"`
	TargetURL  *string `json:"target_url"`
	Enabled    *bool   `json:"enabled"`
}

func (s Server) listRoutes(w http.ResponseWriter, r *http.Request) {
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
	page, perPage := pageParams(r)
	items, err := s.store.ListRoutes(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list routes failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list routes"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, routeResponse)))
}

func (s Server) createRoute(w http.ResponseWriter, r *http.Request) {
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
	var req routeCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeRouteCreateReq(w, &req) {
		return
	}
	route := repository.Route{Id: repository.NewId(), ProjectId: &projectId, Name: req.Name, Domain: req.Domain, PathPrefix: req.PathPrefix, TargetURL: req.TargetURL, Enabled: req.Enabled, HTTPSEnabled: false, CertType: certTypeManual}
	if err := s.store.CreateRoute(r.Context(), route); err != nil {
		s.logger.Error("create route failed", "route_name", route.Name, "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create route"})
		return
	}
	created, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load created route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if created.Enabled && !s.syncRouteFiles(w, r, created) {
		return
	}
	writeJSON(w, http.StatusCreated, routeResponse(created))
}

func (s Server) getRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(route))
}

func (s Server) updateRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	oldName := route.Name
	var req routeUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !applyRouteUpdateReq(w, &route, req) {
		return
	}
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("update route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update route"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load updated route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if oldName != updated.Name {
		if !s.revokeRouteFiles(w, r, oldName) || !s.revokeRouteCertFiles(w, r, oldName) {
			return
		}
	}
	if !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(updated))
}

func (s Server) deleteRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	if route.Enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot delete enabled route. Please disable it first."})
		return
	}
	if err := s.store.DeleteRoute(r.Context(), route.Id); err != nil {
		s.logger.Error("delete route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete route"})
		return
	}
	if !s.revokeRouteFiles(w, r, route.Name) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) enableRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	route.Enabled = true
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("enable route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enable route"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load enabled route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Route enabled successfully"})
}

func (s Server) disableRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	route.Enabled = false
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("disable route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to disable route"})
		return
	}
	if !s.revokeRouteFiles(w, r, route.Name) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Route disabled successfully"})
}

func (s Server) syncRoutes(w http.ResponseWriter, r *http.Request) {
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
	routes, err := s.store.ListAllRoutes(r.Context(), projectId)
	if err != nil {
		s.logger.Error("list routes for sync failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list routes"})
		return
	}
	for _, route := range routes {
		if !s.syncRouteFiles(w, r, route) {
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Routes synced successfully"})
}

func (s Server) uploadRouteCert(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	file, _, err := r.FormFile("pem")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "pem is required"})
		return
	}
	defer func() { _ = file.Close() }()
	var content bytes.Buffer
	if _, err := content.ReadFrom(file); err != nil {
		s.logger.Error("read certificate upload failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read certificate"})
		return
	}
	certPEM, certKey, ok := splitPEM(content.Bytes())
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid PEM certificate"})
		return
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &certKey
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("upload route certificate failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update route certificate"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load certificate route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(updated))
}

func (s Server) disableRouteHTTPS(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	oldName := route.Name
	route.HTTPSEnabled = false
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("disable route https failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to disable route HTTPS"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load route after https disable failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if !s.revokeRouteCertFiles(w, r, oldName) || !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(updated))
}

func (s Server) enableRouteLetsEncrypt(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	if !s.appCfg.Cert.LetsEncrypt.Enabled {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Let's Encrypt not enabled. Please set cert.letsencrypt.enabled=true and configure email in config"})
		return
	}
	if strings.TrimSpace(s.appCfg.Cert.LetsEncrypt.Email) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "请在配置中设置 cert.letsencrypt.email"})
		return
	}
	if route.HTTPSEnabled && route.CertType == certTypeLetsEncrypt {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Let's Encrypt 证书已启用"})
		return
	}
	if route.Domain == "localhost" || strings.HasSuffix(route.Domain, ".local") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Let's Encrypt 不支持内网域名,请使用手动证书或 mkcert"})
		return
	}
	oldName := route.Name
	route.HTTPSEnabled = true
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeLetsEncrypt
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("enable route letsencrypt failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enable Let's Encrypt"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load letsencrypt route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if !s.revokeRouteCertFiles(w, r, oldName) || !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(updated))
}

func (s Server) enableRouteMkcert(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	certPEM, keyPEM, err := generateMkcert(r, route.Domain)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &keyPEM
	route.CertType = certTypeMkcert
	if err := s.store.UpdateRoute(r.Context(), route); err != nil {
		s.logger.Error("enable route mkcert failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enable mkcert"})
		return
	}
	updated, err := s.store.Route(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load mkcert route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return
	}
	if !s.syncRouteFiles(w, r, updated) {
		return
	}
	writeJSON(w, http.StatusOK, routeResponse(updated))
}

func routeResponse(route repository.Route) routeResp {
	return routeResp{Id: route.Id, Name: route.Name, Domain: route.Domain, PathPrefix: route.PathPrefix, TargetURL: route.TargetURL, Enabled: route.Enabled, HTTPSEnabled: route.HTTPSEnabled, CertType: route.CertType, CreatedAt: formatTime(route.CreatedAt), UpdatedAt: formatTime(route.UpdatedAt)}
}

func normalizeRouteCreateReq(w http.ResponseWriter, req *routeCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Domain = strings.TrimSpace(req.Domain)
	req.PathPrefix = strings.TrimSpace(req.PathPrefix)
	req.TargetURL = strings.TrimSpace(req.TargetURL)
	if req.PathPrefix == "" {
		req.PathPrefix = "/"
	}
	if !validRouteFields(req.Name, req.Domain, req.PathPrefix, req.TargetURL) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid route fields"})
		return false
	}
	return true
}

func applyRouteUpdateReq(w http.ResponseWriter, route *repository.Route, req routeUpdateReq) bool {
	if req.Name != nil {
		route.Name = strings.TrimSpace(*req.Name)
	}
	if req.Domain != nil {
		route.Domain = strings.TrimSpace(*req.Domain)
	}
	if req.PathPrefix != nil {
		route.PathPrefix = strings.TrimSpace(*req.PathPrefix)
	}
	if req.TargetURL != nil {
		route.TargetURL = strings.TrimSpace(*req.TargetURL)
	}
	if req.Enabled != nil {
		route.Enabled = *req.Enabled
	}
	if !validRouteFields(route.Name, route.Domain, route.PathPrefix, route.TargetURL) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid route fields"})
		return false
	}
	return true
}

func validRouteFields(name string, domain string, pathPrefix string, targetURL string) bool {
	return routeNamePattern.MatchString(name) && domain != "" && strings.HasPrefix(pathPrefix, "/") && routeTargetURLPattern.MatchString(targetURL)
}

func (s Server) loadRouteForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.Route, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return repository.Route{}, false
	}
	routeId := urlParam(r, "route_id")
	route, err := s.store.Route(r.Context(), routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Route " + routeId + " not found"})
			return repository.Route{}, false
		}
		s.logger.Error("load route failed", "route_id", routeId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load route"})
		return repository.Route{}, false
	}
	if route.ProjectId == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
		return repository.Route{}, false
	}
	if !s.ensureProjectMembership(w, r, *route.ProjectId, current.Id) {
		return repository.Route{}, false
	}
	return route, true
}

func (s Server) syncRouteFiles(w http.ResponseWriter, r *http.Request, route repository.Route) bool {
	if route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil {
		if !s.writeRouteCertFiles(w, route.Name, *route.CertPEM, *route.CertKey) {
			return false
		}
	}
	if route.Enabled {
		return s.deployRouteFile(w, r, route)
	}
	return s.revokeRouteFiles(w, r, route.Name)
}

func (s Server) deployRouteFile(w http.ResponseWriter, r *http.Request, route repository.Route) bool {
	configDir := s.routeConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		s.logger.Error("create route config directory failed", "path", configDir, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create route config directory"})
		return false
	}
	data, err := yaml.Marshal(routeTraefikConfig(route))
	if err != nil {
		s.logger.Error("marshal route config failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to marshal route config"})
		return false
	}
	path := filepath.Join(configDir, route.Name+".yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		s.logger.Error("write route config failed", "route_id", route.Id, "path", path, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to write route config"})
		return false
	}
	return s.reloadTraefik(w, r)
}

func (s Server) revokeRouteFiles(w http.ResponseWriter, r *http.Request, routeName string) bool {
	path := filepath.Join(s.routeConfigDir(), routeName+".yml")
	if err := removeIfExists(path); err != nil {
		s.logger.Error("revoke route config failed", "route_name", routeName, "path", path, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to revoke route config"})
		return false
	}
	return s.reloadTraefik(w, r)
}

func (s Server) writeRouteCertFiles(w http.ResponseWriter, routeName string, certPEM string, certKey string) bool {
	certDir := s.routeCertDir()
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		s.logger.Error("create route cert directory failed", "path", certDir, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create route cert directory"})
		return false
	}
	if err := os.WriteFile(filepath.Join(certDir, routeName+".pem"), []byte(certPEM), 0o600); err != nil {
		s.logger.Error("write route cert failed", "route_name", routeName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to write route certificate"})
		return false
	}
	if err := os.WriteFile(filepath.Join(certDir, routeName+"-key.pem"), []byte(certKey), 0o600); err != nil {
		s.logger.Error("write route cert key failed", "route_name", routeName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to write route certificate key"})
		return false
	}
	return true
}

func (s Server) revokeRouteCertFiles(w http.ResponseWriter, r *http.Request, routeName string) bool {
	for _, suffix := range []string{".pem", "-key.pem"} {
		path := filepath.Join(s.routeCertDir(), routeName+suffix)
		if err := removeIfExists(path); err != nil {
			s.logger.Error("revoke route cert failed", "route_name", routeName, "path", path, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to revoke route certificate"})
			return false
		}
	}
	return s.reloadTraefik(w, r)
}

func (s Server) routeConfigDir() string {
	if strings.TrimSpace(s.appCfg.Traefik.DynamicRouteDir) == "" {
		return filepath.Join(s.appCfg.DataRoot(), "cd", "traefik", "data", "dynamic")
	}
	return cleanConfigPath(s.appCfg.OrbitRoot(), s.appCfg.Traefik.DynamicRouteDir)
}

func (s Server) routeCertDir() string {
	if strings.TrimSpace(s.appCfg.Traefik.CertDir) == "" {
		return filepath.Join(s.appCfg.DataRoot(), "cd", "traefik", "data", "certs")
	}
	return cleanConfigPath(s.appCfg.OrbitRoot(), s.appCfg.Traefik.CertDir)
}

func cleanConfigPath(root string, path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func (s Server) reloadTraefik(w http.ResponseWriter, r *http.Request) bool {
	if runtime.GOOS != "windows" {
		return true
	}
	containerName := strings.TrimSpace(s.appCfg.Traefik.ContainerName)
	if containerName == "" {
		containerName = "traefik"
	}
	ps := exec.CommandContext(r.Context(), "docker", "ps", "-q", "-f", "name="+containerName)
	output, err := ps.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return true
	}
	if err := exec.CommandContext(r.Context(), "docker", "kill", "--signal=HUP", containerName).Run(); err != nil {
		s.logger.Error("reload traefik failed", "container", containerName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to reload Traefik"})
		return false
	}
	return true
}

func routeTraefikConfig(route repository.Route) map[string]any {
	serviceName := route.Name + "-service"
	routerName := route.Name + "-route"
	rule := "Host(`" + route.Domain + "`)"
	if route.PathPrefix != "/" {
		rule += " && PathPrefix(`" + route.PathPrefix + "`)"
	}
	router := map[string]any{"rule": rule, "service": serviceName}
	if route.HTTPSEnabled {
		router["entryPoints"] = []string{"websecure"}
		if route.CertType == certTypeLetsEncrypt {
			router["tls"] = map[string]any{"certResolver": "letsencrypt"}
		} else if route.CertPEM != nil {
			router["tls"] = map[string]any{}
		}
	} else {
		router["entryPoints"] = []string{"web"}
	}
	config := map[string]any{
		"http": map[string]any{
			"routers":  map[string]any{routerName: router},
			"services": map[string]any{serviceName: map[string]any{"loadBalancer": map[string]any{"servers": []map[string]string{{"url": route.TargetURL}}}}},
		},
	}
	if route.HTTPSEnabled && route.CertType != certTypeLetsEncrypt && route.CertPEM != nil {
		config["tls"] = map[string]any{"certificates": []map[string]string{{"certFile": "/etc/traefik/certs/" + route.Name + ".pem", "keyFile": "/etc/traefik/certs/" + route.Name + "-key.pem"}}}
	}
	return config
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
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

func generateMkcert(r *http.Request, domain string) (string, string, error) {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return "", "", errors.New("mkcert 未安装或不可用,请参考 https://github.com/FiloSottile/mkcert#installation")
	}
	caRootOutput, err := exec.CommandContext(r.Context(), "mkcert", "-CAROOT").Output()
	if err != nil {
		return "", "", errors.New("mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	caRoot := strings.TrimSpace(string(caRootOutput))
	if caRoot == "" {
		return "", "", errors.New("mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	if _, err := os.Stat(filepath.Join(caRoot, "rootCA.pem")); err != nil {
		return "", "", errors.New("mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	tmp, err := os.MkdirTemp("", "pomelo-route-cert-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	certFile := filepath.Join(tmp, "cert.pem")
	keyFile := filepath.Join(tmp, "key.pem")
	cmd := exec.CommandContext(r.Context(), "mkcert", "-cert-file", certFile, "-key-file", keyFile, domain)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", "", errors.New("mkcert 生成证书失败: " + message)
	}
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return "", "", err
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return "", "", err
	}
	return string(certPEM), string(keyPEM), nil
}
