package model

import (
	"fmt"
	"strings"
)

func ValidateSharedMount(sourceType, source string, shared bool) error {
	if shared && (sourceType != "directory" || !isAbsoluteMountSource(source)) {
		return fmt.Errorf("shared is only allowed for absolute directory sources")
	}
	return nil
}

// ValidateMountSource validates a mount source independently from its target
// and type-specific content options.
func ValidateMountSource(sourceType, source string, sourceIsHostPath bool) error {
	if source == "" {
		return fmt.Errorf("source is required")
	}
	if sourceIsHostPath {
		if sourceType != "directory" && sourceType != "file" {
			return fmt.Errorf("source_is_host_path is only allowed for directory or file mounts")
		}
		if !isAbsoluteMountSource(source) {
			return fmt.Errorf("host path source must be an absolute path")
		}
		return nil
	}
	switch sourceType {
	case "directory", "file", "controlled_file":
		if !isExplicitMountSource(source) {
			return fmt.Errorf("source must be an absolute path or a relative path starting with ./ without parent directory segments")
		}
	case "named_volume":
		if strings.ContainsAny(source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
	default:
		return fmt.Errorf("unsupported source_type %q", sourceType)
	}
	return nil
}

func MergeServiceComponentMount(declaration VersionComponentMount, overlay ServiceComponentMount) (*VersionComponentMount, error) {
	if overlay.Target != declaration.Target {
		return nil, fmt.Errorf("mount target does not match declaration")
	}
	hasValues := overlay.Source != nil || overlay.SourceIsHostPath != nil || overlay.Content != nil || overlay.Mode != nil || overlay.IgnoreIfExists != nil || overlay.ReadOnly != nil
	switch overlay.State {
	case ServiceComponentOverlayDeleted:
		if hasValues {
			return nil, fmt.Errorf("deleted mount cannot have override values")
		}
		return nil, nil
	case ServiceComponentOverlayOverride:
		if !hasValues {
			return nil, fmt.Errorf("mount override requires at least one value")
		}
	default:
		return nil, fmt.Errorf("invalid mount overlay state")
	}
	if declaration.SourceType != "controlled_file" && (overlay.Content != nil || overlay.Mode != nil || overlay.IgnoreIfExists != nil) {
		return nil, fmt.Errorf("content options are only allowed on controlled_file mounts")
	}
	if declaration.SourceType == "controlled_file" && (declaration.ReadOnly || overlay.ReadOnly != nil) {
		return nil, fmt.Errorf("controlled_file does not support read_only")
	}
	merged := declaration
	if overlay.Source != nil {
		merged.Source = *overlay.Source
	}
	if overlay.SourceIsHostPath != nil {
		merged.SourceIsHostPath = *overlay.SourceIsHostPath
	}
	if overlay.Content != nil {
		merged.Content = *overlay.Content
	}
	if overlay.Mode != nil {
		merged.Mode = *overlay.Mode
	}
	if overlay.IgnoreIfExists != nil {
		merged.IgnoreIfExists = *overlay.IgnoreIfExists
	}
	if overlay.ReadOnly != nil {
		merged.ReadOnly = *overlay.ReadOnly
	}
	if err := ValidateMountSource(merged.SourceType, merged.Source, merged.SourceIsHostPath); err != nil {
		return nil, err
	}
	if err := ValidateSharedMount(merged.SourceType, merged.Source, merged.Shared); err != nil {
		return nil, err
	}
	if merged.SourceType == "controlled_file" {
		if len(merged.Content) > 256*1024 {
			return nil, fmt.Errorf("controlled_file content exceeds %d bytes", 256*1024)
		}
		if len(merged.Mode) != 4 || merged.Mode[0] != '0' || strings.Trim(merged.Mode[1:], "01234567") != "" {
			return nil, fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
		}
	}
	return &merged, nil
}

func isExplicitMountSource(source string) bool {
	if isAbsoluteMountSource(source) {
		return true
	}
	return strings.HasPrefix(source, "./") && !strings.Contains(source, "\\") && !hasParentMountSegment(source)
}

func isAbsoluteMountSource(source string) bool {
	if strings.HasPrefix(source, "/") {
		return true
	}
	if len(source) >= 3 && source[1] == ':' && source[2] == '/' {
		letter := source[0]
		if (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z') {
			return true
		}
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
