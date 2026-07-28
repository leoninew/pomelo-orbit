package gatewaysvc

import (
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	mountSourceDirectory   = "directory"
	mountSourceFile        = "file"
	mountSourceNamedVolume = "named_volume"
	mountSourceSpecial     = "special"
	specialDockerSock      = "docker.sock"
	contentModeSeed        = "seed"
	contentModeSync        = "sync"
	maxMountContent        = 256 * 1024
)

func validateMountSpec(mount model.VersionComponentMount) error {
	if mount.SourceType == "" || mount.Source == "" || mount.Target == "" || !strings.HasPrefix(mount.Target, "/") {
		return fmt.Errorf("source_type, source and absolute target are required")
	}
	switch mount.SourceType {
	case mountSourceDirectory, mountSourceFile:
		if isAbsoluteMountSource(mount.Source) || hasParentMountSegment(mount.Source) {
			return fmt.Errorf("source must be a relative path without parent directory segments")
		}
		if mount.SourceType == mountSourceFile {
			if mount.ContentMode != contentModeSeed && mount.ContentMode != contentModeSync {
				return fmt.Errorf("file mount content_mode must be seed or sync")
			}
			if len(mount.Content) > maxMountContent {
				return fmt.Errorf("content exceeds %d bytes", maxMountContent)
			}
		} else if mount.Content != "" || mount.ContentMode != "" {
			return fmt.Errorf("content is only allowed on file mounts")
		}
	case mountSourceNamedVolume:
		if strings.ContainsAny(mount.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
	case mountSourceSpecial:
		if mount.Source != specialDockerSock || !mount.ReadOnly {
			return fmt.Errorf("only read-only docker.sock is supported as a special mount")
		}
	default:
		return fmt.Errorf("unsupported source_type %q", mount.SourceType)
	}
	return nil
}

func isAbsoluteMountSource(source string) bool {
	if filepath.IsAbs(source) || strings.HasPrefix(source, "/") || strings.HasPrefix(source, `\\`) || strings.HasPrefix(source, `//`) {
		return true
	}
	if len(source) >= 2 && source[1] == ':' {
		letter := source[0]
		return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
	}
	return false
}

func hasParentMountSegment(source string) bool {
	for _, segment := range strings.Split(source, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func validateVersionComponents(components []model.VersionComponent) error {
	names := make(map[string]struct{}, len(components))
	for _, component := range components {
		if component.Name == "" || component.Image == "" {
			return fmt.Errorf("component name and image are required")
		}
		if _, exists := names[component.Name]; exists {
			return fmt.Errorf("duplicate component name %s", component.Name)
		}
		for _, mount := range component.Mounts {
			if err := validateMountSpec(mount); err != nil {
				return fmt.Errorf("component %s mount: %w", component.Name, err)
			}
		}
		names[component.Name] = struct{}{}
	}
	for _, component := range components {
		for _, dependency := range component.Dependencies {
			if dependency.Name == component.Name {
				return fmt.Errorf("component %s depends on itself", component.Name)
			}
			if _, exists := names[dependency.Name]; !exists {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, dependency.Name)
			}
		}
	}
	return nil
}
