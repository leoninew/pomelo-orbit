package cdsvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"

	"gopkg.in/yaml.v3"
)

const (
	certTypeManual      = "manual"
	certTypeLetsEncrypt = "letsencrypt"
	certTypeMkcert      = "mkcert"
)

var routeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)
var routeTargetURLPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+:\d+$`)

// RouteCreateInput creates a project route.
type RouteCreateInput struct {
	Name       string
	Domain     string
	PathPrefix string
	TargetURL  string
	Enabled    bool
}

// RouteUpdateInput updates a project route.
type RouteUpdateInput struct {
	Name       *string
	Domain     *string
	PathPrefix *string
	TargetURL  *string
	Enabled    *bool
}

// TraefikRouterResp mirrors the Traefik API response used by the HTTP handler.
type TraefikRouterResp struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Entrypoints []string `json:"entrypoints"`
	TLS         bool     `json:"tls"`
}

// TraefikConfigResp reports the dashboard route state.
type TraefikConfigResp struct {
	DashboardDomain string `json:"dashboard_domain"`
	HTTPSEnabled    bool   `json:"https_enabled"`
}

// TraefikRouteListResp wraps the Traefik router list.
type TraefikRouteListResp struct {
	Items []TraefikRouterResp `json:"items"`
	Total int                 `json:"total"`
}

// ListRoutes returns routes visible to the user.
func (s Service) ListRoutes(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Route]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Route]{}, err
	}
	items, err := s.store.ListRoutes(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Route]{}, apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
	}
	return items, nil
}

// CreateRoute creates a route and persists it.
func (s Service) CreateRoute(ctx context.Context, userId string, projectId string, input RouteCreateInput) (model.Route, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return model.Route{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Route{}, err
	}
	name, domain, pathPrefix, targetURL, enabled, err := normalizeRouteInput(input.Name, input.Domain, input.PathPrefix, input.TargetURL, input.Enabled)
	if err != nil {
		return model.Route{}, err
	}
	route := model.Route{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Domain: domain, PathPrefix: pathPrefix, TargetURL: targetURL, Enabled: enabled, HTTPSEnabled: false, CertType: certTypeManual}
	if err := s.store.CreateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to create route", err)
	}
	created, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if created.Enabled {
		if err := s.syncRouteFiles(ctx, created); err != nil {
			return model.Route{}, err
		}
	}
	return created, nil
}

// RouteForUser loads a route visible to the current user.
func (s Service) RouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	return s.loadRouteForUser(ctx, userId, routeId)
}

// UpdateRoute updates a route and synchronizes its files.
func (s Service) UpdateRoute(ctx context.Context, userId string, routeId string, input RouteUpdateInput) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	oldName := route.Name
	if input.Name != nil {
		route.Name = strings.TrimSpace(*input.Name)
	}
	if input.Domain != nil {
		route.Domain = strings.TrimSpace(*input.Domain)
	}
	if input.PathPrefix != nil {
		route.PathPrefix = strings.TrimSpace(*input.PathPrefix)
	}
	if input.TargetURL != nil {
		route.TargetURL = strings.TrimSpace(*input.TargetURL)
	}
	if input.Enabled != nil {
		route.Enabled = *input.Enabled
	}
	if !validRouteFields(route.Name, route.Domain, route.PathPrefix, route.TargetURL) {
		return model.Route{}, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if oldName != updated.Name {
		if err := s.revokeRouteFiles(ctx, oldName); err != nil {
			return model.Route{}, err
		}
		if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
			return model.Route{}, err
		}
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DeleteRoute removes a route after confirming it is disabled.
func (s Service) DeleteRoute(ctx context.Context, userId string, routeId string) error {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return err
	}
	if route.Enabled {
		return apperror.New(apperror.KindValidation, "Cannot delete enabled route. Please disable it first.")
	}
	if err := s.store.DeleteRoute(ctx, route.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete route", err)
	}
	if err := s.revokeRouteFiles(ctx, route.Name); err != nil {
		return err
	}
	return nil
}

// EnableRoute marks a route enabled and deploys its config.
func (s Service) EnableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = true
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable route", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DisableRoute marks a route disabled and removes its config.
func (s Service) DisableRoute(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.Enabled = false
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route", err)
	}
	if err := s.revokeRouteFiles(ctx, route.Name); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	return updated, nil
}

// SyncRoutes synchronizes every route in a project.
func (s Service) SyncRoutes(ctx context.Context, userId string, projectId string) error {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return err
	}
	routes, err := s.store.ListAllRoutes(ctx, projectId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list routes", err)
	}
	for _, route := range routes {
		if err := s.syncRouteFiles(ctx, route); err != nil {
			return err
		}
	}
	return nil
}

// UploadRouteCert stores a manual certificate and updates the route.
func (s Service) UploadRouteCert(ctx context.Context, userId string, routeId string, certPEM string, certKey string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &certKey
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to update route certificate", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// DisableRouteHTTPS clears HTTPS settings and removes certificate files.
func (s Service) DisableRouteHTTPS(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	oldName := route.Name
	route.HTTPSEnabled = false
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeManual
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to disable route HTTPS", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// EnableRouteLetsEncrypt enables Let's Encrypt for the route.
func (s Service) EnableRouteLetsEncrypt(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	if !s.cfg.Cert.LetsEncrypt.Enabled {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt not enabled. Please set cert.letsencrypt.enabled=true and configure email in config")
	}
	if strings.TrimSpace(s.cfg.Cert.LetsEncrypt.Email) == "" {
		return model.Route{}, apperror.New(apperror.KindValidation, "请在配置中设置 cert.letsencrypt.email")
	}
	if route.HTTPSEnabled && route.CertType == certTypeLetsEncrypt {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt 证书已启用")
	}
	if route.Domain == "localhost" || strings.HasSuffix(route.Domain, ".local") {
		return model.Route{}, apperror.New(apperror.KindValidation, "Let's Encrypt 不支持内网域名,请使用手动证书或 mkcert")
	}
	oldName := route.Name
	route.HTTPSEnabled = true
	route.CertPEM = nil
	route.CertKey = nil
	route.CertType = certTypeLetsEncrypt
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable Let's Encrypt", err)
	}
	if err := s.revokeRouteCertFiles(ctx, oldName); err != nil {
		return model.Route{}, err
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// EnableRouteMkcert generates a certificate with mkcert.
func (s Service) EnableRouteMkcert(ctx context.Context, userId string, routeId string) (model.Route, error) {
	route, err := s.loadRouteForUser(ctx, userId, routeId)
	if err != nil {
		return model.Route{}, err
	}
	certPEM, keyPEM, err := generateMkcert(ctx, route.Domain)
	if err != nil {
		return model.Route{}, err
	}
	route.HTTPSEnabled = true
	route.CertPEM = &certPEM
	route.CertKey = &keyPEM
	route.CertType = certTypeMkcert
	if err := s.store.UpdateRoute(ctx, route); err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to enable mkcert", err)
	}
	updated, err := s.store.Route(ctx, route.Id)
	if err != nil {
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if err := s.syncRouteFiles(ctx, updated); err != nil {
		return model.Route{}, err
	}
	return updated, nil
}

// TraefikRouteConfig reports the dashboard route state.
func (s Service) TraefikRouteConfig(ctx context.Context, userId string, projectId string) (TraefikConfigResp, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return TraefikConfigResp{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return TraefikConfigResp{}, err
	}
	dashboardDomain := fmt.Sprintf("traefik.%s", s.cfg.Traefik.DomainSuffix)
	route, err := s.store.RouteByDomain(ctx, dashboardDomain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TraefikConfigResp{DashboardDomain: dashboardDomain, HTTPSEnabled: false}, nil
		}
		return TraefikConfigResp{}, apperror.Wrap(apperror.KindInternal, "Failed to load Traefik route config", err)
	}
	return TraefikConfigResp{DashboardDomain: dashboardDomain, HTTPSEnabled: route.Enabled && route.HTTPSEnabled}, nil
}

// ListTraefikRoutes fetches routers from the Traefik API.
func (s Service) ListTraefikRoutes(ctx context.Context, userId string, projectId string) (TraefikRouteListResp, error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return TraefikRouteListResp{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return TraefikRouteListResp{}, err
	}
	items, err := fetchTraefikRouters(ctx, s.cfg.Traefik.APIURL)
	if err != nil {
		if isTraefikConnectionError(err) {
			return TraefikRouteListResp{}, apperror.New(apperror.KindValidation, fmt.Sprintf("无法连接到 Traefik: %v", err))
		}
		return TraefikRouteListResp{}, apperror.Wrap(apperror.KindInternal, "Failed to list Traefik routes", err)
	}
	return TraefikRouteListResp{Items: items, Total: len(items)}, nil
}

func (s Service) loadRouteForUser(ctx context.Context, userId string, routeId string) (model.Route, error) {
	routeId = strings.TrimSpace(routeId)
	route, err := s.store.Route(ctx, routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Route{}, apperror.New(apperror.KindNotFound, "Route "+routeId+" not found")
		}
		return model.Route{}, apperror.Wrap(apperror.KindInternal, "Failed to load route", err)
	}
	if route.ProjectId == nil {
		return model.Route{}, apperror.New(apperror.KindForbidden, "Permission denied")
	}
	if err := s.ensureProjectMembership(ctx, *route.ProjectId, userId); err != nil {
		return model.Route{}, err
	}
	return route, nil
}

func (s Service) syncRouteFiles(ctx context.Context, route model.Route) error {
	if route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil {
		if err := s.writeRouteCertFiles(ctx, route.Name, *route.CertPEM, *route.CertKey); err != nil {
			return err
		}
	}
	if route.Enabled {
		return s.deployRouteFile(ctx, route)
	}
	return s.revokeRouteFiles(ctx, route.Name)
}

func (s Service) deployRouteFile(ctx context.Context, route model.Route) error {
	configDir := s.routeConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to create route config directory", err)
	}
	data, err := yaml.Marshal(routeTraefikConfig(route))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to marshal route config", err)
	}
	path := filepath.Join(configDir, route.Name+".yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to write route config", err)
	}
	return s.reloadTraefik(ctx)
}

func (s Service) revokeRouteFiles(ctx context.Context, routeName string) error {
	path := filepath.Join(s.routeConfigDir(), routeName+".yml")
	if err := removeIfExists(path); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to revoke route config", err)
	}
	return s.reloadTraefik(ctx)
}

func (s Service) writeRouteCertFiles(ctx context.Context, routeName string, certPEM string, certKey string) error {
	certDir := s.routeCertDir()
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to create route cert directory", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, routeName+".pem"), []byte(certPEM), 0o600); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to write route certificate", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, routeName+"-key.pem"), []byte(certKey), 0o600); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to write route certificate key", err)
	}
	return nil
}

func (s Service) revokeRouteCertFiles(ctx context.Context, routeName string) error {
	for _, suffix := range []string{".pem", "-key.pem"} {
		path := filepath.Join(s.routeCertDir(), routeName+suffix)
		if err := removeIfExists(path); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to revoke route certificate", err)
		}
	}
	return s.reloadTraefik(ctx)
}

func (s Service) routeConfigDir() string {
	if strings.TrimSpace(s.cfg.Traefik.DynamicRouteDir) == "" {
		return filepath.Join(s.workspace.AppDir("traefik"), "data", "dynamic")
	}
	return cleanConfigPath(s.cfg.OrbitRoot(), s.cfg.Traefik.DynamicRouteDir)
}

func (s Service) routeCertDir() string {
	if strings.TrimSpace(s.cfg.Traefik.CertDir) == "" {
		return filepath.Join(s.workspace.AppDir("traefik"), "data", "certs")
	}
	return cleanConfigPath(s.cfg.OrbitRoot(), s.cfg.Traefik.CertDir)
}

func cleanConfigPath(root string, path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func (s Service) reloadTraefik(ctx context.Context) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	containerName := strings.TrimSpace(s.cfg.Traefik.ContainerName)
	if containerName == "" {
		containerName = "traefik"
	}
	ps := exec.CommandContext(ctx, "docker", "ps", "-q", "-f", "name="+containerName)
	output, err := ps.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return nil
	}
	if err := exec.CommandContext(ctx, "docker", "kill", "--signal=HUP", containerName).Run(); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to reload Traefik", err)
	}
	return nil
}

func routeTraefikConfig(route model.Route) map[string]any {
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

func isTraefikConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func fetchTraefikRouters(ctx context.Context, apiURL string) ([]TraefikRouterResp, error) {
	url := strings.TrimRight(strings.TrimSpace(apiURL), "/") + "/api/http/routers"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create traefik request: %w", err)
	}
	client := http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request traefik routers: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("traefik returned status %d", response.StatusCode)
	}
	var routers []struct {
		Name        string           `json:"name"`
		Provider    string           `json:"provider"`
		Status      string           `json:"status"`
		Rule        string           `json:"rule"`
		Service     string           `json:"service"`
		Entrypoints []string         `json:"entryPoints"`
		TLS         *json.RawMessage `json:"tls"`
	}
	if err := json.NewDecoder(response.Body).Decode(&routers); err != nil {
		return nil, fmt.Errorf("decode traefik routers: %w", err)
	}
	items := make([]TraefikRouterResp, 0, len(routers))
	for _, router := range routers {
		items = append(items, TraefikRouterResp{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: router.Entrypoints, TLS: router.TLS != nil && string(*router.TLS) != "null"})
	}
	return items, nil
}

func generateMkcert(ctx context.Context, domain string) (string, string, error) {
	if _, err := exec.LookPath("mkcert"); err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert 未安装或不可用,请参考 https://github.com/FiloSottile/mkcert#installation")
	}
	caRootOutput, err := exec.CommandContext(ctx, "mkcert", "-CAROOT").Output()
	if err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	caRoot := strings.TrimSpace(string(caRootOutput))
	if caRoot == "" {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	if _, err := os.Stat(filepath.Join(caRoot, "rootCA.pem")); err != nil {
		return "", "", apperror.New(apperror.KindValidation, "mkcert CA 未安装到系统信任库。请先运行 mkcert -install 然后重启浏览器")
	}
	tmp, err := os.MkdirTemp("", "pomelo-route-cert-*")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	certFile := filepath.Join(tmp, "cert.pem")
	keyFile := filepath.Join(tmp, "key.pem")
	cmd := exec.CommandContext(ctx, "mkcert", "-cert-file", certFile, "-key-file", keyFile, domain)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", "", apperror.New(apperror.KindValidation, "mkcert 生成证书失败: "+message)
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

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func normalizeRouteInput(name string, domain string, pathPrefix string, targetURL string, enabled bool) (string, string, string, string, bool, error) {
	name = strings.TrimSpace(name)
	domain = strings.TrimSpace(domain)
	pathPrefix = strings.TrimSpace(pathPrefix)
	targetURL = strings.TrimSpace(targetURL)
	if pathPrefix == "" {
		pathPrefix = "/"
	}
	if !validRouteFields(name, domain, pathPrefix, targetURL) {
		return "", "", "", "", false, apperror.New(apperror.KindValidation, "Invalid route fields")
	}
	return name, domain, pathPrefix, targetURL, enabled, nil
}

func validRouteFields(name string, domain string, pathPrefix string, targetURL string) bool {
	return routeNamePattern.MatchString(name) && domain != "" && strings.HasPrefix(pathPrefix, "/") && routeTargetURLPattern.MatchString(targetURL)
}
