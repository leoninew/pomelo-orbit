package deploymentsvc

import (
	"fmt"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
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
	if plan.Gateway == nil {
		return fmt.Errorf("gateway configuration snapshot is required")
	}
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
	patched, err := patchGatewayRouteProvider(file.Content)
	if err != nil {
		return err
	}
	file.Content = patched
	file.IgnoreIfExists = false
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
	if cfg.AcmeProfile == "" {
		return nil
	}
	if strings.TrimSpace(cfg.AcmeEmail) == "" {
		return fmt.Errorf("gateway ACME email is required for profile %s", cfg.AcmeProfile)
	}
	resolvers := gatewayProfileResolvers(cfg.AcmeProfile)
	if len(resolvers) == 0 {
		return fmt.Errorf("gateway ACME profile %q is invalid", cfg.AcmeProfile)
	}
	file = gatewayStaticConfigMount(component)
	patched, err = patchDeclaredResolverEmails(file.Content, resolvers, cfg.AcmeEmail)
	if err != nil {
		return err
	}
	file.Content = patched
	if cfg.AcmeProfile == "dns" || cfg.AcmeProfile == "http-dns" {
		if strings.TrimSpace(cfg.DNSApiToken) == "" {
			return fmt.Errorf("gateway DNS API token is required for profile %s", cfg.AcmeProfile)
		}
		setEffectiveComponentEnv(component, gatewayDNSApiEnvKey, cfg.DNSApiToken)
	}
	return nil
}

func patchGatewayRouteProvider(content string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", fmt.Errorf("parse declared Gateway static config: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("declared Gateway static config must be a YAML mapping")
	}
	root := document.Content[0]
	providers := yamlMappingValue(root, "providers")
	if providers == nil {
		providers = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "providers"}, providers)
	}
	if providers.Kind != yaml.MappingNode {
		return "", fmt.Errorf("declared Gateway providers must be a YAML mapping")
	}
	var file yaml.Node
	if err := file.Encode(struct {
		Directory string `yaml:"directory"`
		Watch     bool   `yaml:"watch"`
	}{Directory: model.GatewayRouteConfigTarget, Watch: true}); err != nil {
		return "", fmt.Errorf("encode managed Gateway File provider: %w", err)
	}
	entries := make([]*yaml.Node, 0, len(providers.Content)+2)
	for index := 0; index+1 < len(providers.Content); index += 2 {
		if key := providers.Content[index].Value; key != "file" && key != "rest" {
			entries = append(entries, providers.Content[index], providers.Content[index+1])
		}
	}
	providers.Content = append(entries, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "file"}, &file)
	result, err := yaml.Marshal(&document)
	if err != nil {
		return "", fmt.Errorf("marshal Gateway static config: %w", err)
	}
	return string(result), nil
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

func patchDeclaredResolverEmails(content string, resolvers []string, email string) (string, error) {
	var document yaml.Node
	if err := yaml.Unmarshal([]byte(content), &document); err != nil {
		return "", fmt.Errorf("parse declared Gateway static config: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("declared Gateway static config must be a YAML mapping")
	}
	resolverMap := yamlMappingValue(document.Content[0], "certificatesResolvers")
	if resolverMap == nil || resolverMap.Kind != yaml.MappingNode {
		return "", fmt.Errorf("declared Gateway static config is missing certificatesResolvers")
	}
	for _, name := range resolvers {
		resolver := yamlMappingValue(resolverMap, name)
		if resolver == nil || resolver.Kind != yaml.MappingNode {
			return "", fmt.Errorf("declared Gateway static config is missing resolver %q", name)
		}
		acme := yamlMappingValue(resolver, "acme")
		if acme == nil || acme.Kind != yaml.MappingNode {
			return "", fmt.Errorf("declared Gateway resolver %q is missing acme", name)
		}
		emailNode := yamlMappingValue(acme, "email")
		if emailNode == nil || emailNode.Kind != yaml.ScalarNode {
			return "", fmt.Errorf("declared Gateway resolver %q is missing acme.email", name)
		}
		emailNode.Tag = "!!str"
		emailNode.Value = email
		emailNode.Style = yaml.DoubleQuotedStyle
	}
	result, err := yaml.Marshal(&document)
	if err != nil {
		return "", fmt.Errorf("marshal declared Gateway static config: %w", err)
	}
	return string(result), nil
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
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
