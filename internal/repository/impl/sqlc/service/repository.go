package servicerepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	servicesqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/service"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.ServiceStore = Repository{}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return Repository{db: db} }

func (r Repository) q(ctx context.Context) *servicesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *servicesqlc.Queries { return servicesqlc.New(dbtx) })
}

func (r Repository) ListServicesByApplication(ctx context.Context, projectId, applicationId string) ([]model.Service, error) {
	rows, err := r.q(ctx).ListServicesByApplication(ctx, servicesqlc.ListServicesByApplicationParams{ProjectId: projectId, ApplicationId: applicationId})
	if err != nil {
		return nil, fmt.Errorf("list services by application %s: %w", applicationId, err)
	}
	items := make([]model.Service, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceFrom(row))
	}
	return items, nil
}

func (r Repository) ListServicesByVersion(ctx context.Context, projectId, versionId string) ([]model.Service, error) {
	rows, err := r.q(ctx).ListServicesByVersion(ctx, servicesqlc.ListServicesByVersionParams{ProjectId: projectId, VersionId: versionId})
	if err != nil {
		return nil, fmt.Errorf("list services by version %s: %w", versionId, err)
	}
	items := make([]model.Service, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceFrom(row))
	}
	return items, nil
}

func (r Repository) ListServicesByProject(ctx context.Context, projectId, applicationId, statusFilter, search string, page, perPage int) (repository.Page[model.ServiceListItem], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	searchRaw, searchValue := dbmodel.SearchPattern(search)
	applicationFilter := strings.TrimSpace(applicationId)
	serviceStatusFilter := strings.TrimSpace(statusFilter)
	applicationIdFilter := sql.NullString{String: applicationFilter, Valid: applicationFilter != ""}
	serviceStatus := sql.NullString{String: serviceStatusFilter, Valid: serviceStatusFilter != ""}
	searchPattern := sql.NullString{String: searchValue, Valid: searchRaw != ""}
	q := r.q(ctx)
	params := servicesqlc.CountServicesByProjectParams{
		ProjectId:     strings.TrimSpace(projectId),
		ApplicationId: applicationIdFilter,
		Status:        serviceStatus,
		SearchPattern: searchPattern,
	}
	total, err := q.CountServicesByProject(ctx, params)
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("count services by project: %w", err)
	}
	rows, err := q.ListServicesByProject(ctx, servicesqlc.ListServicesByProjectParams{
		ProjectId:     params.ProjectId,
		ApplicationId: applicationIdFilter,
		Status:        serviceStatus,
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("list services by project: %w", err)
	}
	items := make([]model.ServiceListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceListFrom(row.Id, row.ProjectId, row.ApplicationId, row.Code, row.VersionId, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel))
	}
	return repository.Page[model.ServiceListItem]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ServiceListItem(ctx context.Context, projectId, id string) (model.ServiceListItem, error) {
	row, err := r.q(ctx).ServiceListItemById(ctx, servicesqlc.ServiceListItemByIdParams{Id: id, ProjectId: projectId})
	if err != nil {
		return model.ServiceListItem{}, fmt.Errorf("load service list item %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceListFrom(row.Id, row.ProjectId, row.ApplicationId, row.Code, row.VersionId, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel), nil
}

func (r Repository) ServiceByProjectAndCode(ctx context.Context, projectId, code string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByProjectAndCode(ctx, servicesqlc.ServiceByProjectAndCodeParams{ProjectId: projectId, Code: code})
	if err != nil {
		return model.Service{}, fmt.Errorf("load service by project and code: %w", sqlcommon.TranslateError(err))
	}
	return serviceFrom(row), nil
}

func (r Repository) Service(ctx context.Context, projectId, id string) (model.Service, error) {
	row, err := r.q(ctx).ServiceById(ctx, servicesqlc.ServiceByIdParams{Id: id, ProjectId: projectId})
	if err != nil {
		return model.Service{}, fmt.Errorf("load service %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceFrom(row), nil
}

func (r Repository) ServiceEnvByService(ctx context.Context, projectId, serviceId string) ([]model.ServiceEnv, error) {
	rows, err := r.q(ctx).ServiceEnvByService(ctx, servicesqlc.ServiceEnvByServiceParams{ServiceId: serviceId, ProjectId: projectId})
	if err != nil {
		return nil, fmt.Errorf("list service environment %s: %w", serviceId, err)
	}
	items := make([]model.ServiceEnv, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ServiceEnv{Key: row.EnvKey, Value: row.Value})
	}
	return items, nil
}

func (r Repository) ServiceComponentsByService(ctx context.Context, projectId, serviceId string) ([]model.ServiceComponent, error) {
	rows, err := r.q(ctx).ServiceComponentsByService(ctx, servicesqlc.ServiceComponentsByServiceParams{ServiceId: serviceId, ProjectId: projectId})
	if err != nil {
		return nil, fmt.Errorf("list service components %s: %w", serviceId, err)
	}
	items := make([]model.ServiceComponent, 0, len(rows))
	for _, row := range rows {
		component, err := r.serviceComponentFromRow(ctx, r.q(ctx), projectId, row)
		if err != nil {
			return nil, err
		}
		items = append(items, component)
	}
	return items, nil
}

func (r Repository) ServiceComponent(ctx context.Context, projectId, id string) (model.ServiceComponent, error) {
	row, err := r.q(ctx).ServiceComponentById(ctx, servicesqlc.ServiceComponentByIdParams{Id: id, ProjectId: projectId})
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component %s: %w", id, sqlcommon.TranslateError(err))
	}
	return r.serviceComponentFromRow(ctx, r.q(ctx), projectId, row)
}

func (r Repository) UpsertService(ctx context.Context, projectId string, svc model.Service) error {
	if strings.TrimSpace(svc.ProjectId) != strings.TrimSpace(projectId) {
		return fmt.Errorf("service project %s does not match scope %s", svc.ProjectId, projectId)
	}
	if strings.TrimSpace(svc.Id) == "" {
		return fmt.Errorf("service id is required")
	}
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		_, err := q.ServiceById(txCtx, servicesqlc.ServiceByIdParams{Id: svc.Id, ProjectId: projectId})
		if errors.Is(err, sql.ErrNoRows) {
			return r.insertService(txCtx, projectId, svc)
		}
		if err != nil {
			return fmt.Errorf("lookup service %s: %w", svc.Id, err)
		}
		if err := q.UpdateService(txCtx, servicesqlc.UpdateServiceParams{VersionId: svc.VersionId, Status: svc.Status, UpdatedAt: time.Now().UTC(), Id: svc.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("update service %s: %w", svc.Id, err)
		}
		return nil
	})
}

func (r Repository) CreateServiceWithComponents(ctx context.Context, projectId string, svc model.Service, components []model.ServiceComponent) error {
	if strings.TrimSpace(svc.ProjectId) != strings.TrimSpace(projectId) {
		return fmt.Errorf("service project %s does not match scope %s", svc.ProjectId, projectId)
	}
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.insertService(txCtx, projectId, svc); err != nil {
			return err
		}
		return r.replaceServiceComponents(txCtx, projectId, svc, components)
	})
}

