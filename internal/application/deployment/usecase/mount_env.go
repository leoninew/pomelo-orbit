package deploymentsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	mountSourceLogical = "logical"
	mountSourceVolume  = "volume"
	mountSourceSpecial = "special"

	specialDockerSock = "docker.sock"

	contentModeSeed = "seed"
	contentModeSync = "sync"

	// maxMountContentBytes is Spec D1 limit for a single mount content field.
	maxMountContentBytes = 256 * 1024
)

var (
	envPlaceholderRequired = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
	envPlaceholderDefault  = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}$`)
	fileMountSuffixes      = []string{
		".json", ".yml", ".yaml", ".toml", ".pem", ".key", ".crt", ".conf", ".cfg", ".txt", ".env",
	}
)

// MountSpec is the structured VersionComponent mount schema (Spec D1).
type MountSpec struct {
	SourceType  string `json:"source_type"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	ReadOnly    bool   `json:"read_only"`
	Content     string `json:"content,omitempty"`
	ContentMode string `json:"content_mode,omitempty"`
}

// EnvVar is a Version / Component environment entry (Spec D2).
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ResolvedMount is a compose-ready volume entry plus materialize metadata.
type ResolvedMount struct {
	Compose     string
	HostSource  string // set for logical mounts after physical resolution
	IsFile      bool
	SourceType  string
	Content     string
	ContentMode string
}

func parseMountSpecs(raw *string) ([]MountSpec, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var rawItems []map[string]any
	if err := json.Unmarshal([]byte(*raw), &rawItems); err != nil {
		return nil, fmt.Errorf("mounts must be a JSON array of objects: %w", err)
	}
	out := make([]MountSpec, 0, len(rawItems))
	for i, item := range rawItems {
		sourceType, _ := item["source_type"].(string)
		source, _ := item["source"].(string)
		target, _ := item["target"].(string)
		ro := false
		if strings.TrimSpace(sourceType) == mountSourceSpecial && strings.TrimSpace(source) == specialDockerSock {
			ro = true // Spec default for docker.sock
		}
		if v, ok := item["read_only"].(bool); ok {
			ro = v
		}
		content, _ := item["content"].(string)
		contentMode, _ := item["content_mode"].(string)
		spec := MountSpec{
			SourceType:  strings.TrimSpace(sourceType),
			Source:      strings.TrimSpace(source),
			Target:      strings.TrimSpace(target),
			ReadOnly:    ro,
			Content:     content,
			ContentMode: strings.TrimSpace(strings.ToLower(contentMode)),
		}
		if err := validateMountSpec(spec); err != nil {
			return nil, fmt.Errorf("mounts[%d]: %w", i, err)
		}
		out = append(out, spec)
	}
	return out, nil
}

