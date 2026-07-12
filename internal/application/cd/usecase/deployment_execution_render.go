package cdsvc

import (
	"context"
	"errors"
	"strconv"
	"strings"

	templatex "gitee.com/leoninew/PomeloOrbit-go/internal/common/template"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

func (s Service) renderDeploymentCompose(ctx context.Context, app model.Application, compose model.ApplicationConfigFile) (string, error) {
	serviceConfigs, err := s.executionStore.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return "", err
	}
	var routes []model.ApplicationRoute
	if app.RouteManaged {
		routes, err = s.executionStore.Routes(ctx, app.Id)
		if err != nil {
			return "", err
		}
	}
	_, content, err := s.renderDeploymentConfigFile(ctx, app, compose.Path, compose.Content, serviceConfigs, routes)
	return content, err
}

func (s Service) renderDeploymentConfigFile(ctx context.Context, app model.Application, path string, content string, serviceConfigs []model.ApplicationServiceConfig, routes []model.ApplicationRoute) (string, string, error) {
	if strippedPath, ok := strings.CutSuffix(path, ".liquid"); ok {
		path = strippedPath
		rendered, err := s.renderDeploymentTemplate(ctx, content, app.Code)
		if err != nil {
			return "", "", err
		}
		content = rendered
	}
	if path == "docker-compose.yml" {
		content = applyDeploymentServiceConfigs(content, serviceConfigs)
		if app.RouteManaged {
			rendered, err := injectDeploymentRouteLabels(content, routes, s.cfg.Cert.LetsEncrypt.Enabled)
			if err != nil {
				return "", "", err
			}
			content = rendered
		}
	}
	return path, content, nil
}

func (s Service) renderDeploymentTemplate(ctx context.Context, content string, appCode string) (string, error) {
	physicalDir, err := s.workspace.PhysicalDir(ctx)
	if err != nil {
		return "", err
	}
	physicalAppDir, err := s.workspace.PhysicalAppDir(ctx, appCode)
	if err != nil {
		return "", err
	}
	context := map[string]any{
		"app": map[string]any{
			"code":             appCode,
			"physical_dir":     physicalDir,
			"physical_app_dir": physicalAppDir,
		},
		"config": map[string]any{"domain_suffix": s.cfg.Traefik.DomainSuffix},
		"cert": map[string]any{"letsencrypt": map[string]any{
			"enabled":      s.cfg.Cert.LetsEncrypt.Enabled,
			"email":        s.cfg.Cert.LetsEncrypt.Email,
			"challenge":    s.cfg.Cert.LetsEncrypt.Challenge,
			"dns_provider": s.cfg.Cert.LetsEncrypt.DNSProvider,
		}},
	}
	return templatex.Render(content, context)
}

func applyDeploymentServiceConfigs(compose string, configs []model.ApplicationServiceConfig) string {
	if len(configs) == 0 {
		return compose
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return compose
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return compose
	}
	for _, config := range configs {
		service, ok := services[config.ServiceName].(map[string]any)
		if !ok || config.Image == nil || strings.TrimSpace(*config.Image) == "" {
			continue
		}
		service["image"] = strings.TrimSpace(*config.Image)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return compose
	}
	return string(out)
}

func injectDeploymentRouteLabels(compose string, routes []model.ApplicationRoute, letsEncrypt bool) (string, error) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return "", err
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return "", errors.New("docker-compose services must be a mapping")
	}
	for _, raw := range services {
		if service, ok := raw.(map[string]any); ok {
			delete(service, "labels")
		}
	}
	for serviceName, group := range groupDeploymentRoutes(routes) {
		service, ok := services[serviceName].(map[string]any)
		if !ok {
			return "", errors.New("service " + serviceName + " not found in docker-compose.yml")
		}
		labels, err := deploymentRouteLabels(serviceName, group, letsEncrypt)
		if err != nil {
			return "", err
		}
		service["labels"] = labels
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func groupDeploymentRoutes(routes []model.ApplicationRoute) map[string][]model.ApplicationRoute {
	groups := make(map[string][]model.ApplicationRoute)
	for _, route := range routes {
		groups[route.ServiceName] = append(groups[route.ServiceName], route)
	}
	return groups
}

func deploymentRouteLabels(serviceName string, routes []model.ApplicationRoute, letsEncrypt bool) ([]string, error) {
	if len(routes) == 0 {
		return nil, errors.New("application route group is empty")
	}
	port := routes[0].Port
	hosts := make([]string, 0, len(routes))
	seen := map[string]struct{}{}
	for _, route := range routes {
		if route.Port != port {
			return nil, errors.New("service " + serviceName + " has routes with different ports")
		}
		domain := strings.ToLower(strings.TrimSpace(route.Domain))
		if !validDeploymentRouteDomain(domain) {
			return nil, errors.New("service " + serviceName + " has invalid route domain")
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		hosts = append(hosts, "Host(`"+domain+"`)")
	}
	labels := []string{
		"traefik.enable=true",
		"traefik.http.routers." + serviceName + ".rule=" + strings.Join(hosts, " || "),
		"traefik.http.services." + serviceName + ".loadbalancer.server.port=" + strconv.Itoa(port),
	}
	if letsEncrypt {
		labels = append(labels, "traefik.http.routers."+serviceName+".entrypoints=websecure", "traefik.http.routers."+serviceName+".tls=true", "traefik.http.routers."+serviceName+".tls.certresolver=letsencrypt")
	} else {
		labels = append(labels, "traefik.http.routers."+serviceName+".entrypoints=web")
	}
	return labels, nil
}

func validDeploymentRouteDomain(domain string) bool {
	return domain != "" && len(domain) <= 253 && !strings.ContainsAny(domain, " `\t\r\n")
}