func (r Repository) UpdateServiceConfiguration(ctx context.Context, projectId string, svc model.Service, components []model.ServiceComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.UpdateServiceConfiguration(txCtx, servicesqlc.UpdateServiceConfigurationParams{VersionId: svc.VersionId, UpdatedAt: time.Now().UTC(), Id: svc.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("update service configuration %s: %w", svc.Id, err)
		}
		return r.replaceServiceComponents(txCtx, projectId, svc, components)
	})
}

func (r Repository) ReplaceServiceEnv(ctx context.Context, projectId, serviceId string, env []model.ServiceEnv) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteServiceEnv(txCtx, servicesqlc.DeleteServiceEnvParams{ServiceId: serviceId, ProjectId: projectId}); err != nil {
			return fmt.Errorf("delete service environment: %w", err)
		}
		for _, item := range env {
			if err := q.InsertServiceEnv(txCtx, servicesqlc.InsertServiceEnvParams{ServiceId: serviceId, EnvKey: item.Key, Value: item.Value, ProjectId: projectId}); err != nil {
				return fmt.Errorf("insert service environment %s: %w", item.Key, err)
			}
		}
		if err := q.TouchService(txCtx, servicesqlc.TouchServiceParams{UpdatedAt: time.Now().UTC(), Id: serviceId, ProjectId: projectId}); err != nil {
			return fmt.Errorf("touch service %s: %w", serviceId, err)
		}
		return nil
	})
}

