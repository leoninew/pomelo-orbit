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

// RenderInput contains the already-merged desired state for one Service.
// Rendering never reads Version defaults and Service overlays independently.
type RenderInput struct {
	Plan           model.EffectiveServicePlan
	PhysicalSvcDir string
}

type RenderResult struct {
	Compose        string
	ResolvedMounts []ResolvedMount
}

const defaultGatewayNetworkName = "traefik"
const gatewayNetworkKey = "traefik"
const consumerPlatformNetworkKey = "traefik"

func (s Service) RenderCompose(ctx context.Context, input RenderInput) (string, error) {
	result, err := s.RenderComposeDetailed(ctx, input)
	if err != nil {
		return "", err
	}
	return result.Compose, nil
}

func (s Service) RenderComposeDetailed(ctx context.Context, input RenderInput) (RenderResult, error) {
	_ = ctx
	if len(input.Plan.Components) == 0 {
		return RenderResult{}, fmt.Errorf("service %s has no effective components", input.Plan.Service.Id)
	}
	services := make(map[string]any, len(input.Plan.Components))
	volumes := map[string]any{}
	var resolved []ResolvedMount
	for _, effective := range input.Plan.Components {
		component := versionComponentFromEffective(effective)
		if err := validateVersionComponent(component); err != nil {
			return RenderResult{}, err
		}
		service, mounts, err := renderVersionComponentService(component, input.Plan.Application.Code, input.PhysicalSvcDir, nil, false)
		if err != nil {
			return RenderResult{}, fmt.Errorf("component %s: %w", component.Name, err)
		}
		services[component.Name] = service
		resolved = append(resolved, mounts...)
		for _, mount := range mounts {
			if mount.NamedVolumeName != "" {
				volumes[mount.NamedVolumeName] = map[string]any{}
			}
		}
	}
	if err := applyEffectiveEndpoints(services, input.Plan); err != nil {
		return RenderResult{}, err
	}
	switch input.Plan.Application.Kind {
	case status.ApplicationKindGateway:
		for _, raw := range services {
			raw.(map[string]any)["networks"] = []string{gatewayNetworkKey}
		}
		if err := injectGatewayDashboardLabels(services, input.Plan); err != nil {
			return RenderResult{}, err
		}
	case status.ApplicationKindStandard:
		if err := injectAllComponentsPlatformNetwork(services, input.Plan.Application.Code); err != nil {
			return RenderResult{}, err
		}
	default:
		return RenderResult{}, fmt.Errorf("unsupported application kind %q", input.Plan.Application.Kind)
	}
	data := map[string]any{"services": services}
	if len(volumes) > 0 {
		data["volumes"] = volumes
	}
	if input.Plan.Application.Kind == status.ApplicationKindGateway {
		data["networks"] = map[string]any{gatewayNetworkKey: map[string]any{"name": defaultGatewayNetworkName, "driver": "bridge"}}
	} else {
		data["networks"] = map[string]any{consumerPlatformNetworkKey: map[string]any{"name": defaultGatewayNetworkName, "external": true}}
	}
	content, err := yaml.Marshal(data)
	if err != nil {
		return RenderResult{}, fmt.Errorf("marshal compose: %w", err)
	}
	return RenderResult{Compose: string(content), ResolvedMounts: resolved}, nil
}

func versionComponentFromEffective(component model.EffectiveServiceComponent) model.VersionComponent {
	return model.VersionComponent{Id: component.SourceComponentId, Name: component.Name, Image: component.Image, Entrypoint: component.Entrypoint, Command: component.Command, Env: component.Env, Mounts: component.Mounts, Dependencies: component.Dependencies, Healthcheck: component.Healthcheck, Resources: component.Resources, PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy, Tmpfs: component.Tmpfs, Ulimits: component.Ulimits, Devices: component.Devices, Endpoints: component.Endpoints}
}

