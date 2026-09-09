package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var _ routeport.RouteConfigPublisher = (*RouteManager)(nil)
var _ routeport.TraefikRouterClient = (*RouteManager)(nil)

const restApiReadyPollInterval = 500 * time.Millisecond

type RouteManager struct {
	targetResolver environmentport.TargetResolver
	runtime        deploymentport.Runtime
	mu             sync.Mutex
}

func NewRouteManager(targetResolver environmentport.TargetResolver, runtime deploymentport.Runtime) *RouteManager {
	return &RouteManager{targetResolver: targetResolver, runtime: runtime}
}

func (m *RouteManager) WaitUntilReady(ctx context.Context, projectID string, gateway model.GatewayConfig, timeout time.Duration) error {
	base, err := traefikBaseURL(gateway.RestApiUrl)
	if err != nil {
		return err
	}
	if timeout <= 0 {
		return apperror.New(apperror.KindValidation, "gateway rest_ready_timeout_seconds is required")
	}
	target, err := m.resolveTarget(ctx, projectID)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		_, lastErr = m.runtime.QueryAtEnvironmentRoot(ctx, target, "curl", "-fsS", "--max-time", "5", base+"/api/overview")
		if lastErr == nil {
			return nil
		}
		if !time.Now().Before(deadline) {
			return apperror.Wrap(apperror.KindInternal, "Traefik REST API not ready within timeout", lastErr)
		}
		timer := time.NewTimer(restApiReadyPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return apperror.Wrap(apperror.KindInternal, "Traefik REST API readiness canceled", lastErr)
		case <-timer.C:
		}
	}
}

func (m *RouteManager) ApplySnapshot(ctx context.Context, projectID string, gateway model.GatewayConfig, routes []model.Route) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	base, err := traefikBaseURL(gateway.RestApiUrl)
	if err != nil {
		return err
	}
	target, err := m.resolveTarget(ctx, projectID)
	if err != nil {
		return err
	}
	serviceDir, err := m.runtime.ServiceDir(target, gateway.RuntimeServiceCode)
	if err != nil {
		return err
	}
	certDir := path.Join(strings.ReplaceAll(serviceDir, "\\", "/"), "gateway", "certs")
	certFiles := make([]deploymentport.WorkspaceFile, 0)
	for _, route := range routes {
		if !routeHasStoredCertificate(route) {
			continue
		}
		certFiles = append(certFiles,
			deploymentport.WorkspaceFile{Path: path.Join(certDir, route.Name+".pem"), Content: []byte(*route.CertPEM), Mode: 0o600},
			deploymentport.WorkspaceFile{Path: path.Join(certDir, route.Name+"-key.pem"), Content: []byte(*route.CertKey), Mode: 0o600},
		)
	}
	if err := m.runtime.SyncFiles(ctx, target, certDir, certFiles, ".pem"); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to sync remote route certificates", err)
	}
	body, err := json.Marshal(buildRestSnapshot(routes))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to marshal traefik rest snapshot", err)
	}
	stateDir := path.Join(strings.ReplaceAll(serviceDir, "\\", "/"), ".orbit")
	snapshotPath := path.Join(stateDir, "traefik-rest.json")
	if err := m.runtime.SyncFiles(ctx, target, stateDir, []deploymentport.WorkspaceFile{{Path: snapshotPath, Content: body, Mode: 0o600}}, ""); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to stage remote traefik snapshot", err)
	}
	output, err := m.runtime.QueryAtEnvironmentRoot(ctx, target, "curl", "-fsS", "--max-time", "15", "-X", "PUT", "-H", "Content-Type: application/json", "--data-binary", "@"+snapshotPath, base+"/api/providers/rest")
	if err != nil {
		return apperror.New(apperror.KindInternal, outputOrRemoteError("Failed to put traefik rest config", output, err))
	}
	return nil
}

func (m *RouteManager) ListRouters(ctx context.Context, projectID string, gateway model.GatewayConfig) ([]routeport.TraefikRouter, error) {
	httpRouters, err := m.listRouters(ctx, projectID, gateway, "http")
	if err != nil {
		return nil, err
	}
	tcpRouters, err := m.listRouters(ctx, projectID, gateway, "tcp")
	if err != nil {
		return nil, err
	}
	return append(httpRouters, tcpRouters...), nil
}

