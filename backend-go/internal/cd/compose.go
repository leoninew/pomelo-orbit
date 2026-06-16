package cd

import (
	"fmt"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/repository"
	"backend/internal/templatex"
	"gopkg.in/yaml.v3"
)

func renderTemplate(content string, appCode string, cfg config.Config) (string, error) {
	context := map[string]any{
		"app": map[string]any{
			"code":             appCode,
			"physical_dir":     cfg.DataRoot(),
			"physical_app_dir": filepath.Join(cfg.DataRoot(), "cd", appCode),
		},
		"config": map[string]any{
			"domain_suffix": cfg.Traefik.DomainSuffix,
		},
		"cert": map[string]any{
			"letsencrypt": map[string]any{
				"enabled":      cfg.Cert.LetsEncrypt.Enabled,
				"email":        cfg.Cert.LetsEncrypt.Email,
				"challenge":    cfg.Cert.LetsEncrypt.Challenge,
				"dns_provider": cfg.Cert.LetsEncrypt.DNSProvider,
			},
		},
	}
	return templatex.Render(content, context)
}

func applyServiceConfigs(compose string, configs []repository.ApplicationServiceConfig) string {
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

func injectRouteLabels(compose string, routes []repository.ApplicationRoute, letsEncrypt bool) (string, error) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return "", err
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("docker-compose services must be a mapping")
	}
	for _, raw := range services {
		if service, ok := raw.(map[string]any); ok {
			delete(service, "labels")
		}
	}
	for _, route := range routes {
		service, ok := services[route.ServiceName].(map[string]any)
		if !ok {
			return "", fmt.Errorf("service %q not found in docker-compose.yml", route.ServiceName)
		}
		labels := []string{
			"traefik.enable=true",
			fmt.Sprintf("traefik.http.routers.%s.rule=Host(`%s`)", route.ServiceName, route.Domain),
			fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port=%d", route.ServiceName, route.Port),
		}
		if letsEncrypt {
			labels = append(labels,
				fmt.Sprintf("traefik.http.routers.%s.entrypoints=websecure", route.ServiceName),
				fmt.Sprintf("traefik.http.routers.%s.tls=true", route.ServiceName),
				fmt.Sprintf("traefik.http.routers.%s.tls.certresolver=letsencrypt", route.ServiceName),
			)
		} else {
			labels = append(labels, fmt.Sprintf("traefik.http.routers.%s.entrypoints=web", route.ServiceName))
		}
		service["labels"] = labels
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
