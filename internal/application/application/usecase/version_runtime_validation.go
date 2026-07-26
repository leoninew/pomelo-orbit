package applicationsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

const (
	minTmpfsSizeBytes = 1 << 20
	maxTmpfsSizeBytes = 8 << 30
)

var runtimeEnvNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
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

type componentEnvVar struct {
	Key string `json:"key"`
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
	return validateSecretEnvRefs(component)
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

func validateSecretEnvRefs(component model.VersionComponent) error {
	if len(component.SecretEnvRefs) == 0 {
		return nil
	}
	if len(component.SecretEnvRefs) > 64 {
		return fmt.Errorf("component %s secret_env_refs supports at most 64 entries", component.Name)
	}
	envKeys, err := componentEnvKeys(component.EnvJSON)
	if err != nil {
		return fmt.Errorf("component %s env_json: %w", component.Name, err)
	}
	seen := make(map[string]struct{}, len(component.SecretEnvRefs))
	for _, ref := range component.SecretEnvRefs {
		if !runtimeEnvNamePattern.MatchString(ref.EnvKey) || !runtimeEnvNamePattern.MatchString(ref.DataKey) || ref.CredentialId == "" {
			return fmt.Errorf("component %s has invalid secret_env_ref", component.Name)
		}
		if _, exists := seen[ref.EnvKey]; exists {
			return fmt.Errorf("component %s has duplicate secret env key %s", component.Name, ref.EnvKey)
		}
		if _, exists := envKeys[ref.EnvKey]; exists {
			return fmt.Errorf("component %s secret env key %s conflicts with env_json", component.Name, ref.EnvKey)
		}
		seen[ref.EnvKey] = struct{}{}
	}
	return nil
}

func componentEnvKeys(raw *string) (map[string]struct{}, error) {
	keys := map[string]struct{}{}
	if raw == nil || *raw == "" {
		return keys, nil
	}
	var entries []componentEnvVar
	if err := json.Unmarshal([]byte(*raw), &entries); err != nil {
		return nil, fmt.Errorf("must be an array of key/value entries")
	}
	for _, entry := range entries {
		keys[entry.Key] = struct{}{}
	}
	return keys, nil
}

func runtimeEnvData(raw string) (map[string]string, error) {
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("data must be a JSON object of string values")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("data must not be empty")
	}
	for key, value := range values {
		if !runtimeEnvNamePattern.MatchString(key) || value == "" {
			return nil, fmt.Errorf("data contains an invalid runtime environment value")
		}
	}
	return values, nil
}

func (s Service) validateRuntimeEnvReferences(ctx context.Context, app model.Application, components []model.VersionComponent) error {
	for _, component := range components {
		for _, ref := range component.SecretEnvRefs {
			if s.credential == nil || s.secretKey == "" {
				return apperror.New(apperror.KindInternal, "runtime_env credential validation is not configured")
			}
			credential, err := s.credential.Credential(ctx, ref.CredentialId)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return apperror.New(apperror.KindValidation, "runtime_env credential "+ref.CredentialId+" not found")
				}
				return apperror.Wrap(apperror.KindInternal, "Failed to load runtime_env credential", err)
			}
			if credential.Type != "runtime_env" || app.ProjectId == nil || credential.ProjectId == nil || *credential.ProjectId != *app.ProjectId {
				return apperror.New(apperror.KindValidation, "runtime_env credential does not belong to the application project")
			}
			values, err := s.decryptRuntimeEnvData(credential.EncryptedData)
			if err != nil {
				return apperror.Wrap(apperror.KindInternal, "Failed to read runtime_env credential", err)
			}
			if _, exists := values[ref.DataKey]; !exists {
				return apperror.New(apperror.KindValidation, "runtime_env credential data_key is missing")
			}
		}
	}
	return nil
}
