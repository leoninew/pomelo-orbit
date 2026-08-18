package deploymentsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	mountSourceDirectory      = "directory"
	mountSourceFile           = "file"
	mountSourceNamedVolume    = "named_volume"
	mountSourceControlledFile = "controlled_file"

	maxMountContentBytes = 256 * 1024
)

var (
	envPlaceholderRequired        = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
	envPlaceholderDefault         = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}$`)
	envPlaceholderRequiredMessage = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):\?[^}]*\}$`)
)

type MountSpec = model.VersionComponentMount

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResolvedMount struct {
	Compose           string
	HostSource        string
	LogicalSource     string
	IsFile            bool
	ShouldMaterialize bool
	SourceType        string
	Content           string
	IgnoreIfExists    bool
	FileMode          os.FileMode
	NamedVolumeName   string
}

func validateMountSpec(m MountSpec) error {
	if m.SourceType == "" || m.Source == "" || m.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if m.SourceIsHostPath {
		if m.SourceType != mountSourceDirectory && m.SourceType != mountSourceFile {
			return fmt.Errorf("source_is_host_path is only allowed for directory or file mounts")
		}
		if !isAbsoluteMountSource(m.Source) {
			return fmt.Errorf("host path source must be an absolute path")
		}
	} else if m.SourceType == mountSourceDirectory || m.SourceType == mountSourceFile || m.SourceType == mountSourceControlledFile {
		if isAbsoluteMountSource(m.Source) || hasParentMountSegment(m.Source) {
			return fmt.Errorf("source must be a relative path without parent directory segments")
		}
	}
	switch m.SourceType {
	case mountSourceDirectory, mountSourceFile:
		if m.Content != "" || m.Mode != "" || m.IgnoreIfExists {
			return fmt.Errorf("content options are only allowed on controlled_file mounts")
		}
	case mountSourceNamedVolume:
		if strings.ContainsAny(m.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
		if m.SourceIsHostPath || m.Content != "" || m.Mode != "" || m.IgnoreIfExists {
			return fmt.Errorf("named_volume does not support file source options")
		}
	case mountSourceControlledFile:
		if m.SourceIsHostPath {
			return fmt.Errorf("controlled_file source must be platform-relative")
		}
		if len(m.Content) > maxMountContentBytes {
			return fmt.Errorf("content exceeds %d bytes", maxMountContentBytes)
		}
		if _, err := parseUnixFileMode(m.Mode); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported source_type %q", m.SourceType)
	}
	return nil
}

func parseUnixFileMode(mode string) (os.FileMode, error) {
	if len(mode) != 4 || mode[0] != '0' {
		return 0, fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
	}
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil || parsed > 0o777 {
		return 0, fmt.Errorf("controlled_file mode must be a four-digit Unix octal mode")
	}
	return os.FileMode(parsed), nil
}

type placeholderNeed struct {
	Required   bool
	Default    string
	HasDefault bool
}

func parsePlaceholder(value string) (string, placeholderNeed, bool) {
	if m := envPlaceholderRequiredMessage.FindStringSubmatch(value); len(m) == 2 {
		return m[1], placeholderNeed{Required: true}, true
	}
	if m := envPlaceholderRequired.FindStringSubmatch(value); len(m) == 2 {
		return m[1], placeholderNeed{Required: true}, true
	}
	if m := envPlaceholderDefault.FindStringSubmatch(value); len(m) == 3 {
		return m[1], placeholderNeed{HasDefault: true, Default: m[2]}, true
	}
	return "", placeholderNeed{}, false
}

func applyEnvPlaceholders(vars []EnvVar, runtime map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	out := make(map[string]string, len(vars))
	for _, item := range vars {
		value := item.Value
		if name, _, ok := parsePlaceholder(value); ok && runtime != nil {
			if resolved, exists := runtime[name]; exists {
				value = resolved
			}
		}
		out[item.Key] = value
	}
	return out
}

func componentEnv(component []model.VersionComponentEnv, runtime map[string]string) map[string]string {
	vars := make([]EnvVar, 0, len(component))
	for _, item := range component {
		vars = append(vars, EnvVar{Key: item.Key, Value: item.Value})
	}
	return applyEnvPlaceholders(vars, runtime)
}

func isAbsoluteMountSource(source string) bool {
	if source == "" {
		return false
	}
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

func resolveMountSpecs(mounts []MountSpec, composeMountSourceDir string) ([]ResolvedMount, error) {
	return resolveMountSpecsForPaths(mounts, composeMountSourceDir, composeMountSourceDir)
}

func resolveMountSpecsForPaths(mounts []MountSpec, logicalServiceDir string, composeMountSourceDir string) ([]ResolvedMount, error) {
	out := make([]ResolvedMount, 0, len(mounts))
	for _, mount := range mounts {
		if err := validateMountSpec(mount); err != nil {
			return nil, err
		}
		item := ResolvedMount{SourceType: mount.SourceType, Content: mount.Content, IgnoreIfExists: mount.IgnoreIfExists}
		if mount.SourceType == mountSourceControlledFile {
			mode, err := parseUnixFileMode(mount.Mode)
			if err != nil {
				return nil, err
			}
			item.FileMode = mode
		}
		switch mount.SourceType {
		case mountSourceDirectory, mountSourceFile, mountSourceControlledFile:
			if mount.SourceIsHostPath {
				item.HostSource = mount.Source
				item.IsFile = mount.SourceType == mountSourceFile
				item.Compose = item.HostSource + ":" + mount.Target
				break
			}
			if logicalServiceDir == "" {
				return nil, fmt.Errorf("logical service dir required for %s mount %s", mount.SourceType, mount.Source)
			}
			item.LogicalSource = filepath.Join(logicalServiceDir, filepath.FromSlash(mount.Source))
			item.IsFile = mount.SourceType == mountSourceFile || mount.SourceType == mountSourceControlledFile
			item.ShouldMaterialize = mount.SourceType == mountSourceDirectory || mount.SourceType == mountSourceControlledFile
			if composeMountSourceDir == "" {
				item.Compose = "./" + mount.Source + ":" + mount.Target
			} else {
				item.HostSource = filepath.ToSlash(filepath.Join(composeMountSourceDir, filepath.FromSlash(mount.Source)))
				item.Compose = item.HostSource + ":" + mount.Target
			}
		case mountSourceNamedVolume:
			item.Compose = mount.Source + ":" + mount.Target
			item.NamedVolumeName = mount.Source
		}
		if mount.ReadOnly {
			item.Compose += ":ro"
		}
		out = append(out, item)
	}
	return out, nil
}

func MaterializeLogicalMountSources(resolved []ResolvedMount) error {
	for _, item := range resolved {
		if !item.ShouldMaterialize {
			continue
		}
		if item.LogicalSource == "" {
			return fmt.Errorf("logical mount source is required for materialization")
		}
		logicalSource := filepath.FromSlash(item.LogicalSource)
		if item.IsFile {
			if err := materializeFile(logicalSource, item.Content, item.IgnoreIfExists, item.FileMode); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(logicalSource, 0o755); err != nil {
			return fmt.Errorf("create mount directory %s: %w", logicalSource, err)
		}
	}
	return nil
}

func materializeFile(path string, content string, ignoreIfExists bool, mode os.FileMode) error {
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return fmt.Errorf("mount source %s exists as directory but must be a file", path)
		}
		if !ignoreIfExists {
			if err := os.WriteFile(path, []byte(content), mode); err != nil {
				return fmt.Errorf("write controlled mount file %s: %w", path, err)
			}
		}
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("set controlled mount file mode %s: %w", path, err)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mount source %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent for mount file %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("create mount file %s: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set controlled mount file mode %s: %w", path, err)
	}
	return nil
}
