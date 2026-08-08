package traefik

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	routeport "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/port"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var _ routeport.RouteConfigPublisher = (*RouteManager)(nil)
var _ routeport.TraefikRouterClient = (*RouteManager)(nil)

const deploymentDataDir = "deployment"

// RouteManager publishes platform routes via Traefik providers.rest full PUT.
type RouteManager struct {
	cfg    config.Config
	client *http.Client
	mu     sync.Mutex
}

func NewRouteManager(cfg config.Config) *RouteManager {
	return &RouteManager{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ApplySnapshot replaces the entire @rest HTTP configuration with the given enabled routes.
// restAPIURL is the Gateway config control-plane base URL (required).
func (m *RouteManager) ApplySnapshot(ctx context.Context, restApiUrl string, routes []model.Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, route := range routes {
		if route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil && strings.TrimSpace(*route.CertPEM) != "" {
			if err := m.writeCertificateUnlocked(route.Name, *route.CertPEM, *route.CertKey); err != nil {
				return err
			}
		}
	}

	body, err := json.Marshal(buildRestSnapshot(routes))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to marshal traefik rest snapshot", err)
	}
	return m.putRestConfig(ctx, restApiUrl, body)
}

func (m *RouteManager) WriteCertificate(_ context.Context, routeName string, certPEM string, certKey string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeCertificateUnlocked(routeName, certPEM, certKey)
}

func (m *RouteManager) RevokeCertificate(_ context.Context, routeName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, suffix := range []string{".pem", "-key.pem"} {
		path := filepath.Join(m.routeCertDir(), routeName+suffix)
		if err := removeIfExists(path); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to revoke route certificate", err)
		}
	}
	return nil
}

func (m *RouteManager) ListRouters(ctx context.Context, restApiUrl string) ([]routeport.TraefikRouter, error) {
	base := strings.TrimRight(strings.TrimSpace(restApiUrl), "/")
	if base == "" {
		return nil, apperror.New(apperror.KindValidation, "gateway rest_api_url is required")
	}
	url := base + "/api/http/routers"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create traefik request: %w", err)
	}
	response, err := m.client.Do(request)
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
	items := make([]routeport.TraefikRouter, 0, len(routers))
	for _, router := range routers {
		items = append(items, routeport.TraefikRouter{
			Name:        router.Name,
			Provider:    router.Provider,
			Status:      router.Status,
			Rule:        router.Rule,
			Service:     router.Service,
			Entrypoints: append([]string(nil), router.Entrypoints...),
			TLS:         router.TLS != nil && string(*router.TLS) != "null",
		})
	}
	return items, nil
}

func (m *RouteManager) IsConnectionError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func (m *RouteManager) putRestConfig(ctx context.Context, restApiUrl string, body []byte) error {
	base := strings.TrimRight(strings.TrimSpace(restApiUrl), "/")
	if base == "" {
		return apperror.New(apperror.KindValidation, "gateway rest_api_url is required for rest route publish")
	}
	url := base + "/api/providers/rest"
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to create traefik rest request", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to put traefik rest config", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return apperror.New(apperror.KindInternal, fmt.Sprintf("traefik rest PUT returned status %d: %s", response.StatusCode, strings.TrimSpace(string(snippet))))
	}
	return nil
}

func (m *RouteManager) writeCertificateUnlocked(routeName string, certPEM string, certKey string) error {
	certDir := m.routeCertDir()
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

func (m *RouteManager) routeCertDir() string {
	if strings.TrimSpace(m.cfg.Traefik.CertDir) == "" {
		return filepath.Join(m.cfg.DataRoot(), deploymentDataDir, "traefik", "data", "certs")
	}
	return cleanConfigPath(m.cfg.OrbitRoot(), m.cfg.Traefik.CertDir)
}

func cleanConfigPath(root string, path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

// buildRestSnapshot assembles a full providers.rest HTTP config (full replace semantics).
func buildRestSnapshot(routes []model.Route) map[string]any {
	routers := map[string]any{}
	services := map[string]any{}
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		serviceName := sanitizeTraefikName(route.Name) + "-service"
		routerName := sanitizeTraefikName(route.Name) + "-route"
		rule := "Host(`" + route.Domain + "`)"
		if route.PathPrefix != "" && route.PathPrefix != "/" {
			rule += " && PathPrefix(`" + route.PathPrefix + "`)"
		}
		router := map[string]any{
			"rule":    rule,
			"service": serviceName,
		}
		if route.HTTPSEnabled {
			router["entryPoints"] = []string{"websecure"}
			if route.CertType == "letsencrypt" {
				router["tls"] = map[string]any{"certResolver": "letsencrypt"}
			} else {
				router["tls"] = map[string]any{}
			}
		} else {
			router["entryPoints"] = []string{"web"}
		}
		routers[routerName] = router
		services[serviceName] = map[string]any{
			"loadBalancer": map[string]any{
				"servers": []map[string]string{{"url": route.TargetUrl}},
			},
		}
	}
	// Empty maps clear the @rest namespace (unlike {} / {"http":{}}).
	return map[string]any{
		"http": map[string]any{
			"routers":  routers,
			"services": services,
		},
	}
}

func sanitizeTraefikName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "route"
	}
	return name
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
