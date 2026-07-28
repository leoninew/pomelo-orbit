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
	requirements := map[string]Requirement{}
	if versionEnv != nil && strings.TrimSpace(*versionEnv) != "" {
		var vars []envVar
		if err := json.Unmarshal([]byte(*versionEnv), &vars); err != nil {
			return nil, fmt.Errorf("env_json must be an array of key/value entries: %w", err)
		}
		for _, variable := range vars {
			addRequirement(requirements, variable.Value)
		}
	}
	for _, component := range components {
		for _, variable := range component.Env {
			addRequirement(requirements, variable.Value)
		}
	}
	return requirements, nil
}

func addRequirement(requirements map[string]Requirement, value string) {
	name, requirement, ok := parsePlaceholder(value)
	if !ok {
		return
	}
	existing, exists := requirements[name]
	if !exists || requirement.Required {
		requirements[name] = requirement
		return
	}
	if !existing.Required {
		requirements[name] = requirement
	}
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
