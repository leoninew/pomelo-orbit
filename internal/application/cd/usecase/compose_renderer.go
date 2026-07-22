package cdsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

// RenderInput is the dual-input render context: Version×Environment.
type RenderInput struct {
	App        model.Application
	Version    model.Version
	Components []model.VersionComponent
	Exposes    []model.VersionExpose
	Env        model.Environment
	Service    model.Service
}

var domainTemplateTokenPattern = regexp.MustCompile(`\{([a-z_]+)\}`)

// RenderCompose builds docker-compose.yml from Version + Environment.
// Preview and deploy share this function. Branch is driven by Application.kind only.
func (s Service) RenderCompose(ctx context.Context, input RenderInput) (string, error) {
	_ = ctx
	kind := strings.TrimSpace(input.App.Kind)
	if kind == "" {
		kind = status.ApplicationKindStandard
	}
	switch kind {
	case status.ApplicationKindStandard:
		return renderStandardCompose(input)
	case status.ApplicationKindGateway:
		return renderGatewayCompose(input)
	default:
		return "", fmt.Errorf("unsupported application kind %q", kind)
	}
}

// renderStandardCompose materializes components and injects Traefik labels when Exposes exist.
func renderStandardCompose(input RenderInput) (string, error) {
	return renderComposeServices(input)
}

// renderGatewayCompose keeps a kind-level branch hook for future gateway strategy (E1/E5).
// R3 reuses the same component materialization path as standard.
func renderGatewayCompose(input RenderInput) (string, error) {
	return renderComposeServices(input)
}

