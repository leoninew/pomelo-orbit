package deploymentsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) materializeRuntimeEnvFiles(ctx context.Context, app model.Application, components []model.VersionComponent, service model.Service) (map[string]string, error) {
	allValues := map[string]string{}
	for _, component := range components {
		if len(component.SecretEnvRefs) == 0 {
			continue
		}
		values, err := s.resolveComponentRuntimeEnv(ctx, app, component)
		if err != nil {
			return nil, fmt.Errorf("component %s runtime_env: %w", component.Name, err)
		}
		if err := s.workspace.WriteRuntimeEnv(app.Code, service.InstanceKey, component.Name, runtimeEnvFileContent(values)); err != nil {
			return nil, fmt.Errorf("write component %s runtime env file: %w", component.Name, err)
		}
		for key, value := range values {
			allValues[key] = value
		}
	}
	return allValues, nil
}

func (s Service) resolveComponentRuntimeEnv(ctx context.Context, app model.Application, component model.VersionComponent) (map[string]string, error) {
	if s.credential == nil || s.secretKey == "" {
		return nil, fmt.Errorf("runtime_env credential store is not configured")
	}
	values := make(map[string]string, len(component.SecretEnvRefs))
	for _, ref := range component.SecretEnvRefs {
		if ref.EnvKey == "" || ref.CredentialId == "" || ref.DataKey == "" {
			return nil, fmt.Errorf("invalid secret environment reference")
		}
		if _, exists := values[ref.EnvKey]; exists {
			return nil, fmt.Errorf("duplicate secret environment key %s", ref.EnvKey)
		}
		credential, err := s.credential.Credential(ctx, ref.CredentialId)
		if err != nil {
			return nil, fmt.Errorf("load credential %s: %w", ref.CredentialId, err)
		}
		if credential.Type != "runtime_env" || app.ProjectId == nil || credential.ProjectId == nil || *credential.ProjectId != *app.ProjectId {
			return nil, fmt.Errorf("credential %s is not a runtime_env credential in this project", ref.CredentialId)
		}
		plain, err := security.DecryptString(s.secretKey, credential.EncryptedData)
		if err != nil {
			return nil, fmt.Errorf("decrypt credential %s: %w", ref.CredentialId, err)
		}
		credentialValues, err := parseRuntimeEnvValues(plain)
		if err != nil {
			return nil, fmt.Errorf("credential %s data: %w", ref.CredentialId, err)
		}
		value, exists := credentialValues[ref.DataKey]
		if !exists {
			return nil, fmt.Errorf("credential %s does not define data_key %s", ref.CredentialId, ref.DataKey)
		}
		values[ref.EnvKey] = value
	}
	return values, nil
}

func parseRuntimeEnvValues(raw string) (map[string]string, error) {
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("must be a JSON object of string values: %w", err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("must not be empty")
	}
	for key, value := range values {
		if !runtimeEnvNamePattern.MatchString(key) || value == "" {
			return nil, fmt.Errorf("contains an invalid environment value")
		}
	}
	return values, nil
}

func runtimeEnvFileContent(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(strconv.Quote(values[key]))
		builder.WriteByte('\n')
	}
	return builder.String()
}
