package cdsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

// RenderInput is the dual-input render context: Version×Environment.
type RenderInput struct {
	App        model.Application
	Version    model.Version
	Components []model.Component
	Exposes    []model.Expose
	Env        model.Environment
	Bindings   []model.EnvironmentBinding
	Service    model.Service
}

// RenderCompose builds docker-compose.yml from Version + Environment.
// Preview and deploy share this function.
func (s Service) RenderCompose(ctx context.Context, input RenderInput) (string, error) {
	_ = ctx
	if len(input.Components) == 0 {
		return "", fmt.Errorf("version %s has no components", input.Version.Id)
	}
	if err := validateComponents(input.Components); err != nil {
		return "", err
	}
	if err := validateExposes(input.Exposes, input.Components); err != nil {
		return "", err
	}

	versionEnv, err := parseStringMapJSON(input.Version.EnvJSON)
	if err != nil {
		return "", fmt.Errorf("version env_json: %w", err)
	}

	services := make(map[string]any, len(input.Components))
	for _, component := range input.Components {
		service, err := renderComponentService(component, versionEnv)
		if err != nil {
			return "", fmt.Errorf("component %s: %w", component.Name, err)
		}
		services[component.Name] = service
	}

	if input.Service.IsIngress && len(input.Exposes) > 0 {
		if err := injectExposeLabels(services, input); err != nil {
			return "", err
		}
	}

	data := map[string]any{"services": services}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal compose: %w", err)
	}
	return string(out), nil
}

func injectExposeLabels(services map[string]any, input RenderInput) error {
	bindingIndex := make(map[string]model.EnvironmentBinding, len(input.Bindings))
	for _, binding := range input.Bindings {
		bindingIndex[exposeKey(binding.ComponentName, binding.Protocol, binding.ContainerPort)] = binding
	}
	letsEncryptEnabled := false
	if sCfg := input; sCfg.App.Id != "" {
		// letsencrypt flag comes from service config at call site via labels builder
	}
	_ = letsEncryptEnabled

	for _, expose := range input.Exposes {
		binding, ok := bindingIndex[exposeKey(expose.ComponentName, expose.Protocol, expose.ContainerPort)]
		if !ok {
			return fmt.Errorf("missing environment binding for expose %s %s:%d", expose.ComponentName, expose.Protocol, expose.ContainerPort)
		}
		service, ok := services[expose.ComponentName].(map[string]any)
		if !ok {
			return fmt.Errorf("component %s not found in compose services", expose.ComponentName)
		}
		routerName := sanitizeComposeName(fmt.Sprintf("%s-%s-%s-%s-%s",
			input.App.Code, input.Env.Code, input.Service.InstanceKey, expose.ComponentName, expose.Protocol))
		labels, err := buildTraefikLabels(routerName, expose, binding, input)
		if err != nil {
			return err
		}
		existing, _ := service["labels"].([]string)
		service["labels"] = append(existing, labels...)
	}
	return nil
}

func buildTraefikLabels(routerName string, expose model.Expose, binding model.EnvironmentBinding, input RenderInput) ([]string, error) {
	labels := []string{"traefik.enable=true"}
	entrypoint := strings.TrimSpace(binding.Entrypoint)
	if entrypoint == "" {
		return nil, fmt.Errorf("binding entrypoint is required for %s", expose.ComponentName)
	}
	switch strings.ToLower(strings.TrimSpace(expose.Protocol)) {
	case "http":
		domains, err := parseStringSliceJSON(&binding.DomainsJSON)
		if err != nil {
			return nil, fmt.Errorf("binding domains_json: %w", err)
		}
		if len(domains) == 0 {
			return nil, fmt.Errorf("http binding for %s requires at least one domain", expose.ComponentName)
		}
		hosts := make([]string, 0, len(domains))
		for _, domain := range domains {
			domain = strings.ToLower(strings.TrimSpace(domain))
			if !validDeploymentRouteDomain(domain) {
				return nil, fmt.Errorf("invalid domain %q", domain)
			}
			hosts = append(hosts, "Host(`"+domain+"`)")
		}
		rule := strings.Join(hosts, " || ")
		if prefix := strings.TrimSpace(ptrString(expose.PathPrefix)); prefix != "" && prefix != "/" {
			rule = "(" + rule + ") && PathPrefix(`" + prefix + "`)"
		}
		labels = append(labels,
			"traefik.http.routers."+routerName+".rule="+rule,
			"traefik.http.routers."+routerName+".entrypoints="+entrypoint,
			"traefik.http.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.http.routers."+routerName+".service="+routerName,
		)
		tlsMode := strings.ToLower(strings.TrimSpace(binding.TLSMode))
		if tlsMode == "letsencrypt" {
			labels = append(labels,
				"traefik.http.routers."+routerName+".tls=true",
				"traefik.http.routers."+routerName+".tls.certresolver=letsencrypt",
			)
		}
	case "tcp":
		sni := "*"
		if binding.SNIHost != nil && strings.TrimSpace(*binding.SNIHost) != "" {
			sni = strings.TrimSpace(*binding.SNIHost)
		}
		labels = append(labels,
			"traefik.tcp.routers."+routerName+".rule=HostSNI(`"+sni+"`)",
			"traefik.tcp.routers."+routerName+".entrypoints="+entrypoint,
			"traefik.tcp.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.tcp.routers."+routerName+".service="+routerName,
		)
		if strings.ToLower(strings.TrimSpace(binding.TLSMode)) == "letsencrypt" || strings.ToLower(strings.TrimSpace(binding.TLSMode)) == "tls" {
			labels = append(labels, "traefik.tcp.routers."+routerName+".tls=true")
		}
	default:
		return nil, fmt.Errorf("unsupported expose protocol %s", expose.Protocol)
	}
	_ = input
	return labels, nil
}