func validateMountSpec(m MountSpec) error {
	if m.SourceType == "" || m.Source == "" || m.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if !strings.HasPrefix(m.Target, "/") {
		return fmt.Errorf("target must be an absolute container path")
	}
	switch m.SourceType {
	case mountSourceLogical:
		// filepath.IsAbs is OS-dependent: on Windows, "/var/run/..." is not absolute.
		// Logical sources must stay relative on every host GOOS, so also reject Unix roots.
		if isAbsoluteMountSource(m.Source) || strings.Contains(m.Source, "..") {
			return fmt.Errorf("logical source must be a relative path without parent directory segments")
		}
	case mountSourceVolume:
		if strings.ContainsAny(m.Source, `/\`) {
			return fmt.Errorf("volume source must be a volume name")
		}
	case mountSourceSpecial:
		if m.Source != specialDockerSock {
			return fmt.Errorf("unsupported special mount %q", m.Source)
		}
	default:
		return fmt.Errorf("unsupported source_type %q", m.SourceType)
	}
	hasContent := m.Content != ""
	mode := m.ContentMode
	if mode == "" {
		mode = contentModeSeed
	}
	if hasContent || m.ContentMode != "" {
		if m.SourceType != mountSourceLogical {
			return fmt.Errorf("content is only allowed on logical file mounts")
		}
		if !isFileMountSource(m.Source, m.Target) {
			return fmt.Errorf("content is only allowed on file mounts (source/target with file suffix)")
		}
		if mode != contentModeSeed && mode != contentModeSync {
			return fmt.Errorf("content_mode must be seed or sync")
		}
		if len(m.Content) > maxMountContentBytes {
			return fmt.Errorf("content exceeds %d bytes", maxMountContentBytes)
		}
	}
	return nil
}

func parseEnvVars(raw *string) ([]EnvVar, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if strings.HasPrefix(trimmed, "{") {
		return nil, fmt.Errorf("env_json must be a JSON array of {key,value}, not an object")
	}
	var vars []EnvVar
	if err := json.Unmarshal([]byte(trimmed), &vars); err != nil {
		return nil, fmt.Errorf("env must be a JSON array of objects: %w", err)
	}
	for i, item := range vars {
		if strings.TrimSpace(item.Key) == "" {
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

func collectPlaceholders(varsList ...[]EnvVar) map[string]placeholderNeed {
	out := map[string]placeholderNeed{}
	for _, vars := range varsList {
		for _, item := range vars {
			name, need, ok := parsePlaceholder(item.Value)
			if !ok {
				continue
			}
			existing, exists := out[name]
			if !exists {
				out[name] = need
				continue
			}
			if need.Required {
				existing.Required = true
				existing.HasDefault = false
				existing.Default = ""
			}
			out[name] = existing
		}
	}
	return out
}

func parsePlaceholder(value string) (string, placeholderNeed, bool) {
	value = strings.TrimSpace(value)
	if m := envPlaceholderRequired.FindStringSubmatch(value); len(m) == 2 {
		return m[1], placeholderNeed{Required: true}, true
	}
	if m := envPlaceholderDefault.FindStringSubmatch(value); len(m) == 3 {
		return m[1], placeholderNeed{HasDefault: true, Default: m[2]}, true
	}
	return "", placeholderNeed{}, false
}

func resolveRuntimeConfig(placeholders map[string]placeholderNeed, provided map[string]string) (map[string]string, []string) {
	resolved := map[string]string{}
	var missing []string
	for name, need := range placeholders {
		if provided != nil {
			if v, ok := provided[name]; ok {
				resolved[name] = v
				continue
			}
		}
		if need.HasDefault {
			resolved[name] = need.Default
			continue
		}
		if need.Required {
			missing = append(missing, name)
		}
	}
	return resolved, missing
}

func applyEnvPlaceholders(vars []EnvVar, runtime map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	out := make(map[string]string, len(vars))
	for _, item := range vars {
		key := strings.TrimSpace(item.Key)
		value := item.Value
		if name, _, ok := parsePlaceholder(value); ok {
			if runtime != nil {
				if v, exists := runtime[name]; exists {
					value = v
				}
			}
		}
		out[key] = value
	}
	return out
}

func isFileMountSource(source string, target string) bool {
	for _, path := range []string{source, target} {
		lower := strings.ToLower(path)
		for _, suffix := range fileMountSuffixes {
			if strings.HasSuffix(lower, suffix) {
				return true
			}
		}
	}
	return false
}

// isAbsoluteMountSource rejects host absolute paths for logical sources, independent of GOOS.
func isAbsoluteMountSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if filepath.IsAbs(source) {
		return true
	}
	// Unix absolute path (must reject even when the API process runs on Windows).
	if strings.HasPrefix(source, "/") {
		return true
	}
	// Windows drive / UNC when validation runs on non-Windows.
	if len(source) >= 2 && source[1] == ':' {
		c := source[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	if strings.HasPrefix(source, `\\`) || strings.HasPrefix(source, `//`) {
		return true
	}
	return false
}

func resolveMountSpecs(mounts []MountSpec, physicalServiceDir string) ([]ResolvedMount, error) {
	if len(mounts) == 0 {
		return nil, nil
	}
	out := make([]ResolvedMount, 0, len(mounts))
	for _, m := range mounts {
		var compose string
		var host string
		isFile := false
		switch m.SourceType {
		case mountSourceLogical:
			if physicalServiceDir == "" {
				return nil, fmt.Errorf("physical service dir required for logical mount %s", m.Source)
			}
			host = filepath.ToSlash(filepath.Join(physicalServiceDir, filepath.FromSlash(m.Source)))
			compose = host + ":" + m.Target
			if m.ReadOnly {
				compose += ":ro"
			}
			isFile = isFileMountSource(m.Source, m.Target)
		case mountSourceVolume:
			compose = m.Source + ":" + m.Target
			if m.ReadOnly {
				compose += ":ro"
			}
		case mountSourceSpecial:
			if m.Source == specialDockerSock {
				compose = "/var/run/docker.sock:" + m.Target
				if m.ReadOnly {
					compose += ":ro"
				}
			}
		default:
			return nil, fmt.Errorf("unsupported source_type %q", m.SourceType)
		}
		mode := strings.TrimSpace(strings.ToLower(m.ContentMode))
		if mode == "" {
			mode = contentModeSeed
		}
		out = append(out, ResolvedMount{
			Compose:     compose,
			HostSource:  host,
			IsFile:      isFile,
			SourceType:  m.SourceType,
			Content:     m.Content,
			ContentMode: mode,
		})
	}
	return out, nil
}

// MaterializeLogicalMountSources creates missing logical mount host sources (replaces init.sh prep).
// File mounts may carry Version content (seed: write if missing; sync: overwrite each deploy).
func MaterializeLogicalMountSources(resolved []ResolvedMount) error {
	for _, item := range resolved {
		if item.SourceType != mountSourceLogical || item.HostSource == "" {
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
		mode = contentModeSeed
	}
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return fmt.Errorf("mount source %s exists as directory but must be a file", path)
		}
		if mode == contentModeSync && content != "" {
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				return fmt.Errorf("sync mount file %s: %w", path, err)
			}
		}
		// seed (or sync with empty content): never overwrite existing payload (acme.json safe).
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mount source %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent for mount file %s: %w", path, err)
	}
	payload := []byte(content)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("create mount file %s: %w", path, err)
	}
	return nil
}

func resolveDeployRuntimeConfig(version model.Version, components []model.VersionComponent, provided map[string]string) (map[string]string, error) {
	versionEnv, err := parseEnvVars(version.EnvJSON)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, "version env_json: "+err.Error())
	}
	var lists [][]EnvVar
	lists = append(lists, versionEnv)
	for _, component := range components {
		componentEnv, err := parseEnvVars(component.EnvJSON)
		if err != nil {
			return nil, apperror.New(apperror.KindValidation, "component "+component.Name+" env_json: "+err.Error())
		}
		lists = append(lists, componentEnv)
	}
	placeholders := collectPlaceholders(lists...)
	resolved, missing := resolveRuntimeConfig(placeholders, provided)
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, apperror.New(apperror.KindValidation, "missing runtime_config keys: "+strings.Join(missing, ", "))
	}
	return resolved, nil
}