func applyEffectiveEndpoints(services map[string]any, plan model.EffectiveServicePlan) error {
	host := ""
	for _, component := range plan.Components {
		service, ok := services[component.Name].(map[string]any)
		if !ok {
			return fmt.Errorf("effective component %s is missing from compose", component.Name)
		}
		for _, endpoint := range component.Endpoints {
			switch endpoint.Mode {
			case "internal":
				continue
			case "local", "host":
				if endpoint.ListenPort == nil {
					return fmt.Errorf("endpoint %s/%s requires listen_port", component.Name, endpoint.Name)
				}
				address := "0.0.0.0"
				if endpoint.Mode == "local" {
					address = "127.0.0.1"
				}
				if endpoint.BindAddress != nil && *endpoint.BindAddress != "" {
					address = *endpoint.BindAddress
				}
				if endpoint.Mode == "local" && address != "127.0.0.1" && address != "::1" {
					return fmt.Errorf("local endpoint %s/%s must bind loopback", component.Name, endpoint.Name)
				}
				appendString(service, "ports", fmt.Sprintf("%s:%d:%d", address, *endpoint.ListenPort, endpoint.ContainerPort))
			case "gateway_http":
				if endpoint.Protocol != "http" {
					return fmt.Errorf("gateway_http endpoint %s/%s must use http", component.Name, endpoint.Name)
				}
				if plan.Gateway == nil {
					return fmt.Errorf("gateway config required for endpoint %s/%s", component.Name, endpoint.Name)
				}
				if host == "" {
					var err error
					host, err = deriveHost(plan.Gateway, plan.Application.Code)
					if err != nil {
						return err
					}
				}
				entrypoint := plan.Gateway.DefaultEntrypoint
				if endpoint.Entrypoint != nil && *endpoint.Entrypoint != "" {
					entrypoint = *endpoint.Entrypoint
				}
				if entrypoint == "" {
					return fmt.Errorf("gateway_http endpoint %s/%s requires entrypoint", component.Name, endpoint.Name)
				}
				router := routerName(plan, component.Name, endpoint.Name)
				rule := "Host(`" + host + "`)"
				if endpoint.PathPrefix != nil && *endpoint.PathPrefix != "" && *endpoint.PathPrefix != "/" {
					rule += " && PathPrefix(`" + *endpoint.PathPrefix + "`)"
				}
				appendStrings(service, "labels", []string{"traefik.enable=true", "traefik.http.routers." + router + ".rule=" + rule, "traefik.http.routers." + router + ".entrypoints=" + entrypoint, "traefik.http.routers." + router + ".service=" + router, "traefik.http.services." + router + ".loadbalancer.server.port=" + strconv.Itoa(endpoint.ContainerPort)})
				appendTLSLabels(service, router, "http", plan.Gateway.TLSMode)
			case "gateway_tcp":
				if endpoint.Protocol != "tcp" {
					return fmt.Errorf("gateway_tcp endpoint %s/%s must use tcp", component.Name, endpoint.Name)
				}
				if plan.Gateway == nil {
					return fmt.Errorf("gateway config required for endpoint %s/%s", component.Name, endpoint.Name)
				}
				if endpoint.Entrypoint == nil || *endpoint.Entrypoint == "" {
					return fmt.Errorf("gateway_tcp endpoint %s/%s requires entrypoint", component.Name, endpoint.Name)
				}
				router := routerName(plan, component.Name, endpoint.Name)
				sni := "*"
				if plan.Gateway.TLSMode == "letsencrypt" || plan.Gateway.TLSMode == "tls" {
					var err error
					sni, err = deriveHost(plan.Gateway, plan.Application.Code)
					if err != nil {
						return err
					}
				}
				appendStrings(service, "labels", []string{"traefik.enable=true", "traefik.tcp.routers." + router + ".rule=HostSNI(`" + sni + "`)", "traefik.tcp.routers." + router + ".entrypoints=" + *endpoint.Entrypoint, "traefik.tcp.routers." + router + ".service=" + router, "traefik.tcp.services." + router + ".loadbalancer.server.port=" + strconv.Itoa(endpoint.ContainerPort)})
				appendTLSLabels(service, router, "tcp", plan.Gateway.TLSMode)
			default:
				return fmt.Errorf("endpoint %s/%s has unsupported mode %q", component.Name, endpoint.Name, endpoint.Mode)
			}
		}
	}
	return nil
}

func routerName(plan model.EffectiveServicePlan, component, endpoint string) string {
	return plan.Application.Code + "-" + plan.Service.InstanceKey + "-" + component + "-" + endpoint
}

func appendTLSLabels(service map[string]any, router, protocol, mode string) {
	base := "traefik." + protocol + ".routers." + router + ".tls"
	switch mode {
	case "letsencrypt":
		appendStrings(service, "labels", []string{base + "=true", base + ".certresolver=letsencrypt"})
	case "tls":
		appendString(service, "labels", base+"=true")
	}
}

func appendString(service map[string]any, key, value string) {
	appendStrings(service, key, []string{value})
}
func appendStrings(service map[string]any, key string, values []string) {
	current, _ := service[key].([]string)
	service[key] = append(current, values...)
}

