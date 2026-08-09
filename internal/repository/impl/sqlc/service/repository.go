package servicerepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	servicesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.ServiceStore = Repository{}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return Repository{db: db} }

func (r Repository) q(ctx context.Context) *servicesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *servicesqlc.Queries { return servicesqlc.New(dbtx) })
}

func (r Repository) ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error) {
	rows, err := r.q(ctx).ListServicesByApplication(ctx, applicationId)
	if err != nil {
		return nil, fmt.Errorf("list services by application %s: %w", applicationId, err)
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
	applicationID := sql.NullString{String: applicationFilter, Valid: applicationFilter != ""}
	serviceStatus := sql.NullString{String: serviceStatusFilter, Valid: serviceStatusFilter != ""}
	searchPattern := sql.NullString{String: searchValue, Valid: searchRaw != ""}
	q := r.q(ctx)
	params := servicesqlc.CountServicesByProjectParams{
		ProjectID:     strings.TrimSpace(projectId),
		ApplicationID: applicationID,
		Status:        serviceStatus,
		SearchPattern: searchPattern,
	}
	total, err := q.CountServicesByProject(ctx, params)
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("count services by project: %w", err)
	}
	rows, err := q.ListServicesByProject(ctx, servicesqlc.ListServicesByProjectParams{
		ProjectID:     params.ProjectID,
		ApplicationID: applicationID,
		Status:        serviceStatus,
		SearchPattern: searchPattern,
		Limit:         int64(perPage),
		Offset:        int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("list services by project: %w", err)
	}
	items := make([]model.ServiceListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.Code, row.VersionID, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel))
	}
	return repository.Page[model.ServiceListItem]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error) {
	row, err := r.q(ctx).ServiceListItemByID(ctx, id)
	if err != nil {
		return model.ServiceListItem{}, fmt.Errorf("load service list item %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.Code, row.VersionID, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel), nil
}

func (r Repository) ServiceByKey(ctx context.Context, applicationId, instanceKey string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByKey(ctx, servicesqlc.ServiceByKeyParams{ApplicationID: applicationId, InstanceKey: instanceKey})
	if err != nil {
		return model.Service{}, fmt.Errorf("load service by key: %w", sqlcommon.TranslateError(err))
	}
	return serviceFrom(row), nil
}

func (r Repository) ServiceByCode(ctx context.Context, code string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByCode(ctx, code)
	if err != nil {
		return model.Service{}, fmt.Errorf("load service by code: %w", sqlcommon.TranslateError(err))
	}
	return serviceFrom(row), nil
}

func (r Repository) Service(ctx context.Context, id string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByID(ctx, id)
	if err != nil {
		return model.Service{}, fmt.Errorf("load service %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceFrom(row), nil
}

func (r Repository) ServiceEnvByService(ctx context.Context, serviceId string) ([]model.ServiceEnv, error) {
	rows, err := r.q(ctx).ServiceEnvByService(ctx, serviceId)
	if err != nil {
		return nil, fmt.Errorf("list service environment %s: %w", serviceId, err)
	}
	items := make([]model.ServiceEnv, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.ServiceEnv{Key: row.EnvKey, Value: row.Value})
	}
	return items, nil
}

func (r Repository) ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error) {
	rows, err := r.q(ctx).ServiceComponentsByService(ctx, serviceId)
	if err != nil {
		return nil, fmt.Errorf("list service components %s: %w", serviceId, err)
	}
	items := make([]model.ServiceComponent, 0, len(rows))
	for _, row := range rows {
		component, err := r.serviceComponentFromRow(ctx, r.q(ctx), row)
		if err != nil {
			return nil, err
		}
		items = append(items, component)
	}
	return items, nil
}

func (r Repository) ServiceComponent(ctx context.Context, id string) (model.ServiceComponent, error) {
	row, err := r.q(ctx).ServiceComponentByID(ctx, id)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component %s: %w", id, sqlcommon.TranslateError(err))
	}
	return r.serviceComponentFromRow(ctx, r.q(ctx), row)
}

func (r Repository) UpsertService(ctx context.Context, svc model.Service) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		existingID, err := q.ServiceIDByKey(txCtx, servicesqlc.ServiceIDByKeyParams{ApplicationID: svc.ApplicationId, InstanceKey: svc.InstanceKey})
		if errors.Is(err, sql.ErrNoRows) {
			return r.insertService(txCtx, svc)
		}
		if err != nil {
			return fmt.Errorf("lookup service for application %s: %w", svc.ApplicationId, err)
		}
		id := existingID
		if strings.TrimSpace(svc.Id) != "" {
			id = svc.Id
		}
		if err := q.UpdateService(txCtx, servicesqlc.UpdateServiceParams{VersionID: svc.VersionId, Status: svc.Status, UpdatedAt: time.Now().UTC(), ID: id}); err != nil {
			return fmt.Errorf("update service %s: %w", id, err)
		}
		return nil
	})
}

