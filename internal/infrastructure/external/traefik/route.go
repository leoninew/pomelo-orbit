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
	"strconv"
	"strings"
	"sync"
	"time"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var _ routeport.RouteConfigPublisher = (*RouteManager)(nil)
var _ routeport.TraefikRouterClient = (*RouteManager)(nil)

const (
	restApiReadyPollInterval = 500 * time.Millisecond
)

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

// WaitUntilReady polls the Traefik API until it responds successfully.
// compose up -d returning zero does not mean the REST control plane is listening yet.
func (m *RouteManager) WaitUntilReady(ctx context.Context, restApiUrl string, timeout time.Duration) error {
	base := strings.TrimRight(strings.TrimSpace(restApiUrl), "/")
	if base == "" {
		return apperror.New(apperror.KindValidation, "gateway rest_api_url is required")
	}
	url := base + "/api/overview"
	if timeout <= 0 {
		return apperror.New(apperror.KindValidation, "gateway rest_ready_timeout_seconds is required")
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return apperror.Wrap(apperror.KindInternal, "Traefik REST API readiness canceled", lastErr)
			}
			return apperror.Wrap(apperror.KindInternal, "Traefik REST API readiness canceled", err)
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to create Traefik readiness request", err)
		}
		response, err := m.client.Do(request)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 512))
			_ = response.Body.Close()
			if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
				return nil
			}
			lastErr = fmt.Errorf("traefik readiness returned status %d", response.StatusCode)
		} else {
			lastErr = err
		}
		if !time.Now().Before(deadline) {
			return apperror.Wrap(apperror.KindInternal, "Traefik REST API not ready within timeout", lastErr)
		}
		timer := time.NewTimer(restApiReadyPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			if lastErr != nil {
				return apperror.Wrap(apperror.KindInternal, "Traefik REST API readiness canceled", lastErr)
			}
			return apperror.Wrap(apperror.KindInternal, "Traefik REST API readiness canceled", ctx.Err())
		case <-timer.C:
		}
	}
}

// ApplySnapshot replaces the entire @rest HTTP and TCP configuration with the given enabled routes.
func (m *RouteManager) ApplySnapshot(ctx context.Context, gateway model.GatewayConfig, routes []model.Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, route := range routes {
		if route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil && strings.TrimSpace(*route.CertPEM) != "" {
			if err := m.writeCertificateUnlocked(gateway, route.Name, *route.CertPEM, *route.CertKey); err != nil {
				return err
			}
		}
	}

	body, err := json.Marshal(buildRestSnapshot(routes))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to marshal traefik rest snapshot", err)
	}
	if err := m.putRestConfig(ctx, gateway.RestApiUrl, body); err != nil {
		return err
	}
	if err := m.pruneCertificatesUnlocked(gateway, routes); err != nil {
		return err
	}
	return nil
}

func (m *RouteManager) ListRouters(ctx context.Context, restApiUrl string) ([]routeport.TraefikRouter, error) {
	httpRouters, err := m.listRouters(ctx, restApiUrl, "http")
	if err != nil {
		return nil, err
	}
	tcpRouters, err := m.listRouters(ctx, restApiUrl, "tcp")
	if err != nil {
		return nil, err
	}
	return append(httpRouters, tcpRouters...), nil
}

func (m *RouteManager) listRouters(ctx context.Context, restApiUrl string, protocol string) ([]routeport.TraefikRouter, error) {
	base := strings.TrimRight(strings.TrimSpace(restApiUrl), "/")
	if base == "" {
		return nil, apperror.New(apperror.KindValidation, "gateway rest_api_url is required")
	}
	url := base + "/api/" + protocol + "/routers"
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
			TLSConfig:   normalizeJSON(router.TLS),
			TLS:         normalizeJSON(router.TLS) != "",
		})
	}
	return items, nil
}

func (m *RouteManager) ListServices(ctx context.Context, restApiUrl string) ([]routeport.TraefikService, error) {
	httpServices, err := m.listServices(ctx, restApiUrl, "http")
	if err != nil {
		return nil, err
	}
	tcpServices, err := m.listServices(ctx, restApiUrl, "tcp")
	if err != nil {
		return nil, err
	}
	return append(httpServices, tcpServices...), nil
}

