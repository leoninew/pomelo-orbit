package deploymentsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gopkg.in/yaml.v3"
)

// RenderInput is the render context for one application version and gateway policy.
type RenderInput struct {
	App            model.Application
	Version        model.Version
	Components     []model.VersionComponent
	Exposes        []model.VersionExpose
	Service        model.Service
	Gateway        *model.GatewayConfig // required for public HTTP host / gateway dashboard when domains needed
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

// consumerPlatformNetworkKey is the compose network key standard apps use to join the gateway network.
const consumerPlatformNetworkKey = "traefik"

// RenderCompose builds docker-compose.yml from a version and runtime binding.
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
	switch input.App.Kind {
	case status.ApplicationKindStandard:
		return renderStandardCompose(input)
	case status.ApplicationKindGateway:
		return renderGatewayCompose(input)
	default:
		return RenderResult{}, fmt.Errorf("unsupported application kind %q", input.App.Kind)
	}
}

func renderStandardCompose(input RenderInput) (RenderResult, error) {
	return renderComposeServices(input, false)
}

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

	services := make(map[string]any, len(input.Components))
	var allResolved []ResolvedMount
	composeVolumes := map[string]any{}
	for _, component := range input.Components {
		service, resolved, err := renderVersionComponentService(component, input.App.Code, input.PhysicalSvcDir, input.RuntimeConfig)
		if err != nil {
			return RenderResult{}, fmt.Errorf("component %s: %w", component.Name, err)
		}
		if injectGatewayNetwork {
			service["networks"] = []string{gatewayNetworkKey}
		}
		services[component.Name] = service
		allResolved = append(allResolved, resolved...)
		for _, mount := range resolved {
			if mount.NamedVolumeName != "" {
				composeVolumes[mount.NamedVolumeName] = map[string]any{}
			}
		}
	}

	if len(input.Exposes) > 0 {
		if err := injectExposeOutlets(services, input); err != nil {
			return RenderResult{}, err
		}
	}
	// Gateway control-plane dashboard: kind=gateway × ingress policy → api@internal.
	if injectGatewayNetwork {
		if err := injectGatewayDashboardLabels(services, input); err != nil {
			return RenderResult{}, err
		}
	}

	// Standard apps always join platform network for cluster DNS (with or without expose).
	if !injectGatewayNetwork {
		if err := injectAllComponentsPlatformNetwork(services, input.App.Code); err != nil {
			return RenderResult{}, err
		}
	}
	data := map[string]any{"services": services}
	if len(composeVolumes) > 0 {
		data["volumes"] = composeVolumes
	}
	if injectGatewayNetwork {
		data["networks"] = map[string]any{
			gatewayNetworkKey: map[string]any{
				"name":   defaultGatewayNetworkName,
				"driver": "bridge",
			},
		}
	} else {
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

// injectAllComponentsPlatformNetwork joins every standard component to traefik with runtime alias.
func injectAllComponentsPlatformNetwork(services map[string]any, appCode string) error {
	for name, raw := range services {
		service, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ensureServiceJoinsNetworksWithAlias(service, runtimeName(appCode, name))
	}
	return nil
}

func ensureServiceJoinsNetworksWithAlias(service map[string]any, alias string) {
	raw, has := service["networks"]
	nets := map[string]any{
		"default": map[string]any{},
		consumerPlatformNetworkKey: map[string]any{
			"aliases": []string{alias},
		},
	}
	if !has || raw == nil {
		service["networks"] = nets
		return
	}
	switch existing := raw.(type) {
	case map[string]any:
		if _, ok := existing["default"]; !ok {
			existing["default"] = map[string]any{}
		}
		traefikNet, _ := existing[consumerPlatformNetworkKey].(map[string]any)
		if traefikNet == nil {
			traefikNet = map[string]any{}
		}
		traefikNet["aliases"] = []string{alias}
		existing[consumerPlatformNetworkKey] = traefikNet
		service["networks"] = existing
	default:
		service["networks"] = nets
	}
}

func injectExposeOutlets(services map[string]any, input RenderInput) error {
	if err := validatePolicyForExposes(input.Gateway, input.Exposes, input.App); err != nil {
		return err
	}
	if err := validateLocalListenConflicts(input.Exposes); err != nil {
		return err
	}
	host := ""
	if input.Gateway != nil {
		if h, err := deriveHost(input.Gateway, input.App.Code); err == nil {
			host = h
		}
	}
	for _, expose := range input.Exposes {
		service, ok := services[expose.ComponentName].(map[string]any)
		if !ok {
			return fmt.Errorf("component %s not found in compose services", expose.ComponentName)
		}
		switch expose.Access {
		case exposeAccessLocal:
			listen := effectiveListen(expose)
			portMapping := fmt.Sprintf("127.0.0.1:%d:%d", listen, expose.ContainerPort)
			existing, _ := service["ports"].([]any)
			if existing == nil {
				if ports, ok := service["ports"].([]string); ok {
					for _, p := range ports {
						existing = append(existing, p)
					}
				}
			}
			service["ports"] = append(existing, portMapping)
		case exposeAccessPublic:
			routerName := fmt.Sprintf("%s-%s-%s-%s",
				input.App.Code, input.Service.InstanceKey, expose.ComponentName, expose.Protocol)
			labels, err := buildTraefikLabels(routerName, expose, input, host)
			if err != nil {
				return err
			}
			existing, _ := service["labels"].([]string)
			service["labels"] = append(existing, labels...)
		default:
			return fmt.Errorf("unsupported expose access %q", expose.Access)
		}
	}
	return nil
}

func validateLocalListenConflicts(exposes []model.VersionExpose) error {
	seen := map[int]string{}
	for _, expose := range exposes {
		if expose.Access != exposeAccessLocal {
			continue
		}
		listen := effectiveListen(expose)
		key := listen
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("duplicate local listen port %d (%s and %s)", listen, prev, expose.ComponentName)
		}
		seen[key] = expose.ComponentName
	}
	return nil
}