func exposeKey(componentName string, protocol string, port int) string {
	return strings.ToLower(strings.TrimSpace(componentName)) + "|" + strings.ToLower(strings.TrimSpace(protocol)) + "|" + strconv.Itoa(port)
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func validateComponents(components []model.Component) error {
	names := make(map[string]struct{}, len(components))
	for _, component := range components {
		name := strings.TrimSpace(component.Name)
		if name == "" {
			return fmt.Errorf("component name is required")
		}
		if strings.TrimSpace(component.Image) == "" {
			return fmt.Errorf("component %s image is required", name)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("duplicate component name %s", name)
		}
		names[name] = struct{}{}
	}
	for _, component := range components {
		depends, err := parseStringSliceJSON(component.DependsOnJSON)
		if err != nil {
			return fmt.Errorf("component %s depends_on_json: %w", component.Name, err)
		}
		for _, dep := range depends {
			if _, ok := names[dep]; !ok {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, dep)
			}
		}
	}
	return nil
}

func validateExposes(exposes []model.Expose, components []model.Component) error {
	names := make(map[string]struct{}, len(components))
	for _, c := range components {
		names[strings.TrimSpace(c.Name)] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, expose := range exposes {
		name := strings.TrimSpace(expose.ComponentName)
		if name == "" {
			return fmt.Errorf("expose component_name is required")
		}
		if _, ok := names[name]; !ok {
			return fmt.Errorf("expose component %s not found in version components", name)
		}
		protocol := strings.ToLower(strings.TrimSpace(expose.Protocol))
		if protocol != "http" && protocol != "tcp" {
			return fmt.Errorf("expose protocol must be http or tcp")
		}
		if expose.ContainerPort < 1 || expose.ContainerPort > 65535 {
			return fmt.Errorf("expose container_port out of range")
		}
		key := exposeKey(name, protocol, expose.ContainerPort)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate expose %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func renderComponentService(component model.Component, versionEnv map[string]string) (map[string]any, error) {
	service := map[string]any{
		"image": strings.TrimSpace(component.Image),
	}
	if command, err := parseStringSliceJSON(component.CommandJSON); err != nil {
		return nil, err
	} else if len(command) > 0 {
		service["command"] = command
	}
	if args, err := parseStringSliceJSON(component.ArgsJSON); err != nil {
		return nil, err
	} else if len(args) > 0 {
		if existing, ok := service["command"].([]string); ok {
			service["command"] = append(existing, args...)
		} else {
			service["command"] = args
		}
	}
	componentEnv, err := parseStringMapJSON(component.EnvJSON)
	if err != nil {
		return nil, err
	}
	env := mergeEnv(versionEnv, componentEnv)
	if len(env) > 0 {
		service["environment"] = env
	}
	if ports, err := parseAnyJSON(component.PortsJSON); err != nil {
		return nil, err
	} else if ports != nil {
		service["ports"] = ports
	}
	if mounts, err := parseAnyJSON(component.MountsJSON); err != nil {
		return nil, err
	} else if mounts != nil {
		service["volumes"] = mounts
	}
	if networks, err := parseAnyJSON(component.NetworksJSON); err != nil {
		return nil, err
	} else if networks != nil {
		service["networks"] = networks
	}
	if depends, err := parseStringSliceJSON(component.DependsOnJSON); err != nil {
		return nil, err
	} else if len(depends) > 0 {
		service["depends_on"] = depends
	}
	if healthcheck, err := parseAnyJSON(component.HealthcheckJSON); err != nil {
		return nil, err
	} else if healthcheck != nil {
		service["healthcheck"] = healthcheck
	}
	if resources, err := parseAnyJSON(component.ResourcesJSON); err != nil {
		return nil, err
	} else if resources != nil {
		service["deploy"] = map[string]any{"resources": resources}
	}
	if component.PullPolicy != nil && strings.TrimSpace(*component.PullPolicy) != "" {
		service["pull_policy"] = strings.TrimSpace(*component.PullPolicy)
	}
	return service, nil
}

func mergeEnv(base map[string]string, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func parseStringMapJSON(raw *string) (map[string]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseStringSliceJSON(raw *string) ([]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseAnyJSON(raw *string) (any, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var out any
	if err := json.Unmarshal([]byte(*raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func validDeploymentRouteDomain(domain string) bool {
	return domain != "" && len(domain) <= 253 && !strings.ContainsAny(domain, " `\t\r\n")
}
