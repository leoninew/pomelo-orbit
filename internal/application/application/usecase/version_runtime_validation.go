package applicationsvc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	minTmpfsSizeBytes = 1 << 20
	maxTmpfsSizeBytes = 8 << 30
)

var tmpfsModePattern = regexp.MustCompile(`^[0-7]{3,4}$`)

type tmpfsSpec struct {
	Target    string `json:"target"`
	SizeBytes int64  `json:"size_bytes"`
	Mode      string `json:"mode"`
}

type ulimitSpec struct {
	Name string `json:"name"`
	Soft int64  `json:"soft"`
	Hard int64  `json:"hard"`
}

func validateComponentRuntimeFields(component model.VersionComponent) error {
	if component.RestartPolicy != nil && *component.RestartPolicy != "no" && *component.RestartPolicy != "unless-stopped" {
		return fmt.Errorf("component %s restart_policy must be no or unless-stopped", component.Name)
	}
	if err := validateTmpfs(component.Name, component.TmpfsJSON); err != nil {
		return err
	}
	if err := validateUlimits(component.Name, component.UlimitsJSON); err != nil {
		return err
	}
	return nil
}

func validateTmpfs(component string, raw *string) error {
	if raw == nil || *raw == "" {
		return nil
	}
	var entries []tmpfsSpec
	if err := json.Unmarshal([]byte(*raw), &entries); err != nil {
		return fmt.Errorf("component %s tmpfs_json must be an array: %w", component, err)
	}
	if len(entries) > 8 {
		return fmt.Errorf("component %s tmpfs_json supports at most 8 entries", component)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !validTmpfsTarget(entry.Target) {
			return fmt.Errorf("component %s tmpfs target is invalid", component)
		}
		if entry.SizeBytes < minTmpfsSizeBytes || entry.SizeBytes > maxTmpfsSizeBytes {
			return fmt.Errorf("component %s tmpfs size_bytes must be between %d and %d", component, minTmpfsSizeBytes, maxTmpfsSizeBytes)
		}
		if !tmpfsModePattern.MatchString(entry.Mode) {
			return fmt.Errorf("component %s tmpfs mode must be a 3 or 4 digit octal string", component)
		}
		if _, exists := seen[entry.Target]; exists {
			return fmt.Errorf("component %s has duplicate tmpfs target %s", component, entry.Target)
		}
		seen[entry.Target] = struct{}{}
	}
	return nil
}

func validTmpfsTarget(target string) bool {
	if !strings.HasPrefix(target, "/") || target == "/" {
		return false
	}
	for _, segment := range strings.Split(target, "/") {
		if segment == ".." {
			return false
		}
	}
	for _, blocked := range []string{"/proc", "/sys", "/dev"} {
		if target == blocked || strings.HasPrefix(target, blocked+"/") {
			return false
		}
	}
	return true
}

func validateUlimits(component string, raw *string) error {
	if raw == nil || *raw == "" {
		return nil
	}
	var entries []ulimitSpec
	if err := json.Unmarshal([]byte(*raw), &entries); err != nil {
		return fmt.Errorf("component %s ulimits_json must be an array: %w", component, err)
	}
	if len(entries) > 8 {
		return fmt.Errorf("component %s ulimits_json supports at most 8 entries", component)
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if entry.Name != "memlock" && entry.Name != "nofile" {
			return fmt.Errorf("component %s ulimit name must be memlock or nofile", component)
		}
		if _, exists := seen[entry.Name]; exists {
			return fmt.Errorf("component %s has duplicate ulimit %s", component, entry.Name)
		}
		seen[entry.Name] = struct{}{}
		if entry.Soft == -1 || entry.Hard == -1 {
			if entry.Name != "memlock" || entry.Soft != -1 || entry.Hard != -1 {
				return fmt.Errorf("component %s only memlock may use -1 for both soft and hard", component)
			}
			continue
		}
		if entry.Soft < 0 || entry.Hard < 0 || entry.Soft > entry.Hard {
			return fmt.Errorf("component %s ulimit %s requires 0 <= soft <= hard", component, entry.Name)
		}
	}
	return nil
}