func validatePolicyForExposes(gateway *model.GatewayConfig, exposes []model.VersionExpose, app model.Application) error {
	hasPublicHTTP := false
	hasPublicTCP := false
	hasPublicTCPWithTLS := false
	for _, expose := range exposes {
		if expose.Access != exposeAccessPublic {
			continue
		}
		switch expose.Protocol {
		case "http":
			hasPublicHTTP = true
		case "tcp":
			hasPublicTCP = true
		}
	}
	if !hasPublicHTTP && !hasPublicTCP {
		return nil
	}
	if gateway == nil {
		return fmt.Errorf("gateway config required for public expose")
	}
	tlsMode := gateway.TLSMode
	if tlsMode != "none" && tlsMode != "letsencrypt" && tlsMode != "tls" {
		return fmt.Errorf("gateway tls_mode must be none, letsencrypt or tls")
	}
	if hasPublicTCP && (tlsMode == "letsencrypt" || tlsMode == "tls") {
		hasPublicTCPWithTLS = true
	}
	if hasPublicHTTP {
		entrypoint := gateway.DefaultEntrypoint
		if entrypoint == "" {
			return fmt.Errorf("gateway ingress policy incomplete: default_entrypoint is required")
		}
		if !validGatewayEntrypoint(entrypoint) {
			return fmt.Errorf("gateway default_entrypoint must be web or websecure")
		}
		if _, err := deriveHost(gateway, app.Code); err != nil {
			return err
		}
		if err := validateHTTPPathConflicts(gateway, app.Code, exposes); err != nil {
			return err
		}
	}
	if hasPublicTCPWithTLS {
		if _, err := deriveHost(gateway, app.Code); err != nil {
			return fmt.Errorf("gateway base_domain required for TCP TLS: %w", err)
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
		if expose.Protocol != "http" {
			continue
		}
		if expose.Access != exposeAccessPublic {
			continue
		}
		path := ptrString(expose.PathPrefix)
		key := host + "|" + path
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate http route host+path for version: %s%s (use distinct path_prefix)", host, path)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// deriveHost builds {app_code}.{gateway.base_domain}.
func deriveHost(gateway *model.GatewayConfig, appCode string) (string, error) {
	if gateway == nil {
		return "", fmt.Errorf("gateway config required for host derivation")
	}
	baseDomain := gateway.BaseDomain
	if baseDomain == "" {
		return "", fmt.Errorf("gateway base_domain is required for host derivation")
	}
	if appCode == "" {
		return "", fmt.Errorf("app code is required for host derivation")
	}
	host := appCode + "." + baseDomain
	if !validDeploymentRouteDomain(host) {
		return "", fmt.Errorf("invalid derived host %q", host)
	}
	return host, nil
}

// injectGatewayDashboardLabels writes Traefik self-route labels for the dashboard/API.
func injectGatewayDashboardLabels(services map[string]any, input RenderInput) error {
	if input.Gateway == nil {
		return nil
	}
	entrypoint := input.Gateway.DefaultEntrypoint
	if entrypoint == "" {
		return fmt.Errorf("gateway default_entrypoint is required")
	}
	if !validGatewayEntrypoint(entrypoint) {
		return fmt.Errorf("gateway default_entrypoint must be web or websecure")
	}
	host, err := deriveHost(input.Gateway, input.App.Code)
	if err != nil {
		return err
	}
	if len(input.Components) == 0 {
		return nil
	}
	componentName := input.Components[0].Name
	service, ok := services[componentName].(map[string]any)
	if !ok {
		return fmt.Errorf("gateway component %s not found in compose services", componentName)
	}
	routerName := fmt.Sprintf("%s-%s-dashboard", input.App.Code, input.Service.InstanceKey)
	labels := []string{
		"traefik.enable=true",
		"traefik.http.routers." + routerName + ".rule=Host(`" + host + "`)",
		"traefik.http.routers." + routerName + ".entrypoints=" + entrypoint,
		"traefik.http.routers." + routerName + ".service=api@internal",
	}
	tlsMode := input.Gateway.TLSMode
	switch tlsMode {
	case "letsencrypt":
		labels = append(labels,
			"traefik.http.routers."+routerName+".tls=true",
			"traefik.http.routers."+routerName+".tls.certresolver=letsencrypt",
		)
	case "tls":
		labels = append(labels, "traefik.http.routers."+routerName+".tls=true")
	case "none":
	default:
		return fmt.Errorf("gateway tls_mode must be none, letsencrypt or tls")
	}
	existing, _ := service["labels"].([]string)
	service["labels"] = append(existing, labels...)
	return nil
}

func buildTraefikLabels(routerName string, expose model.VersionExpose, input RenderInput, host string) ([]string, error) {
	labels := []string{"traefik.enable=true"}
	if input.Gateway == nil {
		return nil, fmt.Errorf("gateway config required for public expose labels")
	}
	entrypoint := input.Gateway.DefaultEntrypoint
	tlsMode := input.Gateway.TLSMode
	if tlsMode != "none" && tlsMode != "letsencrypt" && tlsMode != "tls" {
		return nil, fmt.Errorf("gateway tls_mode must be none, letsencrypt or tls")
	}

	switch expose.Protocol {
	case "http":
		if entrypoint == "" {
			return nil, fmt.Errorf("gateway default_entrypoint is required for %s", expose.ComponentName)
		}
		if !validGatewayEntrypoint(entrypoint) {
			return nil, fmt.Errorf("gateway default_entrypoint must be web or websecure")
		}
		if host == "" {
			derived, err := deriveHost(input.Gateway, input.App.Code)
			if err != nil {
				return nil, err
			}
			host = derived
		}
		rule := "Host(`" + host + "`)"
		path := ptrString(expose.PathPrefix)
		if path != "" && path != "/" {
			rule = rule + " && PathPrefix(`" + path + "`)"
		}
		labels = append(labels,
			"traefik.http.routers."+routerName+".rule="+rule,
			"traefik.http.routers."+routerName+".entrypoints="+entrypoint,
			"traefik.http.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.http.routers."+routerName+".service="+routerName,
		)
		switch tlsMode {
		case "letsencrypt":
			labels = append(labels,
				"traefik.http.routers."+routerName+".tls=true",
				"traefik.http.routers."+routerName+".tls.certresolver=letsencrypt",
			)
		case "tls":
			labels = append(labels, "traefik.http.routers."+routerName+".tls=true")
		}
	case "tcp":
		listen := effectiveListen(expose)
		tcpEP := tcpEntrypointName(listen)
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
			"traefik.tcp.routers."+routerName+".entrypoints="+tcpEP,
			"traefik.tcp.services."+routerName+".loadbalancer.server.port="+strconv.Itoa(expose.ContainerPort),
			"traefik.tcp.routers."+routerName+".service="+routerName,
		)
		switch tlsMode {
		case "letsencrypt":
			labels = append(labels,
				"traefik.tcp.routers."+routerName+".tls=true",
				"traefik.tcp.routers."+routerName+".tls.certresolver=letsencrypt",
			)
		case "tls":
			labels = append(labels, "traefik.tcp.routers."+routerName+".tls=true")
		}
	default:
		return nil, fmt.Errorf("unsupported expose protocol %s", expose.Protocol)
	}
	return labels, nil
}

func exposeKey(componentName string, protocol string, port int) string {
	return componentName + "|" + protocol + "|" + strconv.Itoa(port)
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func validateVersionComponents(components []model.VersionComponent) error {
	if len(components) == 0 {
		return fmt.Errorf("at least one component is required")
	}
	names := make(map[string]struct{}, len(components))
	for _, component := range components {
		name := component.Name
		if name == "" {
			return fmt.Errorf("component name is required")
		}
		if component.Image == "" {
			return fmt.Errorf("component %s image is required", name)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("duplicate component name %s", name)
		}
		names[name] = struct{}{}
		for _, mount := range component.Mounts {
			if err := validateMountSpec(mount); err != nil {
				return fmt.Errorf("component %s mount: %w", name, err)
			}
		}
		if err := validateComponentRuntimeFields(component); err != nil {
			return err
		}
	}
	for _, component := range components {
		for _, dependency := range component.Dependencies {
			if dependency.Name == component.Name {
				return fmt.Errorf("component %s depends on itself", component.Name)
			}
			if _, ok := names[dependency.Name]; !ok {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, dependency.Name)
			}
			switch dependency.Condition {
			case "service_started", "service_healthy", "service_completed_successfully":
			default:
				return fmt.Errorf("component %s dependency %s has unsupported condition", component.Name, dependency.Name)
			}
		}
	}
	return nil
}

func validateVersionExposes(exposes []model.VersionExpose, components []model.VersionComponent) error {
	names := make(map[string]struct{}, len(components))
	for _, c := range components {
		names[c.Name] = struct{}{}
	}
	seen := map[string]struct{}{}
	for _, expose := range exposes {
		name := expose.ComponentName
		if name == "" {
			return fmt.Errorf("expose component_name is required")
		}
		if _, ok := names[name]; !ok {
			return fmt.Errorf("expose component %s not found in version components", name)
		}
		protocol := expose.Protocol
		if protocol != "http" && protocol != "tcp" {
			return fmt.Errorf("expose protocol must be http or tcp")
		}
		if expose.ContainerPort < 1 || expose.ContainerPort > 65535 {
			return fmt.Errorf("expose container_port out of range")
		}
		access := expose.Access
		if access != exposeAccessLocal && access != exposeAccessPublic {
			return fmt.Errorf("expose access must be local or public")
		}
		if protocol == "tcp" && ptrString(expose.PathPrefix) != "" {
			return fmt.Errorf("path_prefix is only allowed for http expose")
		}
		if expose.ListenPort != nil && *expose.ListenPort != 0 {
			if *expose.ListenPort < 1 || *expose.ListenPort > 65535 {
				return fmt.Errorf("expose listen_port out of range")
			}
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
	appCode string,
	physicalServiceDir string,
	runtime map[string]string,
) (map[string]any, []ResolvedMount, error) {
	service := map[string]any{
		"image":          component.Image,
		"container_name": runtimeName(appCode, component.Name),
	}
	if len(component.Command) > 0 {
		service["command"] = append([]string(nil), component.Command...)
	}
	env := componentEnv(component.Env, runtime)
	if len(env) > 0 {
		service["environment"] = env
	}
	if err := applyComponentRuntimeFields(service, component); err != nil {
		return nil, nil, err
	}
	if len(component.Ports) > 0 {
		ports := make([]string, 0, len(component.Ports))
		for _, port := range component.Ports {
			ports = append(ports, fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort))
		}
		service["ports"] = ports
	}
	resolved, err := resolveMountSpecs(component.Mounts, physicalServiceDir)
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
	if len(component.Dependencies) > 0 {
		depends := make(map[string]map[string]string, len(component.Dependencies))
		for _, dependency := range component.Dependencies {
			depends[dependency.Name] = map[string]string{"condition": dependency.Condition}
		}
		service["depends_on"] = depends
	}
	if component.Healthcheck != nil {
		healthcheck, err := renderComponentHealthcheck(component.Healthcheck)
		if err != nil {
			return nil, nil, err
		}
		service["healthcheck"] = healthcheck
	}
	if component.Resources != nil {
		service["deploy"] = map[string]any{"resources": renderComponentResources(component.Resources)}
	}
	if component.PullPolicy != nil && *component.PullPolicy != "" {
		service["pull_policy"] = *component.PullPolicy
	}
	return service, resolved, nil
}

func renderComponentHealthcheck(healthcheck *model.VersionComponentHealthcheck) (map[string]any, error) {
	if healthcheck.Disabled {
		return map[string]any{"disable": true}, nil
	}
	test := []string{healthcheck.TestMode}
	switch healthcheck.TestMode {
	case "CMD":
		args, err := commandline.Parse(healthcheck.Test)
		if err != nil {
			return nil, fmt.Errorf("parse healthcheck command: %w", err)
		}
		test = append(test, args...)
	case "CMD-SHELL":
		test = append(test, healthcheck.Test)
	default:
		return nil, fmt.Errorf("unsupported healthcheck test mode %q", healthcheck.TestMode)
	}
	result := map[string]any{"test": test}
	if healthcheck.Interval != nil {
		result["interval"] = *healthcheck.Interval
	}
	if healthcheck.Timeout != nil {
		result["timeout"] = *healthcheck.Timeout
	}
	if healthcheck.Retries != nil {
		result["retries"] = *healthcheck.Retries
	}
	if healthcheck.StartPeriod != nil {
		result["start_period"] = *healthcheck.StartPeriod
	}
	if healthcheck.StartInterval != nil {
		result["start_interval"] = *healthcheck.StartInterval
	}
	return result, nil
}

func renderComponentResources(resources *model.VersionComponentResources) map[string]any {
	result := map[string]any{}
	if limits := resourceValues(resources.LimitCPUs, resources.LimitMemory); len(limits) > 0 {
		result["limits"] = limits
	}
	if reservations := resourceValues(resources.ReservationCPUs, resources.ReservationMemory); len(reservations) > 0 {
		result["reservations"] = reservations
	}
	return result
}

func resourceValues(cpus *string, memory *string) map[string]string {
	values := map[string]string{}
	if cpus != nil {
		values["cpus"] = *cpus
	}
	if memory != nil {
		values["memory"] = *memory
	}
	return values
}

func validDeploymentRouteDomain(domain string) bool {
	return domain != "" && len(domain) <= 253 && !strings.ContainsAny(domain, " `\t\r\n")
}
