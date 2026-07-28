package applicationsvc

import (
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

func validateComponentRuntimeFields(component model.VersionComponent) error {
	if component.RestartPolicy != nil && *component.RestartPolicy != "no" && *component.RestartPolicy != "unless-stopped" {
		return fmt.Errorf("component %s restart_policy must be no or unless-stopped", component.Name)
	}
	if err := validateTmpfs(component.Name, component.Tmpfs); err != nil {
		return err
	}
	if err := validateUlimits(component.Name, component.Ulimits); err != nil {
		return err
	}
	return nil
}

func validateTmpfs(component string, entries []model.VersionComponentTmpfs) error {
	if len(entries) > 8 {
		return fmt.Errorf("component %s tmpfs supports at most 8 entries", component)
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

func validateUlimits(component string, entries []model.VersionComponentUlimit) error {
	if len(entries) > 8 {
		return fmt.Errorf("component %s ulimits supports at most 8 entries", component)
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