func (r Repository) CreateServiceWithComponents(ctx context.Context, svc model.Service, components []model.ServiceComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.insertService(txCtx, svc); err != nil {
			return err
		}
		return r.replaceServiceComponents(txCtx, svc, components)
	})
}

func (r Repository) UpdateServiceConfiguration(ctx context.Context, svc model.Service, components []model.ServiceComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.UpdateServiceConfiguration(txCtx, servicesqlc.UpdateServiceConfigurationParams{InstanceKey: svc.InstanceKey, VersionID: svc.VersionId, UpdatedAt: time.Now().UTC(), ID: svc.Id}); err != nil {
			return fmt.Errorf("update service configuration %s: %w", svc.Id, err)
		}
		return r.replaceServiceComponents(txCtx, svc, components)
	})
}

func (r Repository) ReplaceServiceEnv(ctx context.Context, serviceId string, env []model.ServiceEnv) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteServiceEnv(txCtx, serviceId); err != nil {
			return fmt.Errorf("delete service environment: %w", err)
		}
		for _, item := range env {
			if err := q.InsertServiceEnv(txCtx, servicesqlc.InsertServiceEnvParams{ServiceID: serviceId, EnvKey: item.Key, Value: item.Value}); err != nil {
				return fmt.Errorf("insert service environment %s: %w", item.Key, err)
			}
		}
		if err := q.TouchService(txCtx, servicesqlc.TouchServiceParams{UpdatedAt: time.Now().UTC(), ID: serviceId}); err != nil {
			return fmt.Errorf("touch service %s: %w", serviceId, err)
		}
		return nil
	})
}

func (r Repository) UpdateServiceComponentOverlay(ctx context.Context, component model.ServiceComponent) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteServiceComponentEnv(txCtx, component.Id); err != nil {
			return fmt.Errorf("delete service component env: %w", err)
		}
		if err := q.DeleteServiceComponentMounts(txCtx, component.Id); err != nil {
			return fmt.Errorf("delete service component mounts: %w", err)
		}
		if err := q.DeleteServiceComponentResource(txCtx, component.Id); err != nil {
			return fmt.Errorf("delete service component resource: %w", err)
		}
		if err := q.DeleteServiceComponentEndpoints(txCtx, component.Id); err != nil {
			return fmt.Errorf("delete service component endpoints: %w", err)
		}
		return insertServiceComponentOverlay(txCtx, q, component)
	})
}

func (r Repository) DeleteService(ctx context.Context, id string) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteService(txCtx, id); err != nil {
			return fmt.Errorf("delete service %s: %w", id, err)
		}
		return nil
	})
}