func (m *RouteManager) listRouters(ctx context.Context, projectID string, gateway model.GatewayConfig, protocol string) ([]routeport.TraefikRouter, error) {
	body, err := m.get(ctx, projectID, gateway, "/api/"+protocol+"/routers")
	if err != nil {
		return nil, err
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
	if err := json.Unmarshal([]byte(body), &routers); err != nil {
		return nil, fmt.Errorf("decode traefik routers: %w", err)
	}
	items := make([]routeport.TraefikRouter, 0, len(routers))
	for _, router := range routers {
		tlsConfig := normalizeJSON(router.TLS)
		items = append(items, routeport.TraefikRouter{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: append([]string(nil), router.Entrypoints...), TLS: tlsConfig != "", TLSConfig: tlsConfig})
	}
	return items, nil
}

func (m *RouteManager) ListServices(ctx context.Context, projectID string, gateway model.GatewayConfig) ([]routeport.TraefikService, error) {
	httpServices, err := m.listServices(ctx, projectID, gateway, "http")
	if err != nil {
		return nil, err
	}
	tcpServices, err := m.listServices(ctx, projectID, gateway, "tcp")
	if err != nil {
		return nil, err
	}
	return append(httpServices, tcpServices...), nil
}

func (m *RouteManager) listServices(ctx context.Context, projectID string, gateway model.GatewayConfig, protocol string) ([]routeport.TraefikService, error) {
	body, err := m.get(ctx, projectID, gateway, "/api/"+protocol+"/services")
	if err != nil {
		return nil, err
	}
	var services []struct {
		Name         string `json:"name"`
		Provider     string `json:"provider"`
		Status       string `json:"status"`
		LoadBalancer *struct {
			Servers []struct {
				URL     string `json:"url"`
				Address string `json:"address"`
			} `json:"servers"`
		} `json:"loadBalancer"`
	}
	if err := json.Unmarshal([]byte(body), &services); err != nil {
		return nil, fmt.Errorf("decode traefik services: %w", err)
	}
	items := make([]routeport.TraefikService, 0, len(services))
	for _, service := range services {
		servers := make([]string, 0)
		if service.LoadBalancer != nil {
			for _, server := range service.LoadBalancer.Servers {
				value := server.URL
				if value == "" {
					value = server.Address
				}
				servers = append(servers, value)
			}
		}
		items = append(items, routeport.TraefikService{Name: service.Name, Provider: service.Provider, Status: service.Status, Protocol: protocol, Servers: servers})
	}
	return items, nil
}

func (m *RouteManager) get(ctx context.Context, projectID string, gateway model.GatewayConfig, endpoint string) (string, error) {
	base, err := traefikBaseURL(gateway.RestApiUrl)
	if err != nil {
		return "", err
	}
	target, err := m.resolveTarget(ctx, projectID)
	if err != nil {
		return "", err
	}
	output, err := m.runtime.QueryAtEnvironmentRoot(ctx, target, "curl", "-fsS", "--max-time", "15", base+endpoint)
	if err != nil {
		return output, fmt.Errorf("request traefik API: %w", err)
	}
	return output, nil
}

func (m *RouteManager) resolveTarget(ctx context.Context, projectID string) (environmentport.Target, error) {
	if m == nil || m.targetResolver == nil || m.runtime == nil {
		return environmentport.Target{}, apperror.New(apperror.KindInternal, "remote Traefik client is not configured")
	}
	return m.targetResolver.ResolveProjectTarget(ctx, projectID)
}

func (m *RouteManager) IsConnectionError(err error) bool {
	return err != nil
}

func traefikBaseURL(value string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(value), "/")
	if base == "" {
		return "", apperror.New(apperror.KindValidation, "gateway rest_api_url is required")
	}
	return base, nil
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

func outputOrRemoteError(prefix string, output string, err error) string {
	output = strings.TrimSpace(output)
	if output != "" {
		return prefix + ": " + output
	}
	return prefix + ": " + err.Error()
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
