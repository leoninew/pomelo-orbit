package gatewaysvc

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	mountSourceDirectory      = "directory"
	mountSourceFile           = "file"
	mountSourceNamedVolume    = "named_volume"
	mountSourceControlledFile = "controlled_file"
	maxMountContent           = 256 * 1024
)

func validateMountSpec(mount model.VersionComponentMount) error {
	if mount.SourceType == "" || mount.Source == "" || mount.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if mount.SourceIsHostPath {
		if mount.SourceType != mountSourceDirectory && mount.SourceType != mountSourceFile {
			return fmt.Errorf("source_is_host_path is only allowed for directory or file mounts")
		}
		if !isAbsoluteMountSource(mount.Source) {
			return fmt.Errorf("host path source must be an absolute path")
		}
	} else if mount.SourceType == mountSourceDirectory || mount.SourceType == mountSourceFile || mount.SourceType == mountSourceControlledFile {
		if isAbsoluteMountSource(mount.Source) || hasParentMountSegment(mount.Source) {
			return fmt.Errorf("source must be a relative path without parent directory segments")
		}
	}
	switch mount.SourceType {
	case mountSourceDirectory, mountSourceFile:
		if mount.Content != "" || mount.Mode != "" || mount.IgnoreIfExists {
			return fmt.Errorf("content options are only allowed on controlled_file mounts")
		}
	case mountSourceNamedVolume:
		if strings.ContainsAny(mount.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
		if mount.SourceIsHostPath || mount.Content != "" || mount.Mode != "" || mount.IgnoreIfExists {
			return fmt.Errorf("named_volume does not support file source options")
		}
	case mountSourceControlledFile:
		if mount.SourceIsHostPath {
			return fmt.Errorf("controlled_file source must be platform-relative")
		}
		if len(mount.Content) > maxMountContent {
			return fmt.Errorf("content exceeds %d bytes", maxMountContent)
		}
		if !validUnixFileMode(mount.Mode) {
			return fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
		}
	default:
		return fmt.Errorf("unsupported source_type %q", mount.SourceType)
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
