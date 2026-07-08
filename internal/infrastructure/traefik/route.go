package traefik

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

type RouteManager struct {
	cfg config.Config
}

func NewRouteManager(cfg config.Config) RouteManager {
	return RouteManager{cfg: cfg}
}

func (m RouteManager) Sync(ctx context.Context, route model.Route) error {
	if route.HTTPSEnabled && route.CertPEM != nil && route.CertKey != nil {
		if err := m.WriteCertificate(ctx, route.Name, *route.CertPEM, *route.CertKey); err != nil {
			return err
		}
	}
	if route.Enabled {
		return m.deployRouteFile(ctx, route)
	}
	return m.Revoke(ctx, route.Name)
}

func (m RouteManager) Revoke(ctx context.Context, routeName string) error {
	path := filepath.Join(m.routeConfigDir(), routeName+".yml")
	if err := removeIfExists(path); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to revoke route config", err)
	}
	return m.reload(ctx)
}

func (m RouteManager) WriteCertificate(ctx context.Context, routeName string, certPEM string, certKey string) error {
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

func (m RouteManager) RevokeCertificate(ctx context.Context, routeName string) error {
	for _, suffix := range []string{".pem", "-key.pem"} {
		path := filepath.Join(m.routeCertDir(), routeName+suffix)
		if err := removeIfExists(path); err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to revoke route certificate", err)
		}
	}
	return m.reload(ctx)
}

func (m RouteManager) ListRouters(ctx context.Context) ([]model.TraefikRouter, error) {
	url := strings.TrimRight(strings.TrimSpace(m.cfg.Traefik.APIURL), "/") + "/api/http/routers"
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
	items := make([]model.TraefikRouter, 0, len(routers))
	for _, router := range routers {
		items = append(items, model.TraefikRouter{Name: router.Name, Provider: router.Provider, Status: router.Status, Rule: router.Rule, Service: router.Service, Entrypoints: router.Entrypoints, TLS: router.TLS != nil && string(*router.TLS) != "null"})
	}
	return items, nil
}

func (m RouteManager) IsConnectionError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func (m RouteManager) deployRouteFile(ctx context.Context, route model.Route) error {
	configDir := m.routeConfigDir()
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to create route config directory", err)
	}
	data, err := yaml.Marshal(routeConfig(route))
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to marshal route config", err)
	}
	path := filepath.Join(configDir, route.Name+".yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to write route config", err)
	}
	return m.reload(ctx)
}

func (m RouteManager) routeConfigDir() string {
	if strings.TrimSpace(m.cfg.Traefik.DynamicRouteDir) == "" {
		return filepath.Join(m.cfg.DataRoot(), "cd", "traefik", "data", "dynamic")
	}
	return cleanConfigPath(m.cfg.OrbitRoot(), m.cfg.Traefik.DynamicRouteDir)
}

func (m RouteManager) routeCertDir() string {
	if strings.TrimSpace(m.cfg.Traefik.CertDir) == "" {
		return filepath.Join(m.cfg.DataRoot(), "cd", "traefik", "data", "certs")
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

func (m RouteManager) reload(ctx context.Context) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	containerName := strings.TrimSpace(m.cfg.Traefik.ContainerName)
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

func routeConfig(route model.Route) map[string]any {
	serviceName := route.Name + "-service"
	routerName := route.Name + "-route"
	rule := "Host(`" + route.Domain + "`)"
	if route.PathPrefix != "/" {
		rule += " && PathPrefix(`" + route.PathPrefix + "`)"
	}
	router := map[string]any{"rule": rule, "service": serviceName}
	if route.HTTPSEnabled {
		router["entryPoints"] = []string{"websecure"}
		if route.CertType == "letsencrypt" {
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
	if route.HTTPSEnabled && route.CertType != "letsencrypt" && route.CertPEM != nil {
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