func (m *RouteManager) listServices(ctx context.Context, restApiUrl string, protocol string) ([]routeport.TraefikService, error) {
	base := strings.TrimRight(strings.TrimSpace(restApiUrl), "/")
	if base == "" {
		return nil, apperror.New(apperror.KindValidation, "gateway rest_api_url is required")
	}
	url := base + "/api/" + protocol + "/services"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create traefik request: %w", err)
	}
	response, err := m.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request traefik services: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("traefik returned status %d", response.StatusCode)
	}
	var services []struct {
		Name         string `json:"name"`
		Provider     string `json:"provider"`
		Status       string `json:"status"`
		LoadBalancer *struct {
			Servers []struct {
				Url     string `json:"url"`
				Address string `json:"address"`
			} `json:"servers"`
		} `json:"loadBalancer"`
	}
	if err := json.NewDecoder(response.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("decode traefik services: %w", err)
	}
	items := make([]routeport.TraefikService, 0, len(services))
	for _, service := range services {
		servers := make([]string, 0)
		if service.LoadBalancer != nil {
			servers = make([]string, 0, len(service.LoadBalancer.Servers))
			for _, server := range service.LoadBalancer.Servers {
				value := server.Url
				if value == "" {
					value = server.Address
				}
				servers = append(servers, value)
			}
		}
		items = append(items, routeport.TraefikService{
			Name:     service.Name,
			Provider: service.Provider,
			Status:   service.Status,
			Protocol: protocol,
			Servers:  servers,
		})
	}
	return items, nil
}

func normalizeJSON(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var value any
	if err := json.Unmarshal(*raw, &value); err != nil || value == nil {
		return ""
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
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

func (m *RouteManager) writeCertificateUnlocked(gateway model.GatewayConfig, routeName string, certPEM string, certKey string) error {
	certDir := m.routeCertDir(gateway)
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

func (m *RouteManager) pruneCertificatesUnlocked(gateway model.GatewayConfig, routes []model.Route) error {
	certDir := m.routeCertDir(gateway)
	entries, err := os.ReadDir(certDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to inspect route certificate directory", err)
	}
	keep := make(map[string]struct{})
	for _, route := range routes {
		if !route.Enabled || !routeHasStoredCertificate(route) {
			continue
		}
		keep[route.Name+".pem"] = struct{}{}
		keep[route.Name+"-key.pem"] = struct{}{}
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".pem" {
			continue
		}
		if _, found := keep[entry.Name()]; found {
			continue
		}
		if err := os.Remove(filepath.Join(certDir, entry.Name())); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to prune route certificate", err)
		}
	}
	return nil
}

func (m *RouteManager) routeCertDir(gateway model.GatewayConfig) string {
	return filepath.Join(m.cfg.Workspace.Deployment, gateway.RuntimeServiceCode, "gateway", "certs")
}

// buildRestSnapshot assembles a full providers.rest HTTP and TCP config (full replace semantics).
func buildRestSnapshot(routes []model.Route) map[string]any {
	httpRouters := map[string]any{}
	httpServices := map[string]any{}
	tcpRouters := map[string]any{}
	tcpServices := map[string]any{}
	tlsCertificates := make([]map[string]string, 0)
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		if route.Protocol == "tcp" {
			if route.ListenPort == nil || route.TargetAddress == "" || route.TargetPort < 1 {
				continue
			}
			serviceName := sanitizeTraefikName(route.Name) + "-service"
			routerName := sanitizeTraefikName(route.Name) + "-route"
			tcpRouters[routerName] = map[string]any{
				"rule": "HostSNI(`*`)", "service": serviceName,
				"entryPoints": []string{"tcp" + strconv.Itoa(*route.ListenPort)},
			}
			tcpServices[serviceName] = map[string]any{
				"loadBalancer": map[string]any{
					"servers": []map[string]string{{"address": net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))}},
				},
			}
			continue
		}
		targetUrl := route.TargetUrl
		if route.TargetAddress != "" && route.TargetPort > 0 {
			targetUrl = "http://" + net.JoinHostPort(route.TargetAddress, strconv.Itoa(route.TargetPort))
		}
		if strings.TrimSpace(targetUrl) == "" {
			continue
		}
		if routeHasStoredCertificate(route) {
			tlsCertificates = append(tlsCertificates, map[string]string{
				"certFile": "/etc/traefik/certs/" + route.Name + ".pem",
				"keyFile":  "/etc/traefik/certs/" + route.Name + "-key.pem",
			})
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
				resolver := "letsencrypt"
				if route.AcmeChallenge == "dns" {
					resolver = "letsencrypt-dns"
				}
				router["tls"] = map[string]any{"certResolver": resolver}
			} else {
				router["tls"] = map[string]any{}
			}
		} else {
			router["entryPoints"] = []string{"web"}
		}
		httpRouters[routerName] = router
		httpServices[serviceName] = map[string]any{
			"loadBalancer": map[string]any{
				"servers": []map[string]string{{"url": targetUrl}},
			},
		}
	}
	// Empty maps clear the @rest namespace (unlike {} / {"http":{}}).
	return map[string]any{
		"http": map[string]any{
			"routers":  httpRouters,
			"services": httpServices,
		},
		"tcp": map[string]any{
			"routers":  tcpRouters,
			"services": tcpServices,
		},
		"tls": map[string]any{
			"certificates": tlsCertificates,
		},
	}
}

func routeHasStoredCertificate(route model.Route) bool {
	return route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil && strings.TrimSpace(*route.CertPEM) != "" && strings.TrimSpace(*route.CertKey) != ""
}

func sanitizeTraefikName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "route"
	}
	return name
}
