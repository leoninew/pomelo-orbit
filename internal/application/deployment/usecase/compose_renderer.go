package deploymentsvc

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/common/commandline"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

// RenderInput contains the merged, Gateway-enriched desired state for one Service.
// Rendering never reads Version defaults and Service overlays independently.
type RenderInput struct {
	ResolveMountSource    func(context.Context, string) (string, error)
	Plan                  model.EffectiveServicePlan
	LogicalSvcDir         string
	ComposeMountSourceDir string
}

type RenderResult struct {
	Compose        string
	ResolvedMounts []ResolvedMount
}

const gatewayNetworkKey = "traefik"
const consumerPlatformNetworkKey = "traefik"

func RenderCompose(ctx context.Context, input RenderInput) (string, error) {
	result, err := RenderComposeDetailed(ctx, input)
	if err != nil {
		return "", err
	}
	return result.Compose, nil
}

func RenderComposeDetailed(ctx context.Context, input RenderInput) (RenderResult, error) {
	if len(input.Plan.Components) == 0 {
		return RenderResult{}, fmt.Errorf("service %s has no effective components", input.Plan.Service.Id)
	}
	policy, err := resolveComposePolicy(input.Plan)
	if err != nil {
		return RenderResult{}, err
	}
	services := make(map[string]any, len(input.Plan.Components))
	volumes := map[string]any{}
	var resolved []ResolvedMount
	for _, effective := range input.Plan.Components {
		component := versionComponentFromEffective(effective)
		if err := validateVersionComponent(component); err != nil {
			return RenderResult{}, err
		}
		service, mounts, err := renderComponentService(component, input.Plan.Application.Code, input.LogicalSvcDir, input.ComposeMountSourceDir)
		if err != nil {
			return RenderResult{}, fmt.Errorf("component %s: %w", component.Name, err)
		}
		if input.ResolveMountSource != nil {
			volumes := make([]string, 0, len(mounts))
			for index := range mounts {
				mount := &mounts[index]
				if mount.Relative {
					mapped, err := input.ResolveMountSource(ctx, mount.LogicalSource)
					if err != nil {
						return RenderResult{}, err
					}
					mount.Source = mapped
					mount.HostSource = mapped
				}
				volumes = append(volumes, renderMount(*mount))
			}
			if len(volumes) > 0 {
				service["volumes"] = volumes
			}
		}
		services[component.Name] = service
		resolved = append(resolved, mounts...)
		for _, mount := range mounts {
			if mount.NamedVolumeName != "" {
				volumes[mount.NamedVolumeName] = map[string]any{}
			}
		}
	}
	applyComposePolicy(services, input.Plan, policy)
	data := map[string]any{"services": services}
	if len(volumes) > 0 {
		data["volumes"] = volumes
	}
	if policy.gatewayCarrier {
		data["networks"] = map[string]any{gatewayNetworkKey: map[string]any{"name": policy.networkName, "driver": "bridge"}}
	} else if policy.joinNetwork {
		data["networks"] = map[string]any{consumerPlatformNetworkKey: map[string]any{"name": policy.networkName, "external": true}}
	}
	content, err := yaml.Marshal(data)
	if err != nil {
		return RenderResult{}, fmt.Errorf("marshal compose: %w", err)
	}
	return RenderResult{Compose: string(content), ResolvedMounts: resolved}, nil
}

func versionComponentFromEffective(component model.EffectiveServiceComponent) model.VersionComponent {
	return model.VersionComponent{Id: component.SourceComponentId, Name: component.Name, Image: component.Image, Entrypoint: component.Entrypoint, Command: component.Command, User: component.User, GroupAdd: component.GroupAdd, Env: component.Env, Mounts: component.Mounts, Dependencies: component.Dependencies, Healthcheck: component.Healthcheck, Resources: component.Resources, PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy, Tmpfs: component.Tmpfs, Ulimits: component.Ulimits, Devices: component.Devices, Endpoints: component.Endpoints}
}