func (r Repository) UpdateServiceComponentOverlay(ctx context.Context, projectId string, component model.ServiceComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		entrypointJSON, err := optionalCommandJSON(component.Entrypoint)
		if err != nil {
			return fmt.Errorf("encode service component entrypoint: %w", err)
		}
		commandJSON, err := optionalCommandJSON(component.Command)
		if err != nil {
			return fmt.Errorf("encode service component command: %w", err)
		}
		if err := q.UpdateServiceComponentOverlayFields(txCtx, servicesqlc.UpdateServiceComponentOverlayFieldsParams{
			EntrypointJson: entrypointJSON, CommandJson: commandJSON,
			PullPolicy: dbmodel.NullString(component.PullPolicy), RestartPolicy: dbmodel.NullString(component.RestartPolicy),
			UpdatedAt: time.Now().UTC(), Id: component.Id, ProjectId: projectId,
		}); err != nil {
			return fmt.Errorf("update service component runtime overlay: %w", err)
		}
		if err := q.DeleteServiceComponentEnv(txCtx, servicesqlc.DeleteServiceComponentEnvParams{ServiceComponentId: component.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("delete service component env: %w", err)
		}
		if err := q.DeleteServiceComponentMounts(txCtx, servicesqlc.DeleteServiceComponentMountsParams{ServiceComponentId: component.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("delete service component mounts: %w", err)
		}
		if err := q.DeleteServiceComponentResource(txCtx, servicesqlc.DeleteServiceComponentResourceParams{ServiceComponentId: component.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("delete service component resource: %w", err)
		}
		if err := q.DeleteServiceComponentEndpoints(txCtx, servicesqlc.DeleteServiceComponentEndpointsParams{ServiceComponentId: component.Id, ProjectId: projectId}); err != nil {
			return fmt.Errorf("delete service component endpoints: %w", err)
		}
		return insertServiceComponentOverlay(txCtx, q, projectId, component)
	})
}

func (r Repository) DeleteService(ctx context.Context, projectId, id string) error {
	if err := r.q(ctx).DeleteService(ctx, servicesqlc.DeleteServiceParams{Id: id, ProjectId: projectId}); err != nil {
		return fmt.Errorf("delete service %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceStatus(ctx context.Context, projectId, id, status string) error {
	if err := r.q(ctx).UpdateServiceStatus(ctx, servicesqlc.UpdateServiceStatusParams{Status: status, UpdatedAt: time.Now().UTC(), Id: id, ProjectId: projectId}); err != nil {
		return fmt.Errorf("update service status %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceAfterDeploy(ctx context.Context, projectId, id, status, versionId string) error {
	if err := r.q(ctx).UpdateServiceAfterDeploy(ctx, servicesqlc.UpdateServiceAfterDeployParams{Status: status, VersionId: versionId, UpdatedAt: time.Now().UTC(), Id: id, ProjectId: projectId}); err != nil {
		return fmt.Errorf("update service after deploy %s: %w", id, err)
	}
	return nil
}

func (r Repository) insertService(ctx context.Context, projectId string, svc model.Service) error {
	now := time.Now().UTC()
	createdAt, updatedAt := svc.CreatedAt, svc.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if err := r.q(ctx).InsertService(ctx, servicesqlc.InsertServiceParams{Id: svc.Id, ProjectId: projectId, ApplicationId: svc.ApplicationId, Code: svc.Code, VersionId: svc.VersionId, Status: svc.Status, CreatedAt: createdAt, UpdatedAt: updatedAt}); err != nil {
		return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
	}
	return nil
}

func (r Repository) replaceServiceComponents(ctx context.Context, projectId string, svc model.Service, components []model.ServiceComponent) error {
	q := r.q(ctx)
	if err := q.DeleteServiceComponents(ctx, servicesqlc.DeleteServiceComponentsParams{ServiceId: svc.Id, ProjectId: projectId}); err != nil {
		return fmt.Errorf("delete service components %s: %w", svc.Id, err)
	}
	for _, component := range components {
		if component.ServiceId == "" {
			component.ServiceId = svc.Id
		}
		if err := insertServiceComponent(ctx, q, projectId, component); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) serviceComponentFromRow(ctx context.Context, q *servicesqlc.Queries, projectId string, row servicesqlc.ServiceComponent) (model.ServiceComponent, error) {
	entrypoint, err := optionalCommandFromJSON(row.EntrypointJson)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("decode service component entrypoint %s: %w", row.Id, err)
	}
	command, err := optionalCommandFromJSON(row.CommandJson)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("decode service component command %s: %w", row.Id, err)
	}
	component := model.ServiceComponent{
		Id: row.Id, ServiceId: row.ServiceId, SourceVersionComponentId: row.SourceVersionComponentId, ComponentName: row.ComponentName,
		Entrypoint: entrypoint, Command: command, PullPolicy: dbmodel.StringPtr(row.PullPolicy), RestartPolicy: dbmodel.StringPtr(row.RestartPolicy),
		Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	env, err := q.ServiceComponentEnvByComponent(ctx, servicesqlc.ServiceComponentEnvByComponentParams{ServiceComponentId: component.Id, ProjectId: projectId})
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component env %s: %w", component.Id, err)
	}
	for _, item := range env {
		component.Env = append(component.Env, model.ServiceComponentEnv{Key: item.EnvKey, Value: dbmodel.StringPtr(item.Value), State: model.ServiceComponentOverlayState(item.State)})
	}
	mounts, err := q.ServiceComponentMountsByComponent(ctx, servicesqlc.ServiceComponentMountsByComponentParams{ServiceComponentId: component.Id, ProjectId: projectId})
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component mounts %s: %w", component.Id, err)
	}
	for _, item := range mounts {
		component.Mounts = append(component.Mounts, model.ServiceComponentMount{Target: item.Target, Source: dbmodel.StringPtr(item.Source), SourceIsHostPath: dbmodel.BoolPtrFromNullInt64(item.SourceIsHostPath), State: model.ServiceComponentOverlayState(item.State)})
	}
	resource, err := q.ServiceComponentResourceByComponent(ctx, servicesqlc.ServiceComponentResourceByComponentParams{ServiceComponentId: component.Id, ProjectId: projectId})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.ServiceComponent{}, fmt.Errorf("load service component resource %s: %w", component.Id, err)
	}
	if err == nil {
		component.Resources = &model.ServiceComponentResources{LimitCPUs: dbmodel.StringPtr(resource.LimitCpus), LimitMemory: dbmodel.StringPtr(resource.LimitMemory), ReservationCPUs: dbmodel.StringPtr(resource.ReservationCpus), ReservationMemory: dbmodel.StringPtr(resource.ReservationMemory), State: model.ServiceComponentOverlayState(resource.State)}
	}
	endpoints, err := q.ServiceComponentEndpointsByComponent(ctx, servicesqlc.ServiceComponentEndpointsByComponentParams{ServiceComponentId: component.Id, ProjectId: projectId})
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component endpoints %s: %w", component.Id, err)
	}
	for _, item := range endpoints {
		component.Endpoints = append(component.Endpoints, model.ServiceComponentEndpoint{Id: item.Id, Protocol: item.Protocol, ContainerPort: int(item.ContainerPort), Mode: dbmodel.StringPtr(item.Mode), BindAddress: dbmodel.StringPtr(item.BindAddress), ListenPort: dbmodel.IntPtrFromNullInt64(item.ListenPort), Entrypoint: dbmodel.StringPtr(item.Entrypoint), PathPrefix: dbmodel.StringPtr(item.PathPrefix), State: model.ServiceComponentOverlayState(item.State)})
	}
	return component, nil
}

func insertServiceComponent(ctx context.Context, q *servicesqlc.Queries, projectId string, component model.ServiceComponent) error {
	now := time.Now().UTC()
	createdAt, updatedAt := component.CreatedAt, component.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	entrypointJSON, err := optionalCommandJSON(component.Entrypoint)
	if err != nil {
		return fmt.Errorf("encode service component entrypoint %s: %w", component.ComponentName, err)
	}
	commandJSON, err := optionalCommandJSON(component.Command)
	if err != nil {
		return fmt.Errorf("encode service component command %s: %w", component.ComponentName, err)
	}
	if err := q.InsertServiceComponent(ctx, servicesqlc.InsertServiceComponentParams{
		Id: component.Id, ServiceId: component.ServiceId, SourceVersionComponentId: component.SourceVersionComponentId, ComponentName: component.ComponentName,
		EntrypointJson: entrypointJSON, CommandJson: commandJSON, PullPolicy: dbmodel.NullString(component.PullPolicy), RestartPolicy: dbmodel.NullString(component.RestartPolicy),
		Status: component.Status, CreatedAt: createdAt, UpdatedAt: updatedAt, ProjectId: projectId,
	}); err != nil {
		return fmt.Errorf("insert service component %s: %w", component.ComponentName, err)
	}
	return insertServiceComponentOverlay(ctx, q, projectId, component)
}

func optionalCommandFromJSON(value sql.NullString) ([]string, error) {
	if !value.Valid {
		return nil, nil
	}
	var command []string
	if err := json.Unmarshal([]byte(value.String), &command); err != nil {
		return nil, err
	}
	if command == nil {
		return nil, fmt.Errorf("must be a JSON array")
	}
	return command, nil
}

func optionalCommandJSON(command []string) (sql.NullString, error) {
	if command == nil {
		return sql.NullString{}, nil
	}
	encoded, err := json.Marshal(command)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(encoded), Valid: true}, nil
}

