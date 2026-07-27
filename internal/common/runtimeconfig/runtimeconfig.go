package runtimeconfig

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type envVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Requirement struct {
	Required   bool
	HasDefault bool
	Default    string
}

// Requirements returns the K/V names consumed by the Version environment fields.
func Requirements(versionEnv *string, components []model.VersionComponent) (map[string]Requirement, error) {
	items := make([]*string, 0, len(components)+1)
	items = append(items, versionEnv)
	for _, component := range components {
		items = append(items, component.EnvJSON)
	}
	requirements := map[string]Requirement{}
	for _, raw := range items {
		if raw == nil || strings.TrimSpace(*raw) == "" {
			continue
		}
		var vars []envVar
		if err := json.Unmarshal([]byte(*raw), &vars); err != nil {
			return nil, fmt.Errorf("env_json must be an array of key/value entries: %w", err)
		}
		for _, variable := range vars {
			name, requirement, ok := parsePlaceholder(variable.Value)
			if !ok {
				continue
			}
			existing, exists := requirements[name]
			if !exists || requirement.Required {
				requirements[name] = requirement
				continue
			}
			if existing.Required {
				continue
			}
			requirements[name] = requirement
		}
	}
	return requirements, nil
}

func Validate(values map[string]string, versionEnv *string, components []model.VersionComponent) (missing []string, extra []string, err error) {
	requirements, err := Requirements(versionEnv, components)
	if err != nil {
		return nil, nil, err
	}
	for key, requirement := range requirements {
		if requirement.Required {
			if _, exists := values[key]; !exists {
				missing = append(missing, key)
			}
		}
	}
	for key := range values {
		if _, used := requirements[key]; !used {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra, nil
}

func Resolve(values map[string]string, versionEnv *string, components []model.VersionComponent) (map[string]string, []string, error) {
	requirements, err := Requirements(versionEnv, components)
	if err != nil {
		return nil, nil, err
	}
	resolved := make(map[string]string, len(requirements))
	missing := make([]string, 0)
	for key, requirement := range requirements {
		if value, exists := values[key]; exists {
			resolved[key] = value
			continue
		}
		if requirement.HasDefault {
			resolved[key] = requirement.Default
			continue
		}
		if requirement.Required {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, nil, fmt.Errorf("missing runtime config keys: %s", strings.Join(missing, ", "))
	}
	extra := make([]string, 0)
	for key := range values {
		if _, used := requirements[key]; !used {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	return resolved, extra, nil
}

func parsePlaceholder(value string) (string, Requirement, bool) {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		body := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if name, defaultValue, ok := strings.Cut(body, ":-"); ok && name != "" {
			return name, Requirement{HasDefault: true, Default: defaultValue}, true
		}
		if body != "" {
			return body, Requirement{Required: true}, true
		}
	}
	return "", Requirement{}, false
}
