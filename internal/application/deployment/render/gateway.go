// Package render encodes deployment output formats without selecting business policy.
package render

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// GatewayConfigOptions contains policy already selected by the deployment usecase.
type GatewayConfigOptions struct {
	RouteDirectory string
	Resolvers      []string
	Email          string
}

// PatchGatewayConfig replaces managed provider settings and declared resolver emails.
// Other YAML nodes, including custom providers and resolver layouts, are preserved.
func PatchGatewayConfig(content string, options GatewayConfigOptions) (string, error) {
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
	}{Directory: options.RouteDirectory, Watch: true}); err != nil {
		return "", fmt.Errorf("encode managed Gateway File provider: %w", err)
	}
	entries := make([]*yaml.Node, 0, len(providers.Content)+2)
	for index := 0; index+1 < len(providers.Content); index += 2 {
		if key := providers.Content[index].Value; key != "file" && key != "rest" {
			entries = append(entries, providers.Content[index], providers.Content[index+1])
		}
	}
	providers.Content = append(entries, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "file"}, &file)
	if err := patchResolverEmails(root, options.Resolvers, options.Email); err != nil {
		return "", err
	}
	result, err := yaml.Marshal(&document)
	if err != nil {
		return "", fmt.Errorf("marshal Gateway static config: %w", err)
	}
	return string(result), nil
}

func patchResolverEmails(root *yaml.Node, resolvers []string, email string) error {
	if len(resolvers) == 0 {
		return nil
	}
	resolverMap := yamlMappingValue(root, "certificatesResolvers")
	if resolverMap == nil || resolverMap.Kind != yaml.MappingNode {
		return fmt.Errorf("declared Gateway static config is missing certificatesResolvers")
	}
	for _, name := range resolvers {
		resolver := yamlMappingValue(resolverMap, name)
		if resolver == nil || resolver.Kind != yaml.MappingNode {
			return fmt.Errorf("declared Gateway static config is missing resolver %q", name)
		}
		acme := yamlMappingValue(resolver, "acme")
		if acme == nil || acme.Kind != yaml.MappingNode {
			return fmt.Errorf("declared Gateway resolver %q is missing acme", name)
		}
		emailNode := yamlMappingValue(acme, "email")
		if emailNode == nil || emailNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("declared Gateway resolver %q is missing acme.email", name)
		}
		emailNode.Tag = "!!str"
		emailNode.Value = email
		emailNode.Style = yaml.DoubleQuotedStyle
	}
	return nil
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}