func insertServiceComponentOverlay(ctx context.Context, q *servicesqlc.Queries, projectId string, component model.ServiceComponent) error {
	for _, item := range component.Env {
		if err := q.InsertServiceComponentEnv(ctx, servicesqlc.InsertServiceComponentEnvParams{ServiceComponentId: component.Id, EnvKey: item.Key, Value: dbmodel.NullString(item.Value), State: string(item.State), ProjectId: projectId}); err != nil {
			return fmt.Errorf("insert service component env: %w", err)
		}
	}
	for _, item := range component.Mounts {
		if err := q.InsertServiceComponentMount(ctx, servicesqlc.InsertServiceComponentMountParams{Id: idutil.NewId(), ServiceComponentId: component.Id, Target: item.Target, Source: dbmodel.NullString(item.Source), SourceIsHostPath: dbmodel.NullInt64FromBoolPtr(item.SourceIsHostPath), State: string(item.State), ProjectId: projectId}); err != nil {
			return fmt.Errorf("insert service component mount: %w", err)
		}
	}
	if item := component.Resources; item != nil {
		if err := q.InsertServiceComponentResource(ctx, servicesqlc.InsertServiceComponentResourceParams{ServiceComponentId: component.Id, LimitCpus: dbmodel.NullString(item.LimitCPUs), LimitMemory: dbmodel.NullString(item.LimitMemory), ReservationCpus: dbmodel.NullString(item.ReservationCPUs), ReservationMemory: dbmodel.NullString(item.ReservationMemory), State: string(item.State), ProjectId: projectId}); err != nil {
			return fmt.Errorf("insert service component resources: %w", err)
		}
	}
	for _, item := range component.Endpoints {
		if err := q.InsertServiceComponentEndpoint(ctx, servicesqlc.InsertServiceComponentEndpointParams{Id: idutil.NewId(), ServiceComponentId: component.Id, Protocol: item.Protocol, ContainerPort: int64(item.ContainerPort), Mode: dbmodel.NullString(item.Mode), BindAddress: dbmodel.NullString(item.BindAddress), ListenPort: dbmodel.NullInt64FromIntPtr(item.ListenPort), Entrypoint: dbmodel.NullString(item.Entrypoint), PathPrefix: dbmodel.NullString(item.PathPrefix), State: string(item.State), ProjectId: projectId}); err != nil {
			return fmt.Errorf("insert service component endpoint: %w", err)
		}
	}
	return nil
}

func serviceFrom(row servicesqlc.Service) model.Service {
	return model.Service{Id: row.Id, ProjectId: row.ProjectId, ApplicationId: row.ApplicationId, Code: row.Code, VersionId: row.VersionId, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func serviceListFrom(id, projectId, applicationId, code, versionId, status string, createdAt, updatedAt time.Time, applicationName, applicationCode, applicationKind, versionLabel string) model.ServiceListItem {
	return model.ServiceListItem{Id: id, ProjectId: projectId, ApplicationId: applicationId, Code: code, VersionId: versionId, Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt, ApplicationName: applicationName, ApplicationCode: applicationCode, ApplicationKind: applicationKind, VersionLabel: versionLabel}
}
