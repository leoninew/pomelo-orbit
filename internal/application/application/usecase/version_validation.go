package applicationsvc

import (
	"fmt"
	"strconv"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const maxMountContent = 256 * 1024

func validateVersionComponents(components []model.VersionComponent) error {
	names := make(map[string]struct{}, len(components))
	dependencies := make(map[string][]string, len(components))
	for _, component := range components {
		name := component.Name
		if name == "" || !applicationCreateCodePattern.MatchString(name) {
			return fmt.Errorf("component name must match ^[a-z][a-z0-9-]*$")
		}
		if component.Image == "" {
			return fmt.Errorf("component %s image is required", name)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("duplicate component name %s", name)
		}
		if err := validateComponentFields(component); err != nil {
			return err
		}
		names[name] = struct{}{}
	}
	for _, component := range components {
		seen := make(map[string]struct{}, len(component.Dependencies))
		for _, dependency := range component.Dependencies {
			name := dependency.Name
			if name == "" || name == component.Name {
				return fmt.Errorf("component %s has an invalid dependency", component.Name)
			}
			if _, exists := seen[name]; exists {
				return fmt.Errorf("component %s has duplicate dependency %s", component.Name, name)
			}
			if _, exists := names[name]; !exists {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, name)
			}
			switch dependency.Condition {
			case "service_started", "service_healthy", "service_completed_successfully":
			default:
				return fmt.Errorf("component %s dependency %s has unsupported condition", component.Name, name)
			}
			seen[name] = struct{}{}
			dependencies[component.Name] = append(dependencies[component.Name], name)
		}
	}
	visiting := make(map[string]bool, len(components))
	visited := make(map[string]bool, len(components))
	var visit func(string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("component dependencies contain a cycle")
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		for _, dependency := range dependencies[name] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		return nil
	}
	for name := range names {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func validateComponentFields(component model.VersionComponent) error {
	if !validImagePullPolicy(component.PullPolicy) {
		return fmt.Errorf("component %s pull_policy must be always, missing or never", component.Name)
	}
	if err := validateComponentRuntimeFields(component); err != nil {
		return err
	}
	env := make(map[string]struct{}, len(component.Env))
	for _, item := range component.Env {
		if item.Key == "" {
			return fmt.Errorf("component %s env key is required", component.Name)
		}
		if _, exists := env[item.Key]; exists {
			return fmt.Errorf("component %s has duplicate env key %s", component.Name, item.Key)
		}
		env[item.Key] = struct{}{}
	}
	endpoints := make(map[string]struct{}, len(component.Endpoints))
	for _, item := range component.Endpoints {
		if item.Name == "" || (item.Protocol != "http" && item.Protocol != "tcp") || item.ContainerPort < 1 || item.ContainerPort > 65535 {
			return fmt.Errorf("component %s endpoint is invalid", component.Name)
		}
		if item.Mode != "internal" && item.Mode != "local" && item.Mode != "host" && item.Mode != "gateway_http" && item.Mode != "gateway_tcp" {
			return fmt.Errorf("component %s endpoint mode is invalid", component.Name)
		}
		if item.ListenPort != nil && (*item.ListenPort < 1 || *item.ListenPort > 65535) {
			return fmt.Errorf("component %s endpoint listen port is invalid", component.Name)
		}
		if _, exists := endpoints[item.Name]; exists {
			return fmt.Errorf("component %s has duplicate endpoint %s", component.Name, item.Name)
		}
		endpoints[item.Name] = struct{}{}
	}
	mountTargets := make(map[string]struct{}, len(component.Mounts))
	for _, item := range component.Mounts {
		if err := validateComponentMount(item); err != nil {
			return fmt.Errorf("component %s mount: %w", component.Name, err)
		}
		if _, exists := mountTargets[item.Target]; exists {
			return fmt.Errorf("component %s has duplicate mount target %s", component.Name, item.Target)
		}
		mountTargets[item.Target] = struct{}{}
	}
	if err := validateComponentHealthcheck(component.Name, component.Healthcheck); err != nil {
		return err
	}
	if err := validateComponentDevices(component.Name, component.Devices); err != nil {
		return err
	}
	return nil
}

func validateComponentDevices(component string, devices []model.VersionComponentDeviceRequest) error {
	for _, device := range devices {
		if device.Driver == "" || strings.TrimSpace(device.Driver) != device.Driver || strings.ContainsAny(device.Driver, " \t\r\n") {
			return fmt.Errorf("component %s device driver is invalid", component)
		}
		if device.Count != "all" {
			count, err := strconv.ParseUint(device.Count, 10, 64)
			if err != nil || count == 0 || strconv.FormatUint(count, 10) != device.Count {
				return fmt.Errorf("component %s device count must be all or a positive integer", component)
			}
		}
		if len(device.Capabilities) == 0 {
			return fmt.Errorf("component %s device capabilities are required", component)
		}
		capabilities := make(map[string]struct{}, len(device.Capabilities))
		for _, capability := range device.Capabilities {
			if capability == "" || strings.TrimSpace(capability) != capability || strings.ContainsAny(capability, " \t\r\n") {
				return fmt.Errorf("component %s device capability is invalid", component)
			}
			if _, exists := capabilities[capability]; exists {
				return fmt.Errorf("component %s has duplicate device capability %s", component, capability)
			}
			capabilities[capability] = struct{}{}
		}
		if device.Driver == "nvidia" {
			if _, exists := capabilities["gpu"]; !exists {
				return fmt.Errorf("component %s nvidia device requires gpu capability", component)
			}
		}
	}
	return nil
}

func validateComponentMount(mount model.VersionComponentMount) error {
	if mount.Source == "" || mount.Target == "" {
		return fmt.Errorf("source and target are required")
	}
	if mount.SourceIsHostPath {
		if mount.SourceType != "directory" && mount.SourceType != "file" {
			return fmt.Errorf("source_is_host_path is only allowed for directory or file mounts")
		}
		if !isAbsoluteMountSource(mount.Source) {
			return fmt.Errorf("host path source must be an absolute path")
		}
	} else if mount.SourceType == "directory" || mount.SourceType == "file" || mount.SourceType == "controlled_file" {
		if isAbsoluteMountSource(mount.Source) || strings.Contains(mount.Source, "\\") || hasParentDirectory(mount.Source) {
			return fmt.Errorf("source must be a relative path without parent segments")
		}
	}
	switch mount.SourceType {
	case "directory", "file":
		if mount.Content != "" || mount.Mode != "" || mount.IgnoreIfExists {
			return fmt.Errorf("content options are only allowed on controlled_file mounts")
		}
	case "named_volume":
		if strings.ContainsAny(mount.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
		if mount.SourceIsHostPath || mount.Content != "" || mount.Mode != "" || mount.IgnoreIfExists {
			return fmt.Errorf("named_volume does not support file source options")
		}
	case "controlled_file":
		if mount.SourceIsHostPath {
			return fmt.Errorf("controlled_file source must be platform-relative")
		}
		if len(mount.Content) > maxMountContent {
			return fmt.Errorf("controlled_file content exceeds %d bytes", maxMountContent)
		}
		if !validUnixFileMode(mount.Mode) {
			return fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
		}
	default:
		return fmt.Errorf("unsupported source_type %s", mount.SourceType)
	}
	return nil
}

func validUnixFileMode(mode string) bool {
	if len(mode) != 4 || mode[0] != '0' {
		return false
	}
	for _, value := range mode[1:] {
		if value < '0' || value > '7' {
			return false
		}
	}
	return true
}

func hasParentDirectory(source string) bool {
	for _, part := range strings.Split(source, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func isAbsoluteMountSource(source string) bool {
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, `\\`) || strings.HasPrefix(source, `//`) {
		return true
	}
	if len(source) < 2 || source[1] != ':' {
		return false
	}
	letter := source[0]
	return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
}

func validateComponentHealthcheck(component string, healthcheck *model.VersionComponentHealthcheck) error {
	if healthcheck == nil || healthcheck.Disabled {
		return nil
	}
	if healthcheck.TestMode != "CMD" && healthcheck.TestMode != "CMD-SHELL" {
		return fmt.Errorf("component %s healthcheck test_mode must be CMD or CMD-SHELL", component)
	}
	if strings.TrimSpace(healthcheck.Test) == "" {
		return fmt.Errorf("component %s healthcheck test is required", component)
	}
	if healthcheck.TestMode == "CMD" {
		test, err := commandline.Parse(healthcheck.Test)
		if err != nil || len(test) == 0 {
			return fmt.Errorf("component %s healthcheck test is not a valid command", component)
		}
	}
	if healthcheck.Retries != nil && *healthcheck.Retries < 0 {
		return fmt.Errorf("component %s healthcheck retries must be non-negative", component)
	}
	return nil
}