func routerName(plan model.EffectiveServicePlan, component, endpoint string) string {
	return plan.Service.Code + "-" + component + "-" + endpoint
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

func renderComponentService(component model.VersionComponent, appCode, logicalServiceDir, composeMountSourceDir string) (map[string]any, []ResolvedMount, error) {
	service := map[string]any{"image": component.Image, "container_name": model.RuntimeContainerName(appCode, component.Name)}
	if len(component.Entrypoint) > 0 {
		service["entrypoint"] = append([]string(nil), component.Entrypoint...)
	}
	if len(component.Command) > 0 {
		service["command"] = append([]string(nil), component.Command...)
	}
	if len(component.Env) > 0 {
		env := make(map[string]string, len(component.Env))
		for _, item := range component.Env {
			// Effective values are literals; only the Compose interpolation escape remains.
			env[item.Key] = strings.ReplaceAll(item.Value, "$", "$$")
		}
		service["environment"] = env
	}
	if err := applyComponentRuntimeFields(service, component); err != nil {
		return nil, nil, err
	}
	resolved, err := resolveMountSpecsForPaths(component.Mounts, logicalServiceDir, composeMountSourceDir)
	if err != nil {
		return nil, nil, err
	}
	sharedLabel, err := renderSharedMountsLabel(component.Mounts)
	if err != nil {
		return nil, nil, err
	}
	if sharedLabel != "" {
		appendString(service, "labels", sharedMountsLabel+"="+sharedLabel)
	}
	if len(resolved) > 0 {
		volumes := make([]string, 0, len(resolved))
		for _, item := range resolved {
			volumes = append(volumes, renderMount(item))
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

func applyComposePolicy(services map[string]any, plan model.EffectiveServicePlan, policy composePolicy) {
	for _, port := range policy.ports {
		service := services[port.component].(map[string]any)
		appendString(service, "ports", net.JoinHostPort(port.address, strconv.Itoa(port.hostPort))+":"+strconv.Itoa(port.containerPort))
	}
	for _, route := range policy.routes {
		service := services[route.component].(map[string]any)
		router := routerName(plan, route.component, model.EndpointDisplayName("http", route.containerPort))
		rule := "Host(`" + route.host + "`)"
		if route.pathPrefix != "/" {
			rule += " && PathPrefix(`" + route.pathPrefix + "`)"
		}
		appendStrings(service, "labels", []string{"traefik.enable=true", "traefik.http.routers." + router + ".rule=" + rule, "traefik.http.routers." + router + ".entrypoints=" + route.entrypoint, "traefik.http.routers." + router + ".service=" + router, "traefik.http.services." + router + ".loadbalancer.server.port=" + strconv.Itoa(route.containerPort)})
		appendTLSLabels(service, router, "http", plan.Gateway.TLSMode)
	}
	if policy.gatewayCarrier {
		for _, raw := range services {
			raw.(map[string]any)["networks"] = []string{gatewayNetworkKey}
		}
		service := services[plan.Components[0].Name].(map[string]any)
		router := plan.Service.Code + "-dashboard"
		appendStrings(service, "labels", []string{"traefik.enable=true", "traefik.http.routers." + router + ".rule=Host(`" + policy.dashboardHost + "`)", "traefik.http.routers." + router + ".entrypoints=" + policy.dashboardEntrypoint, "traefik.http.routers." + router + ".service=api@internal"})
		appendTLSLabels(service, router, "http", plan.Gateway.TLSMode)
	} else if policy.joinNetwork {
		for name, raw := range services {
			raw.(map[string]any)["networks"] = map[string]any{"default": map[string]any{}, consumerPlatformNetworkKey: map[string]any{"aliases": []string{model.RuntimeContainerName(plan.Application.Code, name)}}}
		}
	}
}

func renderMount(mount ResolvedMount) string {
	value := mount.Source + ":" + mount.Target
	if mount.ReadOnly {
		value += ":ro"
	}
	return value
}
