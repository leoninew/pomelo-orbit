package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

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
		if isAbsoluteMountSource(source) || strings.Contains(source, "\\") || hasParentMountSegment(source) {
			return fmt.Errorf("source must be a relative path without parent directory segments")
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

func isAbsoluteMountSource(source string) bool {
	if filepath.IsAbs(source) || strings.HasPrefix(source, "/") {
		return true
	}
	if len(source) >= 2 && source[1] == ':' {
		letter := source[0]
		if (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z') {
			return true
		}
	}
	return strings.HasPrefix(source, `\\`) || strings.HasPrefix(source, `//`)
}

func hasParentMountSegment(source string) bool {
	for _, segment := range strings.Split(source, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}