func renderComposeServices(input RenderInput) (string, error) {
	if len(input.Components) == 0 {
		return "", fmt.Errorf("version %s has no components", input.Version.Id)
	}
	if err := validateVersionComponents(input.Components); err != nil {
		return "", err
	}
	if err := validateVersionExposes(input.Exposes, input.Components); err != nil {
		return "", err
	}

	versionEnv, err := parseStringMapJSON(input.Version.EnvJSON)
	if err != nil {
		return "", fmt.Errorf("version env_json: %w", err)
	}

	appCode := strings.TrimSpace(input.App.Code)
	services := make(map[string]any, len(input.Components))
	for _, component := range input.Components {
		service, err := renderVersionComponentService(component, versionEnv, appCode)
		if err != nil {
			return "", fmt.Errorf("component %s: %w", component.Name, err)
		}
		services[component.Name] = service
	}

	// Labels are driven by Expose presence, not Service.is_ingress / attach_ingress.
	if len(input.Exposes) > 0 {
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
	if err := validatePolicyForExposes(input.Env, input.Exposes, input.App); err != nil {
		return err
	}
	host, err := deriveHost(input.Env, input.App.Code)
	if err != nil {
		// host may be optional when only tcp with tls_mode=none and no base_domain needed
		host = ""
	}
	for _, expose := range input.Exposes {
		service, ok := services[expose.ComponentName].(map[string]any)
		if !ok {
			return fmt.Errorf("component %s not found in compose services", expose.ComponentName)
		}
		routerName := sanitizeComposeName(fmt.Sprintf("%s-%s-%s-%s-%s",
			input.App.Code, input.Env.Code, input.Service.InstanceKey, expose.ComponentName, expose.Protocol))
		labels, err := buildTraefikLabels(routerName, expose, input, host)
		if err != nil {
			return err
		}
		existing, _ := service["labels"].([]string)
		service["labels"] = append(existing, labels...)
	}
	return nil
}

func validatePolicyForExposes(env model.Environment, exposes []model.VersionExpose, app model.Application) error {
	hasHTTP := false
	hasTCP := false
	for _, expose := range exposes {
		switch strings.ToLower(strings.TrimSpace(expose.Protocol)) {
		case "http":
			hasHTTP = true
		case "tcp":
			hasTCP = true
		}
	}
	entrypoint := strings.TrimSpace(env.DefaultEntrypoint)
	if hasHTTP || hasTCP {
		if entrypoint == "" {
			return fmt.Errorf("environment ingress policy incomplete: default_entrypoint is required")
		}
	}
	if hasHTTP {
		if strings.TrimSpace(env.BaseDomain) == "" && strings.TrimSpace(ptrString(env.DomainTemplate)) == "" {
			return fmt.Errorf("environment ingress policy incomplete: base_domain required for HTTP expose")
		}
		if _, err := deriveHost(env, app.Code); err != nil {
			return err
		}
		if err := validateHTTPPathConflicts(env, app.Code, exposes); err != nil {
			return err
		}
	}
	if hasTCP {
		tcpEP := strings.TrimSpace(ptrString(env.TCPEntrypoint))
		if tcpEP == "" && entrypoint == "" {
			return fmt.Errorf("environment ingress policy incomplete: entrypoint required for TCP expose")
		}
		tlsMode := strings.ToLower(strings.TrimSpace(env.TLSMode))
		if tlsMode == "letsencrypt" || tlsMode == "tls" {
			if _, err := deriveHost(env, app.Code); err != nil {
				return fmt.Errorf("environment ingress policy incomplete: base_domain required for TCP TLS: %w", err)
			}
		}
	}
	return nil
}

func validateHTTPPathConflicts(env model.Environment, appCode string, exposes []model.VersionExpose) error {
	host, err := deriveHost(env, appCode)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, expose := range exposes {
		if strings.ToLower(strings.TrimSpace(expose.Protocol)) != "http" {
			continue
		}
		path := normalizeHTTPPath(ptrString(expose.PathPrefix))
		key := host + "|" + path
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate http route host+path for version: %s%s", host, path)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeHTTPPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func deriveHost(env model.Environment, appCode string) (string, error) {
	template := strings.TrimSpace(ptrString(env.DomainTemplate))
	if template == "" {
		template = "{app_code}.{base_domain}"
	}
	baseDomain := strings.TrimSpace(env.BaseDomain)
	replacements := map[string]string{
		"app_code":    strings.TrimSpace(appCode),
		"env_code":    strings.TrimSpace(env.Code),
		"base_domain": baseDomain,
	}
	var unknown []string
	expanded := domainTemplateTokenPattern.ReplaceAllStringFunc(template, func(token string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		value, ok := replacements[name]
		if !ok {
			unknown = append(unknown, name)
			return token
		}
		return value
	})
	if len(unknown) > 0 {
		return "", fmt.Errorf("domain_template contains unknown token: %s", unknown[0])
	}
	if strings.Contains(expanded, "{") {
		return "", fmt.Errorf("domain_template incomplete after expansion")
	}
	host := strings.ToLower(strings.TrimSpace(expanded))
	if baseDomain == "" && strings.Contains(template, "{base_domain}") {
		return "", fmt.Errorf("environment ingress policy incomplete: base_domain required for host derivation")
	}
	if !validDeploymentRouteDomain(host) {
		return "", fmt.Errorf("invalid derived host %q", host)
	}
	return host, nil
}

func buildTraefikLabels(routerName string, expose model.VersionExpose, input RenderInput, host string) ([]string, error) {
	labels := []string{"traefik.enable=true"}
	entrypoint := strings.TrimSpace(input.Env.DefaultEntrypoint)
	tlsMode := strings.ToLower(strings.TrimSpace(input.Env.TLSMode))
	if tlsMode == "" {
		tlsMode = "none"
	}

	switch strings.ToLower(strings.TrimSpace(expose.Protocol)) {
	case "http":
		if entrypoint == "" {
			return nil, fmt.Errorf("environment default_entrypoint is required for %s", expose.ComponentName)
		}
		if host == "" {
			derived, err := deriveHost(input.Env, input.App.Code)
			if err != nil {
				return nil, err
			}
			host = derived
		}
		rule := "Host(`" + host + "`)"
		path := strings.TrimSpace(ptrString(expose.PathPrefix))
		if path != "" && path != "/" {
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			rule = rule + " && PathPrefix(`" + path + "`)"
		}
		labels = append(labels,
			"traefik.http.routers."+routerName+".rule="+rule,
			"traefik.http.routers."+routerName+".entrypoints="+entrypoint,
			"traefik.http.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.http.routers."+routerName+".service="+routerName,
		)
		if tlsMode == "letsencrypt" {
			labels = append(labels,
				"traefik.http.routers."+routerName+".tls=true",
				"traefik.http.routers."+routerName+".tls.certresolver=letsencrypt",
			)
		} else if tlsMode == "tls" {
			labels = append(labels, "traefik.http.routers."+routerName+".tls=true")
		}
	case "tcp":
		tcpEntrypoint := strings.TrimSpace(ptrString(input.Env.TCPEntrypoint))
		if tcpEntrypoint == "" {
			tcpEntrypoint = entrypoint
		}
		if tcpEntrypoint == "" {
			return nil, fmt.Errorf("environment entrypoint is required for TCP %s", expose.ComponentName)
		}
		sni := "*"
		if tlsMode == "letsencrypt" || tlsMode == "tls" {
			if host == "" {
				derived, err := deriveHost(input.Env, input.App.Code)
				if err != nil {
					return nil, err
				}
				host = derived
			}
			sni = host
		}
		labels = append(labels,
			"traefik.tcp.routers."+routerName+".rule=HostSNI(`"+sni+"`)",
			"traefik.tcp.routers."+routerName+".entrypoints="+tcpEntrypoint,
			"traefik.tcp.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.tcp.routers."+routerName+".service="+routerName,
		)
		if tlsMode == "letsencrypt" || tlsMode == "tls" {
			labels = append(labels, "traefik.tcp.routers."+routerName+".tls=true")
		}
	default:
		return nil, fmt.Errorf("unsupported expose protocol %s", expose.Protocol)
	}
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

func validateVersionComponents(components []model.VersionComponent) error {
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

func validateVersionExposes(exposes []model.VersionExpose, components []model.VersionComponent) error {
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

func renderVersionComponentService(component model.VersionComponent, versionEnv map[string]string, appCode string) (map[string]any, error) {
	service := map[string]any{
		"image":          strings.TrimSpace(component.Image),
		"container_name": appCode + "_" + strings.TrimSpace(component.Name),
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
