package deploymentsvc

import (
	"fmt"
	"slices"
	"strings"

	deploymentrender "github.com/leoninew/pomelo-orbit/internal/application/deployment/render"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	gatewayDNSApiEnvKey          = "CF_DNS_API_TOKEN"
	gatewayMountTargetTraefikYml = "/etc/traefik/traefik.yml"
)

func requiresGatewayConfig(plan model.EffectiveServicePlan) bool {
	return isGatewayCarrier(plan) || (plan.JoinsTraefikNetwork() && hasGatewayEndpoint(plan))
}

func isGatewayCarrier(plan model.EffectiveServicePlan) bool {
	return plan.Gateway != nil && plan.Gateway.ApplicationId != "" && plan.Gateway.ApplicationId == plan.Application.Id
}

func hasGatewayEndpoint(plan model.EffectiveServicePlan) bool {
	for _, component := range plan.Components {
		for _, endpoint := range component.Endpoints {
			if endpoint.Mode == "gateway" {
				return true
			}
		}
	}
	return false
}

// enrichGatewayPlan applies managed Route publication settings and patches
// the ACME values declared by the selected Gateway Version.
func enrichGatewayPlan(plan *model.EffectiveServicePlan) error {
	if !isGatewayCarrier(*plan) {
		return nil
	}
	// Only these slices are changed by enrichment. Preserve declaration ownership,
	// including on validation/encoding failure, and publish the prepared plan once.
	prepared := *plan
	prepared.Components = slices.Clone(plan.Components)
	for index := range prepared.Components {
		prepared.Components[index].Mounts = slices.Clone(prepared.Components[index].Mounts)
		prepared.Components[index].Env = slices.Clone(prepared.Components[index].Env)
	}
	if err := applyGatewayPlan(&prepared); err != nil {
		return err
	}
	*plan = prepared
	return nil
}

func applyGatewayPlan(plan *model.EffectiveServicePlan) error {
	cfg := plan.Gateway
	role := cfg.AcmeProfile
	if role == "" {
		role = "base"
	}
	versionId := cfg.VersionIdForProfile(role)
	if versionId == "" {
		return fmt.Errorf("gateway Version binding is missing for %s", role)
	}
	if plan.Version.Id != versionId {
		return fmt.Errorf("selected Gateway Version does not match the %s binding", role)
	}
	component := gatewayPlanComponent(plan, model.GatewayComponentName())
	if component == nil {
		return fmt.Errorf("selected Version does not contain Gateway component %q", model.GatewayComponentName())
	}
	file := gatewayStaticConfigMount(component)
	if file == nil {
		return fmt.Errorf("gateway component %q does not declare %s", component.Name, gatewayMountTargetTraefikYml)
	}
	for _, required := range []model.VersionComponentMount{
		{SourceType: "directory", Source: model.GatewayRouteConfigSource, Target: model.GatewayRouteConfigTarget, ReadOnly: true},
		{SourceType: "directory", Source: "./gateway/certs", Target: "/etc/traefik/certs", ReadOnly: true},
		{SourceType: "directory", Source: "./gateway/acme", Target: "/letsencrypt"},
	} {
		found := false
		for index, mount := range component.Mounts {
			if mount.Target == required.Target {
				component.Mounts[index] = required
				found = true
				break
			}
		}
		if !found {
			component.Mounts = append(component.Mounts, required)
		}
	}
	var resolvers []string
	if cfg.AcmeProfile != "" {
		if strings.TrimSpace(cfg.AcmeEmail) == "" {
			return fmt.Errorf("gateway ACME email is required for profile %s", cfg.AcmeProfile)
		}
		resolvers = gatewayProfileResolvers(cfg.AcmeProfile)
		if len(resolvers) == 0 {
			return fmt.Errorf("gateway ACME profile %q is invalid", cfg.AcmeProfile)
		}
		if cfg.AcmeProfile == "dns" || cfg.AcmeProfile == "http-dns" {
			if strings.TrimSpace(cfg.DNSApiToken) == "" {
				return fmt.Errorf("gateway DNS API token is required for profile %s", cfg.AcmeProfile)
			}
			setEffectiveComponentEnv(component, gatewayDNSApiEnvKey, cfg.DNSApiToken)
		}
	}
	file = gatewayStaticConfigMount(component)
	patched, err := deploymentrender.PatchGatewayConfig(file.Content, deploymentrender.GatewayConfigOptions{RouteDirectory: model.GatewayRouteConfigTarget, Resolvers: resolvers, Email: cfg.AcmeEmail})
	if err != nil {
		return err
	}
	file.Content, file.IgnoreIfExists = patched, false
	return nil
}

func gatewayPlanComponent(plan *model.EffectiveServicePlan, name string) *model.EffectiveServiceComponent {
	for index := range plan.Components {
		if plan.Components[index].Name == name {
			return &plan.Components[index]
		}
	}
	return nil
}

func gatewayStaticConfigMount(component *model.EffectiveServiceComponent) *model.VersionComponentMount {
	for index := range component.Mounts {
		mount := &component.Mounts[index]
		if mount.Target == gatewayMountTargetTraefikYml && mount.SourceType == "controlled_file" {
			return mount
		}
	}
	return nil
}

func gatewayProfileResolvers(profile string) []string {
	switch profile {
	case "http":
		return []string{"letsencrypt"}
	case "dns":
		return []string{"letsencrypt-dns"}
	case "http-dns":
		return []string{"letsencrypt", "letsencrypt-dns"}
	default:
		return nil
	}
}

func setEffectiveComponentEnv(component *model.EffectiveServiceComponent, key, value string) {
	for index := range component.Env {
		if component.Env[index].Key == key {
			component.Env[index].Value = value
			return
		}
	}
	component.Env = append(component.Env, model.VersionComponentEnv{Key: key, Value: value})
}
