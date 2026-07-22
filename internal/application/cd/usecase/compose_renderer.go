package cdsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

// RenderInput is the dual-input render context: Version×Environment×Gateway.
type RenderInput struct {
	App            model.Application
	Version        model.Version
	Components     []model.VersionComponent
	Exposes        []model.VersionExpose
	Env            model.Environment
	Service        model.Service
	Gateway        *model.GatewayConfig // required for HTTP host / gateway dashboard when domains needed
	RuntimeConfig  map[string]string    // resolved placeholders for this deploy/preview
	PhysicalSvcDir string               // host path for logical mounts; required when mounts present
}

// RenderResult is compose YAML plus resolved logical mounts for deploy materialize.
type RenderResult struct {
	Compose        string
	ResolvedMounts []ResolvedMount
}

const defaultGatewayNetworkName = "traefik"

// gatewayNetworkKey is the compose network key for kind=gateway (provider creates the network).
const gatewayNetworkKey = "default"

// consumerPlatformNetworkKey is the compose network key standard apps use to join the gateway network (E1).
const consumerPlatformNetworkKey = "traefik"

// RenderCompose builds docker-compose.yml from Version + Environment.
// Preview and deploy share this function. Branch is driven by Application.kind only.
func (s Service) RenderCompose(ctx context.Context, input RenderInput) (string, error) {
	result, err := s.RenderComposeDetailed(ctx, input)
	if err != nil {
		return "", err
	}
	return result.Compose, nil
}

// RenderComposeDetailed returns compose YAML and resolved mounts for materialize.
func (s Service) RenderComposeDetailed(ctx context.Context, input RenderInput) (RenderResult, error) {
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
		return RenderResult{}, fmt.Errorf("unsupported application kind %q", kind)
	}
}

// renderStandardCompose materializes components and injects Traefik labels when Exposes exist.
// E1: with Exposes, inject platform network (external) and join exposed components.
func renderStandardCompose(input RenderInput) (RenderResult, error) {
	return renderComposeServices(input, false)
}

// renderGatewayCompose adds top-level network (D4) and uses the same mount/env path.
func renderGatewayCompose(input RenderInput) (RenderResult, error) {
	return renderComposeServices(input, true)
}

func renderComposeServices(input RenderInput, injectGatewayNetwork bool) (RenderResult, error) {
	if len(input.Components) == 0 {
		return RenderResult{}, fmt.Errorf("version %s has no components", input.Version.Id)
	}
	if err := validateVersionComponents(input.Components); err != nil {
		return RenderResult{}, err
	}
	if err := validateVersionExposes(input.Exposes, input.Components); err != nil {
		return RenderResult{}, err
	}

	versionEnvVars, err := parseEnvVars(input.Version.EnvJSON)
	if err != nil {
		return RenderResult{}, fmt.Errorf("version env_json: %w", err)
	}
	versionEnv := applyEnvPlaceholders(versionEnvVars, input.RuntimeConfig)

	appCode := strings.TrimSpace(input.App.Code)
	services := make(map[string]any, len(input.Components))
	var allResolved []ResolvedMount
	for _, component := range input.Components {
		service, resolved, err := renderVersionComponentService(component, versionEnv, appCode, input.PhysicalSvcDir, input.RuntimeConfig)
		if err != nil {
			return RenderResult{}, fmt.Errorf("component %s: %w", component.Name, err)
		}
		if injectGatewayNetwork {
			service["networks"] = []string{gatewayNetworkKey}
		}
		services[component.Name] = service
		allResolved = append(allResolved, resolved...)
	}

	// Business app ingress: Version.Expose → docker provider labels (loadbalancer → container port).
	if len(input.Exposes) > 0 {
		if err := injectExposeLabels(services, input); err != nil {
			return RenderResult{}, err
		}
	}
	// Gateway control-plane dashboard: kind=gateway × Environment.IngressPolicy → api@internal.
	// Independent of Expose; does not use loadbalancer.server.port.
	if injectGatewayNetwork {
		if err := injectGatewayDashboardLabels(services, input); err != nil {
			return RenderResult{}, err
		}
	}

	// E1: standard + Expose → join gateway network as external consumer.
	joinConsumerNetwork := !injectGatewayNetwork && len(input.Exposes) > 0
	if joinConsumerNetwork {
		injectConsumerPlatformNetwork(services, input.Exposes)
	}

	data := map[string]any{"services": services}
	if injectGatewayNetwork {
		data["networks"] = map[string]any{
			gatewayNetworkKey: map[string]any{
				"name":   defaultGatewayNetworkName,
				"driver": "bridge",
			},
		}
	} else if joinConsumerNetwork {
		data["networks"] = map[string]any{
			consumerPlatformNetworkKey: map[string]any{
				"name":     defaultGatewayNetworkName,
				"external": true,
			},
		}
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return RenderResult{}, fmt.Errorf("marshal compose: %w", err)
	}
	return RenderResult{Compose: string(out), ResolvedMounts: allResolved}, nil
}

