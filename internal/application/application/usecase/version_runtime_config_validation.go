package applicationsvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	runtimeconfig "gitee.com/leoninew/PomeloOrbit-go/internal/common/runtimeconfig"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) validateServicesRuntimeConfig(ctx context.Context, version model.Version, components []model.VersionComponent) error {
	services, err := s.store.ListServicesByApplication(ctx, version.ApplicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load services for runtime config validation", err)
	}
	for _, service := range services {
		if service.VersionId != version.Id {
			continue
		}
		missing, extra, err := runtimeconfig.Validate(service.RuntimeConfig, version.EnvJSON, components)
		if err != nil {
			return apperror.Wrap(apperror.KindValidation, "Invalid version runtime config", err)
		}
		if len(missing) > 0 {
			return apperror.New(apperror.KindValidation, fmt.Sprintf("service %s is missing runtime config keys: %s", service.InstanceKey, strings.Join(missing, ", ")))
		}
		if len(extra) > 0 {
			slog.WarnContext(ctx, "service runtime config has unused keys", "service_id", service.Id, "version_id", version.Id, "keys", extra)
		}
	}
	return nil
}
