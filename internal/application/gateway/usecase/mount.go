package gatewaysvc

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	mountSourceLogical = "logical"
	mountSourceVolume  = "volume"
	mountSourceSpecial = "special"
	specialDockerSock  = "docker.sock"
	contentModeSeed    = "seed"
	contentModeSync    = "sync"
	maxMountContent    = 256 * 1024
)

type mountSpec struct {
	SourceType  string `json:"source_type"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	ReadOnly    bool   `json:"read_only"`
	Content     string `json:"content,omitempty"`
	ContentMode string `json:"content_mode,omitempty"`
}

func parseMountSpecs(raw *string) ([]mountSpec, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return nil, fmt.Errorf("mounts must be a JSON array of objects: %w", err)
	}
	mounts := make([]mountSpec, 0, len(items))
	for index, item := range items {
		sourceType, _ := item["source_type"].(string)
		source, _ := item["source"].(string)
		target, _ := item["target"].(string)
		readOnly := false
		if strings.TrimSpace(sourceType) == mountSourceSpecial && strings.TrimSpace(source) == specialDockerSock {
			readOnly = true
		}
		if value, ok := item["read_only"].(bool); ok {
			readOnly = value
		}
		content, _ := item["content"].(string)
		contentMode, _ := item["content_mode"].(string)
		mount := mountSpec{
			SourceType: strings.TrimSpace(sourceType), Source: strings.TrimSpace(source), Target: strings.TrimSpace(target),
			ReadOnly: readOnly, Content: content, ContentMode: strings.TrimSpace(strings.ToLower(contentMode)),
		}
		if err := validateMountSpec(mount); err != nil {
			return nil, fmt.Errorf("mounts[%d]: %w", index, err)
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
}

func validateMountSpec(mount mountSpec) error {
	if mount.SourceType == "" || mount.Source == "" || mount.Target == "" {
		return fmt.Errorf("source_type, source and target are required")
	}
	if !strings.HasPrefix(mount.Target, "/") {
		return fmt.Errorf("target must be an absolute container path")
	}
	switch mount.SourceType {
	case mountSourceLogical:
		if isAbsoluteMountSource(mount.Source) || strings.Contains(mount.Source, "..") {
			return fmt.Errorf("logical source must be a relative path without parent directory segments")
		}
	case mountSourceVolume:
		if strings.ContainsAny(mount.Source, `/\`) {
			return fmt.Errorf("volume source must be a volume name")
		}
	case mountSourceSpecial:
		if mount.Source != specialDockerSock {
			return fmt.Errorf("unsupported special mount %q", mount.Source)
		}
	default:
		return fmt.Errorf("unsupported source_type %q", mount.SourceType)
	}
	mode := mount.ContentMode
	if mode == "" {
		mode = contentModeSeed
	}
	if mount.Content != "" || mount.ContentMode != "" {
		if mount.SourceType != mountSourceLogical {
			return fmt.Errorf("content is only allowed on logical file mounts")
		}
		if !isFileMountSource(mount.Source, mount.Target) {
			return fmt.Errorf("content is only allowed on file mounts (source/target with file suffix)")
		}
		if mode != contentModeSeed && mode != contentModeSync {
			return fmt.Errorf("content_mode must be seed or sync")
		}
		if len(mount.Content) > maxMountContent {
			return fmt.Errorf("content exceeds %d bytes", maxMountContent)
		}
	}
	return nil
}

func isAbsoluteMountSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if filepath.IsAbs(source) || strings.HasPrefix(source, "/") || strings.HasPrefix(source, `\\`) || strings.HasPrefix(source, `//`) {
		return true
	}
	if len(source) >= 2 && source[1] == ':' {
		letter := source[0]
		return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
	}
	return false
}

func isFileMountSource(source string, target string) bool {
	for _, path := range []string{source, target} {
		lower := strings.ToLower(path)
		for _, suffix := range []string{".json", ".yml", ".yaml", ".toml", ".pem", ".key", ".crt", ".conf", ".cfg", ".txt", ".env"} {
			if strings.HasSuffix(lower, suffix) {
				return true
			}
		}
	}
	return false
}

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
		names[name] = struct{}{}
		if _, err := parseMountSpecs(component.MountsJSON); err != nil {
			return fmt.Errorf("component %s mounts_json: %w", name, err)
		}
		if err := validateEnvJSON(component.EnvJSON); err != nil {
			return fmt.Errorf("component %s env_json: %w", name, err)
		}
	}
	for _, component := range components {
		depends, err := parseStringSliceJSON(component.DependsOnJSON)
		if err != nil {
			return fmt.Errorf("component %s depends_on_json: %w", component.Name, err)
		}
		for _, dependency := range depends {
			if _, exists := names[dependency]; !exists {
				return fmt.Errorf("component %s depends on missing component %s", component.Name, dependency)
			}
		}
	}
	return nil
}

func validateEnvJSON(raw *string) error {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	if strings.HasPrefix(strings.TrimSpace(*raw), "{") {
		return fmt.Errorf("env_json must be a JSON array of {key,value}, not an object")
	}
	var items []struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(*raw), &items); err != nil {
		return fmt.Errorf("env must be a JSON array of objects: %w", err)
	}
	for index, item := range items {
		if strings.TrimSpace(item.Key) == "" {
			return fmt.Errorf("env[%d].key is required", index)
		}
	}
	return nil
}

func parseStringSliceJSON(raw *string) ([]string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(*raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}