func deriveHost(gateway *model.GatewayConfig, appCode string) (string, error) {
	if gateway == nil || gateway.BaseDomain == "" {
		return "", fmt.Errorf("gateway base_domain is required")
	}
	if appCode == "" {
		return "", fmt.Errorf("application code is required")
	}
	host := appCode + "." + gateway.BaseDomain
	if !validDeploymentRouteDomain(host) {
		return "", fmt.Errorf("invalid derived host %q", host)
	}
	return host, nil
}

func injectGatewayDashboardLabels(services map[string]any, plan model.EffectiveServicePlan) error {
	if plan.Gateway == nil || len(plan.Components) == 0 {
		return nil
	}
	host, err := deriveHost(plan.Gateway, plan.Application.Code)
	if err != nil {
		return err
	}
	entrypoint := plan.Gateway.DefaultEntrypoint
	if entrypoint == "" {
		return fmt.Errorf("gateway default_entrypoint is required")
	}
	service := services[plan.Components[0].Name].(map[string]any)
	router := plan.Application.Code + "-" + plan.Service.InstanceKey + "-dashboard"
	appendStrings(service, "labels", []string{"traefik.enable=true", "traefik.http.routers." + router + ".rule=Host(`" + host + "`)", "traefik.http.routers." + router + ".entrypoints=" + entrypoint, "traefik.http.routers." + router + ".service=api@internal"})
	appendTLSLabels(service, router, "http", plan.Gateway.TLSMode)
	return nil
}

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
	service["networks"] = map[string]any{"default": map[string]any{}, consumerPlatformNetworkKey: map[string]any{"aliases": []string{alias}}}
}

func validateVersionComponent(component model.VersionComponent) error {
	if component.Name == "" || component.Image == "" {
		return fmt.Errorf("component name and image are required")
	}
	for _, mount := range component.Mounts {
		if err := validateMountSpec(mount); err != nil {
			return fmt.Errorf("component %s mount: %w", component.Name, err)
		}
	}
	return nil
}

func renderVersionComponentService(component model.VersionComponent, appCode, physicalServiceDir string, runtime map[string]string, _ bool) (map[string]any, []ResolvedMount, error) {
	service := map[string]any{"image": component.Image, "container_name": runtimeName(appCode, component.Name)}
	if len(component.Entrypoint) > 0 {
		service["entrypoint"] = append([]string(nil), component.Entrypoint...)
	}
	if len(component.Command) > 0 {
		service["command"] = append([]string(nil), component.Command...)
	}
	if env := componentEnv(component.Env, runtime); len(env) > 0 {
		service["environment"] = env
	}
	if err := applyComponentRuntimeFields(service, component); err != nil {
		return nil, nil, err
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
		dependencies := map[string]map[string]string{}
		for _, dependency := range component.Dependencies {
			dependencies[dependency.Name] = map[string]string{"condition": dependency.Condition}
		}
		service["depends_on"] = dependencies
	}
	if component.Healthcheck != nil {
		healthcheck, err := renderComponentHealthcheck(component.Healthcheck)
		if err != nil {
			return nil, nil, err
		}
		service["healthcheck"] = healthcheck
	}
	if component.Resources != nil || len(component.Devices) > 0 {
		service["deploy"] = map[string]any{"resources": renderComponentResources(component.Resources, component.Devices)}
	}
	service["pull_policy"] = component.PullPolicy
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

func renderComponentResources(resources *model.VersionComponentResources, devices []model.VersionComponentDeviceRequest) map[string]any {
	result := map[string]any{}
	if resources != nil {
		if limits := resourceValues(resources.LimitCPUs, resources.LimitMemory); len(limits) > 0 {
			result["limits"] = limits
		}
		if reservations := resourceValues(resources.ReservationCPUs, resources.ReservationMemory); len(reservations) > 0 {
			result["reservations"] = reservations
		}
	}
	if len(devices) > 0 {
		reservations, _ := result["reservations"].(map[string]string)
		devicesOut := make([]map[string]any, 0, len(devices))
		for _, device := range devices {
			devicesOut = append(devicesOut, map[string]any{"driver": device.Driver, "count": device.Count, "capabilities": append([]string(nil), device.Capabilities...)})
		}
		values := map[string]any{"devices": devicesOut}
		if reservations["cpus"] != "" {
			values["cpus"] = reservations["cpus"]
		}
		if reservations["memory"] != "" {
			values["memory"] = reservations["memory"]
		}
		result["reservations"] = values
	}
	return result
}

func resourceValues(cpus, memory *string) map[string]string {
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

func runtimeName(appCode, component string) string {
	name := strings.ToLower(strings.TrimSpace(appCode) + "-" + strings.TrimSpace(component))
	name = strings.Map(func(char rune) rune {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			return char
		}
		return '-'
	}, name)
	return strings.Trim(name, "-")
}
