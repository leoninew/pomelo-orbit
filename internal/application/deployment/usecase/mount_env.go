package deploymentsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	mountSourceDirectory   = "directory"
	mountSourceFile        = "file"
	mountSourceNamedVolume = "named_volume"
	mountSourceSpecial     = "special"

	specialDockerSock = "docker.sock"

	contentModeSeed = "seed"
	contentModeSync = "sync"

	maxMountContentBytes = 256 * 1024
)

var (
	envPlaceholderRequired = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
	envPlaceholderDefault  = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}$`)
)

type MountSpec = model.VersionComponentMount

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResolvedMount struct {
	Compose     string
	HostSource  string
	IsFile      bool
	SourceType  string
	Content     string
	ContentMode string
}

func validateMountSpec(m MountSpec) error {
	if m.SourceType == "" || m.Source == "" || m.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if !strings.HasPrefix(m.Target, "/") {
		return fmt.Errorf("target must be an absolute container path")
	}
	switch m.SourceType {
	case mountSourceDirectory, mountSourceFile:
		if isAbsoluteMountSource(m.Source) || hasParentMountSegment(m.Source) {
			return fmt.Errorf("source must be a relative path without parent directory segments")
		}
		if m.SourceType == mountSourceFile {
			if m.ContentMode != contentModeSeed && m.ContentMode != contentModeSync {
				return fmt.Errorf("file mount content_mode must be seed or sync")
			}
			if len(m.Content) > maxMountContentBytes {
				return fmt.Errorf("content exceeds %d bytes", maxMountContentBytes)
			}
		} else if m.Content != "" || m.ContentMode != "" {
			return fmt.Errorf("content is only allowed on file mounts")
		}
	case mountSourceNamedVolume:
		if strings.ContainsAny(m.Source, `/\\`) {
			return fmt.Errorf("named_volume source must be a volume name")
		}
		if m.Content != "" || m.ContentMode != "" {
			return fmt.Errorf("content is only allowed on file mounts")
		}
	case mountSourceSpecial:
		if m.Source != specialDockerSock || !m.ReadOnly {
			return fmt.Errorf("only read-only docker.sock is supported as a special mount")
		}
		if m.Content != "" || m.ContentMode != "" {
			return fmt.Errorf("content is only allowed on file mounts")
		}
	default:
		return fmt.Errorf("unsupported source_type %q", m.SourceType)
	}
	return nil
}

func parseEnvVars(raw *string) ([]EnvVar, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(*raw, "{") {
		return nil, fmt.Errorf("env_json must be a JSON array of {key,value}, not an object")
	}
	var vars []EnvVar
	if err := json.Unmarshal([]byte(*raw), &vars); err != nil {
		return nil, fmt.Errorf("env must be a JSON array of objects: %w", err)
	}
	for i, item := range vars {
		if item.Key == "" {
			return nil, fmt.Errorf("env[%d].key is required", i)
		}
	}
	return vars, nil
}

type placeholderNeed struct {
	Required   bool
	Default    string
	HasDefault bool
}

func parsePlaceholder(value string) (string, placeholderNeed, bool) {
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

func resolveMountSpecs(mounts []MountSpec, physicalServiceDir string) ([]ResolvedMount, error) {
	out := make([]ResolvedMount, 0, len(mounts))
	for _, mount := range mounts {
		if err := validateMountSpec(mount); err != nil {
			return nil, err
		}
		item := ResolvedMount{SourceType: mount.SourceType, Content: mount.Content, ContentMode: mount.ContentMode}
		switch mount.SourceType {
		case mountSourceDirectory, mountSourceFile:
			if physicalServiceDir == "" {
				return nil, fmt.Errorf("physical service dir required for %s mount %s", mount.SourceType, mount.Source)
			}
			item.HostSource = filepath.ToSlash(filepath.Join(physicalServiceDir, filepath.FromSlash(mount.Source)))
			item.IsFile = mount.SourceType == mountSourceFile
			item.Compose = item.HostSource + ":" + mount.Target
		case mountSourceNamedVolume:
			item.Compose = mount.Source + ":" + mount.Target
		case mountSourceSpecial:
			item.Compose = "/var/run/docker.sock:" + mount.Target
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
		if (item.SourceType != mountSourceDirectory && item.SourceType != mountSourceFile) || item.HostSource == "" {
			continue
		}
		host := filepath.FromSlash(item.HostSource)
		if item.IsFile {
			if err := materializeFile(host, item.Content, item.ContentMode); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(host, 0o755); err != nil {
			return fmt.Errorf("create mount directory %s: %w", host, err)
		}
	}
	return nil
}

func materializeFile(path string, content string, mode string) error {
	if mode == "" {
		return fmt.Errorf("content_mode is required for file mount %s", path)
	}
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return fmt.Errorf("mount source %s exists as directory but must be a file", path)
		}
		if mode == contentModeSync {
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				return fmt.Errorf("sync mount file %s: %w", path, err)
			}
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mount source %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent for mount file %s: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("create mount file %s: %w", path, err)
	}
	return nil
}
