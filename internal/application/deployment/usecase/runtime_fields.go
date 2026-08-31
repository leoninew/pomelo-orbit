package deploymentsvc

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	minRuntimeTmpfsSizeBytes = 1 << 20
	maxRuntimeTmpfsSizeBytes = 8 << 30
)

var runtimeTmpfsModePattern = regexp.MustCompile(`^[0-7]{3,4}$`)

func applyComponentRuntimeFields(service map[string]any, component model.VersionComponent) error {
	if err := validateComponentRuntimeFields(component); err != nil {
		return err
	}
	if component.RestartPolicy != nil && *component.RestartPolicy != "no" {
		service["restart"] = *component.RestartPolicy
	}
	if len(component.Tmpfs) > 0 {
		tmpfs, _ := renderRuntimeTmpfs(component.Tmpfs)
		service["tmpfs"] = tmpfs
	}
	if len(component.Ulimits) > 0 {
		ulimits, _ := renderRuntimeUlimits(component.Ulimits)
		service["ulimits"] = ulimits
	}
	return nil
}

func validateComponentRuntimeFields(component model.VersionComponent) error {
	if component.RestartPolicy != nil {
		switch *component.RestartPolicy {
		case "no":
		case "on-failure":
		case "always":
		case "unless-stopped":
		default:
			return fmt.Errorf("restart_policy must be no, on-failure, always or unless-stopped")
		}
	}
	if len(component.Tmpfs) > 0 {
		if _, err := renderRuntimeTmpfs(component.Tmpfs); err != nil {
			return err
		}
	}
	if len(component.Ulimits) > 0 {
		if _, err := renderRuntimeUlimits(component.Ulimits); err != nil {
			return err
		}
	}
	return nil
}

func renderRuntimeTmpfs(entries []model.VersionComponentTmpfs) ([]string, error) {
	if len(entries) > 8 {
		return nil, fmt.Errorf("tmpfs supports at most 8 entries")
	}
	seen := make(map[string]struct{}, len(entries))
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !validRuntimeTmpfsTarget(entry.Target) {
			return nil, fmt.Errorf("tmpfs target is invalid")
		}
		if entry.SizeBytes < minRuntimeTmpfsSizeBytes || entry.SizeBytes > maxRuntimeTmpfsSizeBytes {
			return nil, fmt.Errorf("tmpfs size_bytes must be between %d and %d", minRuntimeTmpfsSizeBytes, maxRuntimeTmpfsSizeBytes)
		}
		if !runtimeTmpfsModePattern.MatchString(entry.Mode) {
			return nil, fmt.Errorf("tmpfs mode must be a 3 or 4 digit octal string")
		}
		if _, exists := seen[entry.Target]; exists {
			return nil, fmt.Errorf("duplicate tmpfs target %s", entry.Target)
		}
		seen[entry.Target] = struct{}{}
		result = append(result, fmt.Sprintf("%s:size=%d,mode=%s", entry.Target, entry.SizeBytes, entry.Mode))
	}
	return result, nil
}

func validRuntimeTmpfsTarget(target string) bool {
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

func renderRuntimeUlimits(entries []model.VersionComponentUlimit) (map[string]map[string]int64, error) {
	if len(entries) > 8 {
		return nil, fmt.Errorf("ulimits supports at most 8 entries")
	}
	result := make(map[string]map[string]int64, len(entries))
	for _, entry := range entries {
		if entry.Name != "memlock" && entry.Name != "nofile" {
			return nil, fmt.Errorf("ulimit name must be memlock or nofile")
		}
		if _, exists := result[entry.Name]; exists {
			return nil, fmt.Errorf("duplicate ulimit %s", entry.Name)
		}
		if entry.Soft == -1 || entry.Hard == -1 {
			if entry.Name != "memlock" || entry.Soft != -1 || entry.Hard != -1 {
				return nil, fmt.Errorf("only memlock may use -1 for both soft and hard")
			}
		} else if entry.Soft < 0 || entry.Hard < 0 || entry.Soft > entry.Hard {
			return nil, fmt.Errorf("ulimit %s requires 0 <= soft <= hard", entry.Name)
		}
		result[entry.Name] = map[string]int64{"soft": entry.Soft, "hard": entry.Hard}
	}
	return result, nil
}
