package applicationsvc

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	exposeAccessLocal  = "local"
	exposeAccessPublic = "public"
)

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
		if err := validateOptionalJSON(component.MountsJSON, "mounts_json", name); err != nil {
			return err
		}
		if err := validateOptionalJSON(component.EnvJSON, "env_json", name); err != nil {
			return err
		}
		names[name] = struct{}{}
	}
	for _, component := range components {
		var depends []string
		if component.DependsOnJSON != nil && strings.TrimSpace(*component.DependsOnJSON) != "" {
			if err := json.Unmarshal([]byte(*component.DependsOnJSON), &depends); err != nil {
				return fmt.Errorf("component %s depends_on_json: must be a string array", component.Name)
			}
		}
		for _, dependency := range depends {
			if _, ok := names[dependency]; !ok {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, dependency)
			}
		}
	}
	return nil
}

func validateVersionExposes(exposes []model.VersionExpose, components []model.VersionComponent) error {
	names := make(map[string]struct{}, len(components))
	for _, component := range components {
		names[strings.TrimSpace(component.Name)] = struct{}{}
	}
	seen := make(map[string]struct{}, len(exposes))
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
		access := strings.ToLower(strings.TrimSpace(expose.Access))
		if access == "" {
			access = exposeAccessPublic
		}
		if access != exposeAccessLocal && access != exposeAccessPublic {
			return fmt.Errorf("expose access must be local or public")
		}
		if protocol == "tcp" && expose.PathPrefix != nil && strings.TrimSpace(*expose.PathPrefix) != "" {
			return fmt.Errorf("path_prefix is only allowed for http expose")
		}
		if expose.ListenPort != nil && *expose.ListenPort != 0 && (*expose.ListenPort < 1 || *expose.ListenPort > 65535) {
			return fmt.Errorf("expose listen_port out of range")
		}
		key := fmt.Sprintf("%s/%s/%d", name, protocol, expose.ContainerPort)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate expose %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateOptionalJSON(value *string, field string, component string) error {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	if !json.Valid([]byte(*value)) {
		return fmt.Errorf("component %s %s: invalid JSON", component, field)
	}
	return nil
}