// injectConsumerPlatformNetwork attaches the platform gateway network to every component that has an Expose.
// Keeps project "default" so depends_on / sibling services still share a project network.
func injectConsumerPlatformNetwork(services map[string]any, exposes []model.VersionExpose) {
	exposed := make(map[string]struct{}, len(exposes))
	for _, expose := range exposes {
		name := strings.TrimSpace(expose.ComponentName)
		if name != "" {
			exposed[name] = struct{}{}
		}
	}
	for name, raw := range services {
		if _, ok := exposed[name]; !ok {
			continue
		}
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ensureServiceJoinsNetworks(service, "default", consumerPlatformNetworkKey)
	}
}

// ensureServiceJoinsNetworks merges required network attachments into a service definition.
func ensureServiceJoinsNetworks(service map[string]any, required ...string) {
	raw, has := service["networks"]
	if !has || raw == nil {
		service["networks"] = append([]string(nil), required...)
		return
	}
	switch nets := raw.(type) {
	case []string:
		service["networks"] = mergeStringSet(nets, required...)
	case []any:
		existing := make([]string, 0, len(nets))
		for _, item := range nets {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				existing = append(existing, strings.TrimSpace(s))
			}
		}
		service["networks"] = mergeStringSet(existing, required...)
	case map[string]any:
		for _, name := range required {
			if _, ok := nets[name]; !ok {
				nets[name] = map[string]any{}
			}
		}
		service["networks"] = nets
	default:
		service["networks"] = append([]string(nil), required...)
	}
}