func (r Repository) UpdateServiceStatus(ctx context.Context, id, status string) error {
	if err := r.q(ctx).UpdateServiceStatus(ctx, servicesqlc.UpdateServiceStatusParams{Status: status, UpdatedAt: time.Now().UTC(), ID: id}); err != nil {
		return fmt.Errorf("update service status %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceAfterDeploy(ctx context.Context, id, status, versionId string) error {
	if err := r.q(ctx).UpdateServiceAfterDeploy(ctx, servicesqlc.UpdateServiceAfterDeployParams{Status: status, VersionID: versionId, UpdatedAt: time.Now().UTC(), ID: id}); err != nil {
		return fmt.Errorf("update service after deploy %s: %w", id, err)
	}
	return nil
}

func (r Repository) insertService(ctx context.Context, svc model.Service) error {
	now := time.Now().UTC()
	createdAt, updatedAt := svc.CreatedAt, svc.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if err := r.q(ctx).InsertService(ctx, servicesqlc.InsertServiceParams{ID: svc.Id, ApplicationID: svc.ApplicationId, InstanceKey: svc.InstanceKey, Code: svc.Code, VersionID: svc.VersionId, Status: svc.Status, CreatedAt: createdAt, UpdatedAt: updatedAt}); err != nil {
		return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
	}
	return nil
}

func (r Repository) replaceServiceComponents(ctx context.Context, svc model.Service, components []model.ServiceComponent) error {
	q := r.q(ctx)
	if err := q.DeleteServiceComponents(ctx, svc.Id); err != nil {
		return fmt.Errorf("delete service components %s: %w", svc.Id, err)
	}
	for _, component := range components {
		if component.ServiceId == "" {
			component.ServiceId = svc.Id
		}
		if err := insertServiceComponent(ctx, q, component); err != nil {
			return err
		}
	}
	return nil
}

func (r Repository) serviceComponentFromRow(ctx context.Context, q *servicesqlc.Queries, row servicesqlc.ServiceComponent) (model.ServiceComponent, error) {
	component := model.ServiceComponent{Id: row.ID, ServiceId: row.ServiceID, SourceVersionComponentId: row.SourceVersionComponentID, ComponentName: row.ComponentName, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	env, err := q.ServiceComponentEnvByComponent(ctx, component.Id)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component env %s: %w", component.Id, err)
	}
	for _, item := range env {
		component.Env = append(component.Env, model.ServiceComponentEnv{Key: item.EnvKey, Value: dbmodel.StringPtr(item.Value), State: model.ServiceComponentOverlayState(item.State)})
	}
	mounts, err := q.ServiceComponentMountsByComponent(ctx, component.Id)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component mounts %s: %w", component.Id, err)
	}
	for _, item := range mounts {
		component.Mounts = append(component.Mounts, model.ServiceComponentMount{Target: item.Target, Source: dbmodel.StringPtr(item.Source), SourceIsHostPath: dbmodel.BoolPtrFromNullInt64(item.SourceIsHostPath), State: model.ServiceComponentOverlayState(item.State)})
	}
	resource, err := q.ServiceComponentResourceByComponent(ctx, component.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.ServiceComponent{}, fmt.Errorf("load service component resource %s: %w", component.Id, err)
	}
	if err == nil {
		component.Resources = &model.ServiceComponentResources{LimitCPUs: dbmodel.StringPtr(resource.LimitCpus), LimitMemory: dbmodel.StringPtr(resource.LimitMemory), ReservationCPUs: dbmodel.StringPtr(resource.ReservationCpus), ReservationMemory: dbmodel.StringPtr(resource.ReservationMemory), State: model.ServiceComponentOverlayState(resource.State)}
	}
	endpoints, err := q.ServiceComponentEndpointsByComponent(ctx, component.Id)
	if err != nil {
		return model.ServiceComponent{}, fmt.Errorf("load service component endpoints %s: %w", component.Id, err)
	}
	for _, item := range endpoints {
		component.Endpoints = append(component.Endpoints, model.ServiceComponentEndpoint{Name: item.Name, Mode: dbmodel.StringPtr(item.Mode), BindAddress: dbmodel.StringPtr(item.BindAddress), ListenPort: dbmodel.IntPtrFromNullInt64(item.ListenPort), Entrypoint: dbmodel.StringPtr(item.Entrypoint), PathPrefix: dbmodel.StringPtr(item.PathPrefix), State: model.ServiceComponentOverlayState(item.State)})
	}
	return component, nil
}

func insertServiceComponent(ctx context.Context, q *servicesqlc.Queries, component model.ServiceComponent) error {
	now := time.Now().UTC()
	createdAt, updatedAt := component.CreatedAt, component.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if err := q.InsertServiceComponent(ctx, servicesqlc.InsertServiceComponentParams{ID: component.Id, ServiceID: component.ServiceId, SourceVersionComponentID: component.SourceVersionComponentId, ComponentName: component.ComponentName, Status: component.Status, CreatedAt: createdAt, UpdatedAt: updatedAt}); err != nil {
		return fmt.Errorf("insert service component %s: %w", component.ComponentName, err)
	}
	return insertServiceComponentOverlay(ctx, q, component)
}

func insertServiceComponentOverlay(ctx context.Context, q *servicesqlc.Queries, component model.ServiceComponent) error {
	for _, item := range component.Env {
		if err := q.InsertServiceComponentEnv(ctx, servicesqlc.InsertServiceComponentEnvParams{ServiceComponentID: component.Id, EnvKey: item.Key, Value: dbmodel.NullString(item.Value), State: string(item.State)}); err != nil {
			return fmt.Errorf("insert service component env: %w", err)
		}
	}
	for _, item := range component.Mounts {
		if err := q.InsertServiceComponentMount(ctx, servicesqlc.InsertServiceComponentMountParams{ID: idutil.NewId(), ServiceComponentID: component.Id, Target: item.Target, Source: dbmodel.NullString(item.Source), SourceIsHostPath: dbmodel.NullInt64FromBoolPtr(item.SourceIsHostPath), State: string(item.State)}); err != nil {
			return fmt.Errorf("insert service component mount: %w", err)
		}
	}
	if item := component.Resources; item != nil {
		if err := q.InsertServiceComponentResource(ctx, servicesqlc.InsertServiceComponentResourceParams{ServiceComponentID: component.Id, LimitCpus: dbmodel.NullString(item.LimitCPUs), LimitMemory: dbmodel.NullString(item.LimitMemory), ReservationCpus: dbmodel.NullString(item.ReservationCPUs), ReservationMemory: dbmodel.NullString(item.ReservationMemory), State: string(item.State)}); err != nil {
			return fmt.Errorf("insert service component resources: %w", err)
		}
	}
	for _, item := range component.Endpoints {
		if err := q.InsertServiceComponentEndpoint(ctx, servicesqlc.InsertServiceComponentEndpointParams{ServiceComponentID: component.Id, Name: item.Name, Mode: dbmodel.NullString(item.Mode), BindAddress: dbmodel.NullString(item.BindAddress), ListenPort: dbmodel.NullInt64FromIntPtr(item.ListenPort), Entrypoint: dbmodel.NullString(item.Entrypoint), PathPrefix: dbmodel.NullString(item.PathPrefix), State: string(item.State)}); err != nil {
			return fmt.Errorf("insert service component endpoint: %w", err)
		}
	}
	return nil
}

func serviceFrom(row servicesqlc.Service) model.Service {
	return model.Service{Id: row.ID, ApplicationId: row.ApplicationID, InstanceKey: row.InstanceKey, Code: row.Code, VersionId: row.VersionID, Status: row.Status, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func serviceListFrom(id, applicationId, instanceKey, code, versionId, status string, createdAt, updatedAt time.Time, applicationName, applicationCode, applicationKind, versionLabel string) model.ServiceListItem {
	return model.ServiceListItem{Id: id, ApplicationId: applicationId, InstanceKey: instanceKey, Code: code, VersionId: versionId, Status: status, CreatedAt: createdAt, UpdatedAt: updatedAt, ApplicationName: applicationName, ApplicationCode: applicationCode, ApplicationKind: applicationKind, VersionLabel: versionLabel}
}
