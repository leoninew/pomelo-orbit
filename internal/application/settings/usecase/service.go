package settingssvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	settingsport "github.com/leoninew/pomelo-orbit/internal/application/settings/port"
	"github.com/leoninew/pomelo-orbit/internal/common/envfile"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
)

type Service struct {
	runtime  *config.Runtime
	envStore settingsport.EnvStore
}

func New(runtime *config.Runtime, envStore settingsport.EnvStore) Service {
	return Service{runtime: runtime, envStore: envStore}
}

func (s Service) Config(ctx context.Context) (settingsdto.SystemConfig, error) {
	values, err := s.envStore.Load(ctx)
	if err != nil {
		return settingsdto.SystemConfig{}, storeError(err)
	}
	resolved, resolveErr := s.runtime.Resolve(values)
	return s.describe(values, resolved, resolveErr), nil
}

func (s Service) Update(ctx context.Context, expectedRevision string, updates []settingsdto.Update, resetKeys []string) (settingsdto.SystemConfig, error) {
	if expectedRevision == "" || len(updates)+len(resetKeys) == 0 {
		return settingsdto.SystemConfig{}, apperror.New(apperror.KindValidation, "revision and configuration changes are required")
	}
	fields := make(map[string]config.Field)
	for _, field := range config.Fields() {
		fields[field.Key] = field
	}
	changes := make(map[string]string)
	seen := make(map[string]bool)
	for _, update := range updates {
		field, ok := fields[update.Key]
		if !ok || seen[update.Key] {
			return settingsdto.SystemConfig{}, apperror.New(apperror.KindValidation, "unknown or repeated configuration key")
		}
		seen[update.Key] = true
		value, err := field.Encode(update.Value)
		if err != nil {
			return settingsdto.SystemConfig{}, apperror.New(apperror.KindValidation, err.Error())
		}
		changes[field.EnvName] = value
	}
	var deletes []string
	for _, key := range resetKeys {
		field, ok := fields[key]
		if !ok || seen[key] {
			return settingsdto.SystemConfig{}, apperror.New(apperror.KindValidation, "unknown or repeated configuration key")
		}
		seen[key] = true
		deletes = append(deletes, field.EnvName)
	}
	var response settingsdto.SystemConfig
	_, err := s.envStore.Mutate(ctx, func(values map[string]string) error {
		if revision(values) != expectedRevision {
			return apperror.NewWithCode(apperror.KindConflict, "settings_revision_conflict", "Configuration changed; reload before saving")
		}
		for key, value := range changes {
			values[key] = value
		}
		for _, key := range deletes {
			delete(values, key)
		}
		resolved, err := s.runtime.Resolve(values)
		if err != nil {
			return apperror.New(apperror.KindValidation, err.Error())
		}
		response = s.describe(values, resolved, nil)
		return nil
	})
	if err != nil {
		return settingsdto.SystemConfig{}, storeError(err)
	}
	return response, nil
}

func (s Service) Reset(ctx context.Context, expectedRevision string, keys []string) (settingsdto.SystemConfig, error) {
	return s.Update(ctx, expectedRevision, nil, keys)
}

func (s Service) describe(values map[string]string, next config.Resolved, nextErr error) settingsdto.SystemConfig {
	effective, initial := s.runtime.Effective(), s.runtime.Initial()
	secretKeys := make(map[string]bool)
	for _, key := range effective.Settings.SecretKeys {
		secretKeys[strings.TrimSpace(key)] = true
	}
	response := settingsdto.SystemConfig{Revision: revision(values)}
	if nextErr != nil {
		response.NextConfigError = nextErr.Error()
		response.PendingRestart = true
	}
	for _, field := range config.Fields() {
		raw, overridden := values[field.EnvName]
		var override any
		if overridden {
			parsed, err := field.ParseEnv(raw)
			if err != nil {
				override = raw
			} else if duration, ok := parsed.(time.Duration); ok {
				override = duration.String()
			} else {
				override = parsed
			}
		}
		item := settingsdto.ConfigItem{
			Key: field.Key, Type: field.Type, Description: field.Description,
			Value: field.Value(effective), Default: next.Baseline[field.Key], OverrideValue: override,
			IsOverridden: overridden, Secret: secretKeys[field.Key],
			ValueSource: s.runtime.Source(field.Key), DefaultSource: next.BaselineSources[field.Key], NextSource: next.Sources[field.Key],
			NextValueKnown: nextErr == nil, PendingRestart: nextErr != nil,
		}
		if nextErr == nil {
			item.NextValue = field.Value(next.Config)
			item.PendingRestart = !reflect.DeepEqual(field.Value(initial), item.NextValue)
			if field.Key == "worker__id" && next.Config.Worker.Id == "" {
				item.NextValue = nil
				item.NextValueKnown = false
				item.NextSource = "derived"
			}
		}
		response.PendingRestart = response.PendingRestart || item.PendingRestart
		response.Items = append(response.Items, item)
	}
	return response
}

func revision(values map[string]string) string {
	content, _ := envfile.Encode(values)
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:])
}

func storeError(err error) error {
	if _, ok := apperror.As(err); ok {
		return err
	}
	if errors.Is(err, envfile.ErrBusy) {
		return apperror.WrapWithCode(apperror.KindUnavailable, "settings_store_busy", "Configuration file is busy", err)
	}
	return apperror.Wrap(apperror.KindInternal, fmt.Sprintf("Failed to access %s", config.OverrideFile), err)
}