func mergeStringSet(existing []string, required ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(required))
	out := make([]string, 0, len(existing)+len(required))
	for _, name := range existing {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range required {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func injectExposeLabels(services map[string]any, input RenderInput) error {
	if err := validatePolicyForExposes(input.Env, input.Gateway, input.Exposes, input.App); err != nil {
		return err
	}
	host, err := deriveHost(input.Gateway, input.App.Code)
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

func validatePolicyForExposes(env model.Environment, gateway *model.GatewayConfig, exposes []model.VersionExpose, app model.Application) error {
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
		if _, err := deriveHost(gateway, app.Code); err != nil {
			return err
		}
		if err := validateHTTPPathConflicts(gateway, app.Code, exposes); err != nil {
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
			if _, err := deriveHost(gateway, app.Code); err != nil {
				return fmt.Errorf("gateway base_domain required for TCP TLS: %w", err)
			}
		}
	}
	return nil
}

func validateHTTPPathConflicts(gateway *model.GatewayConfig, appCode string, exposes []model.VersionExpose) error {
	host, err := deriveHost(gateway, appCode)
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
			return fmt.Errorf("duplicate http route host+path for version: %s%s (use distinct path_prefix, or publish host ports via component ports instead of multiple HTTP exposes)", host, path)
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

// deriveHost builds {app_code}.{gateway.base_domain}. Domain templates are not supported (E6).
func deriveHost(gateway *model.GatewayConfig, appCode string) (string, error) {
	if gateway == nil {
		return "", fmt.Errorf("gateway config required for host derivation")
	}
	baseDomain := strings.TrimSpace(gateway.BaseDomain)
	if baseDomain == "" {
		return "", fmt.Errorf("gateway base_domain is required for host derivation")
	}
	appCode = strings.TrimSpace(appCode)
	if appCode == "" {
		return "", fmt.Errorf("app code is required for host derivation")
	}
	host := strings.ToLower(appCode + "." + baseDomain)
	if !validDeploymentRouteDomain(host) {
		return "", fmt.Errorf("invalid derived host %q", host)
	}
	return host, nil
}

// injectGatewayDashboardLabels writes Traefik self-route labels for the dashboard/API.
// Pattern matches the bak probe: enable + Host(policy) + entrypoint + service=api@internal.
// Attaches to the first component only (one Host rule per gateway instance).
// Incomplete IngressPolicy → skip (ports-only deploy still works).
func injectGatewayDashboardLabels(services map[string]any, input RenderInput) error {
	entrypoint := strings.TrimSpace(input.Env.DefaultEntrypoint)
	if entrypoint == "" {
		return nil
	}
	host, err := deriveHost(input.Gateway, input.App.Code)
	if err != nil {
		return nil
	}
	if len(input.Components) == 0 {
		return nil
	}
	componentName := strings.TrimSpace(input.Components[0].Name)
	service, ok := services[componentName].(map[string]any)
	if !ok {
		return fmt.Errorf("gateway component %s not found in compose services", componentName)
	}
	routerName := sanitizeComposeName(fmt.Sprintf("%s-%s-%s-dashboard",
		input.App.Code, input.Env.Code, input.Service.InstanceKey))
	labels := []string{
		"traefik.enable=true",
		"traefik.http.routers." + routerName + ".rule=Host(`" + host + "`)",
		"traefik.http.routers." + routerName + ".entrypoints=" + entrypoint,
		"traefik.http.routers." + routerName + ".service=api@internal",
	}
	tlsMode := strings.ToLower(strings.TrimSpace(input.Env.TLSMode))
	if tlsMode == "letsencrypt" {
		labels = append(labels,
			"traefik.http.routers."+routerName+".tls=true",
			"traefik.http.routers."+routerName+".tls.certresolver=letsencrypt",
		)
	} else if tlsMode == "tls" {
		labels = append(labels, "traefik.http.routers."+routerName+".tls=true")
	}
	existing, _ := service["labels"].([]string)
	service["labels"] = append(existing, labels...)
	return nil
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
			derived, err := deriveHost(input.Gateway, input.App.Code)
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
				derived, err := deriveHost(input.Gateway, input.App.Code)
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
		if _, err := parseMountSpecs(component.MountsJSON); err != nil {
			return fmt.Errorf("component %s mounts_json: %w", name, err)
		}
		if _, err := parseEnvVars(component.EnvJSON); err != nil {
			return fmt.Errorf("component %s env_json: %w", name, err)
		}
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

func renderVersionComponentService(
	component model.VersionComponent,
	versionEnv map[string]string,
	appCode string,
	physicalServiceDir string,
	runtime map[string]string,
) (map[string]any, []ResolvedMount, error) {
	service := map[string]any{
		"image":          strings.TrimSpace(component.Image),
		"container_name": appCode + "_" + strings.TrimSpace(component.Name),
	}
	if command, err := parseStringSliceJSON(component.CommandJSON); err != nil {
		return nil, nil, err
	} else if len(command) > 0 {
		service["command"] = command
	}
	if args, err := parseStringSliceJSON(component.ArgsJSON); err != nil {
		return nil, nil, err
	} else if len(args) > 0 {
		if existing, ok := service["command"].([]string); ok {
			service["command"] = append(existing, args...)
		} else {
			service["command"] = args
		}
	}
	componentEnvVars, err := parseEnvVars(component.EnvJSON)
	if err != nil {
		return nil, nil, err
	}
	componentEnv := applyEnvPlaceholders(componentEnvVars, runtime)
	env := mergeEnv(versionEnv, componentEnv)
	if len(env) > 0 {
		service["environment"] = env
	}
	if ports, err := parseAnyJSON(component.PortsJSON); err != nil {
		return nil, nil, err
	} else if ports != nil {
		service["ports"] = ports
	}
	mounts, err := parseMountSpecs(component.MountsJSON)
	if err != nil {
		return nil, nil, err
	}
	resolved, err := resolveMountSpecs(mounts, physicalServiceDir)
	if err != nil {
		return nil, nil, err
	}
	if len(resolved) > 0 {
		volumes := make([]string, 0, len(resolved))
		for _, item := range resolved {
			volumes = append(volumes, item.Compose)
		}
		service["volumes"] = volumes
	}
	if networks, err := parseAnyJSON(component.NetworksJSON); err != nil {
		return nil, nil, err
	} else if networks != nil {
		service["networks"] = networks
	}
	if depends, err := parseStringSliceJSON(component.DependsOnJSON); err != nil {
		return nil, nil, err
	} else if len(depends) > 0 {
		service["depends_on"] = depends
	}
	if healthcheck, err := parseAnyJSON(component.HealthcheckJSON); err != nil {
		return nil, nil, err
	} else if healthcheck != nil {
		service["healthcheck"] = healthcheck
	}
	if resources, err := parseAnyJSON(component.ResourcesJSON); err != nil {
		return nil, nil, err
	} else if resources != nil {
		service["deploy"] = map[string]any{"resources": resources}
	}
	if component.PullPolicy != nil && strings.TrimSpace(*component.PullPolicy) != "" {
		service["pull_policy"] = strings.TrimSpace(*component.PullPolicy)
	}
	return service, resolved, nil
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
