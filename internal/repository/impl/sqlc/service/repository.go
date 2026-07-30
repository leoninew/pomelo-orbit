package servicerepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	servicesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.ServiceStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *servicesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *servicesqlc.Queries {
		return servicesqlc.New(dbtx)
	})
}

func (r Repository) ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error) {
	rows, err := r.q(ctx).ListServicesByApplication(ctx, applicationId)
	if err != nil {
		return nil, fmt.Errorf("list services by application %s: %w", applicationId, err)
	}
	items := make([]model.Service, 0, len(rows))
	for _, row := range rows {
		item, err := serviceFrom(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r Repository) ListServicesByProject(ctx context.Context, projectId string, applicationId string, statusFilter string, search string, page int, perPage int) (repository.Page[model.ServiceListItem], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	appFilter := strings.TrimSpace(applicationId)
	stFilter := strings.TrimSpace(statusFilter)
	q := r.q(ctx)
	total, err := q.CountServicesByProject(ctx, servicesqlc.CountServicesByProjectParams{
		ProjectID:     sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:       appFilter,
		ApplicationID: appFilter,
		Column4:       stFilter,
		Status:        stFilter,
		Column6:       raw,
		Name:          pattern,
		Code:          pattern,
		InstanceKey:   pattern,
		Label:         pattern,
	})
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("count services by project: %w", err)
	}
	rows, err := q.ListServicesByProject(ctx, servicesqlc.ListServicesByProjectParams{
		ProjectID:     sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Column2:       appFilter,
		ApplicationID: appFilter,
		Column4:       stFilter,
		Status:        stFilter,
		Column6:       raw,
		Name:          pattern,
		Code:          pattern,
		InstanceKey:   pattern,
		Label:         pattern,
		Limit:         int64(perPage),
		Offset:        int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.ServiceListItem]{}, fmt.Errorf("list services by project: %w", err)
	}
	items := make([]model.ServiceListItem, 0, len(rows))
	for _, row := range rows {
		item, err := serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.VersionID, row.RuntimeConfigJson, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel)
		if err != nil {
			return repository.Page[model.ServiceListItem]{}, err
		}
		items = append(items, item)
	}
	return repository.Page[model.ServiceListItem]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error) {
	row, err := r.q(ctx).ServiceListItemByID(ctx, id)
	if err != nil {
		return model.ServiceListItem{}, fmt.Errorf("load service list item %s: %w", id, sqlcommon.TranslateError(err))
	}
	item, err := serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.VersionID, row.RuntimeConfigJson, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel)
	if err != nil {
		return model.ServiceListItem{}, err
	}
	return item, nil
}

func (r Repository) ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByKey(ctx, servicesqlc.ServiceByKeyParams{
		ApplicationID: applicationId,
		InstanceKey:   instanceKey,
	})
	if err != nil {
		return model.Service{}, fmt.Errorf("load service by key: %w", sqlcommon.TranslateError(err))
	}
	return serviceFrom(row)
}

func (r Repository) Service(ctx context.Context, id string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByID(ctx, id)
	if err != nil {
		return model.Service{}, fmt.Errorf("load service %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceFrom(row)
}

func (r Repository) ServiceExposesByService(ctx context.Context, serviceId string) ([]model.ServiceExpose, error) {
	rows, err := r.q(ctx).ServiceExposesByService(ctx, serviceId)
	if err != nil {
		return nil, fmt.Errorf("list service exposes %s: %w", serviceId, err)
	}
	items := make([]model.ServiceExpose, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceExposeFrom(row))
	}
	return items, nil
}

func (r Repository) LocalServiceExposesByListen(ctx context.Context, listenPort int) ([]model.ServiceExpose, error) {
	rows, err := r.q(ctx).LocalServiceExposesByListen(ctx, sql.NullInt64{Int64: int64(listenPort), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list local service exposes on %d: %w", listenPort, err)
	}
	items := make([]model.ServiceExpose, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceExposeFrom(row))
	}
	return items, nil
}

func (r Repository) PublicTCPServiceExposesByListen(ctx context.Context, listenPort int) ([]model.ServiceExpose, error) {
	rows, err := r.q(ctx).PublicTCPServiceExposesByListen(ctx, sql.NullInt64{Int64: int64(listenPort), Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list public TCP service exposes on %d: %w", listenPort, err)
	}
	items := make([]model.ServiceExpose, 0, len(rows))
	for _, row := range rows {
		items = append(items, serviceExposeFrom(row))
	}
	return items, nil
}

func (r Repository) CountServiceExposesByVersionComponent(ctx context.Context, versionId string, componentName string) (int, error) {
	count, err := r.q(ctx).CountServiceExposesByVersionComponent(ctx, servicesqlc.CountServiceExposesByVersionComponentParams{
		VersionID: versionId, ComponentName: componentName,
	})
	if err != nil {
		return 0, fmt.Errorf("count service exposes for version component %s/%s: %w", versionId, componentName, err)
	}
	return int(count), nil
}

func (r Repository) UpsertService(ctx context.Context, svc model.Service) error {
	q := r.q(ctx)
	existingId, err := q.ServiceIDByKey(ctx, servicesqlc.ServiceIDByKeyParams{
		ApplicationID: svc.ApplicationId,
		InstanceKey:   svc.InstanceKey,
	})
	now := time.Now().UTC()
	runtimeConfigJSON, configErr := marshalRuntimeConfig(svc.RuntimeConfig)
	if configErr != nil {
		return configErr
	}
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lookup service for application %s: %w", svc.ApplicationId, err)
		}
		createdAt, updatedAt := svc.CreatedAt, svc.UpdatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		if err := q.InsertService(ctx, servicesqlc.InsertServiceParams{
			ID:                svc.Id,
			ApplicationID:     svc.ApplicationId,
			InstanceKey:       svc.InstanceKey,
			VersionID:         svc.VersionId,
			RuntimeConfigJson: runtimeConfigJSON,
			Status:            svc.Status,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}); err != nil {
			return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
		}
		return nil
	}
	id := existingId
	if strings.TrimSpace(svc.Id) != "" {
		id = svc.Id
	}
	if err := q.UpdateService(ctx, servicesqlc.UpdateServiceParams{
		VersionID:         svc.VersionId,
		RuntimeConfigJson: runtimeConfigJSON,
		Status:            svc.Status,
		UpdatedAt:         now,
		ID:                id,
	}); err != nil {
		return fmt.Errorf("update service %s: %w", id, err)
	}
	return nil
}

func (r Repository) CreateServiceWithExposes(ctx context.Context, svc model.Service, exposes []model.ServiceExpose) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		if err := r.insertService(txCtx, svc); err != nil {
			return err
		}
		return r.replaceServiceExposes(txCtx, svc.Id, exposes)
	})
}

func (r Repository) UpdateServiceConfiguration(ctx context.Context, svc model.Service, exposes []model.ServiceExpose) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		runtimeConfigJSON, err := marshalRuntimeConfig(svc.RuntimeConfig)
		if err != nil {
			return err
		}
		if err := r.q(txCtx).UpdateServiceConfiguration(txCtx, servicesqlc.UpdateServiceConfigurationParams{
			InstanceKey: svc.InstanceKey, VersionID: svc.VersionId, RuntimeConfigJson: runtimeConfigJSON, UpdatedAt: time.Now().UTC(), ID: svc.Id,
		}); err != nil {
			return fmt.Errorf("update service configuration %s: %w", svc.Id, err)
		}
		return r.replaceServiceExposes(txCtx, svc.Id, exposes)
	})
}

func (r Repository) ReplaceServiceExposes(ctx context.Context, serviceId string, exposes []model.ServiceExpose) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		return r.replaceServiceExposes(txCtx, serviceId, exposes)
	})
}

func (r Repository) UpdateServiceRuntimeConfig(ctx context.Context, id string, runtimeConfig map[string]string) error {
	raw, err := marshalRuntimeConfig(runtimeConfig)
	if err != nil {
		return err
	}
	if err := r.q(ctx).UpdateServiceRuntimeConfig(ctx, servicesqlc.UpdateServiceRuntimeConfigParams{
		RuntimeConfigJson: raw,
		UpdatedAt:         time.Now().UTC(),
		ID:                id,
	}); err != nil {
		return fmt.Errorf("update service runtime config %s: %w", id, err)
	}
	return nil
}

func (r Repository) DeleteService(ctx context.Context, id string) error {
	q := r.q(ctx)
	if err := q.DetachDeploymentServiceRefs(ctx, sql.NullString{String: id, Valid: true}); err != nil {
		return fmt.Errorf("detach deployment service refs for service %s: %w", id, err)
	}
	if err := q.DeleteService(ctx, id); err != nil {
		return fmt.Errorf("delete service %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceStatus(ctx context.Context, id string, status string) error {
	err := r.q(ctx).UpdateServiceStatus(ctx, servicesqlc.UpdateServiceStatusParams{
		Status:    status,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("update service status %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string) error {
	err := r.q(ctx).UpdateServiceAfterDeploy(ctx, servicesqlc.UpdateServiceAfterDeployParams{
		Status:    status,
		VersionID: versionId,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("update service after deploy %s: %w", id, err)
	}
	return nil
}

func (r Repository) insertService(ctx context.Context, svc model.Service) error {
	now := time.Now().UTC()
	runtimeConfigJSON, err := marshalRuntimeConfig(svc.RuntimeConfig)
	if err != nil {
		return err
	}
	createdAt, updatedAt := svc.CreatedAt, svc.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if err := r.q(ctx).InsertService(ctx, servicesqlc.InsertServiceParams{
		ID: svc.Id, ApplicationID: svc.ApplicationId, InstanceKey: svc.InstanceKey, VersionID: svc.VersionId,
		RuntimeConfigJson: runtimeConfigJSON, Status: svc.Status, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}); err != nil {
		return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
	}
	return nil
}

func (r Repository) replaceServiceExposes(ctx context.Context, serviceId string, exposes []model.ServiceExpose) error {
	q := r.q(ctx)
	if err := q.DeleteServiceExposes(ctx, serviceId); err != nil {
		return fmt.Errorf("delete service exposes %s: %w", serviceId, err)
	}
	now := time.Now().UTC()
	for _, expose := range exposes {
		createdAt, updatedAt := expose.CreatedAt, expose.UpdatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		if err := q.InsertServiceExpose(ctx, servicesqlc.InsertServiceExposeParams{
			ID: expose.Id, ServiceID: serviceId, ComponentName: expose.ComponentName, Protocol: expose.Protocol,
			ContainerPort: int64(expose.ContainerPort), PathPrefix: dbmodel.NullString(expose.PathPrefix), Access: expose.Access,
			ListenPort: dbmodel.NullInt64FromIntPtr(expose.ListenPort), CreatedAt: createdAt, UpdatedAt: updatedAt,
		}); err != nil {
			return fmt.Errorf("insert service expose %s: %w", expose.Id, err)
		}
	}
	return nil
}

func serviceFrom(row servicesqlc.Service) (model.Service, error) {
	runtimeConfig, err := unmarshalRuntimeConfig(row.RuntimeConfigJson)
	if err != nil {
		return model.Service{}, fmt.Errorf("decode service runtime config %s: %w", row.ID, err)
	}
	return model.Service{
		Id:            row.ID,
		ApplicationId: row.ApplicationID,
		InstanceKey:   row.InstanceKey,
		VersionId:     row.VersionID,
		RuntimeConfig: runtimeConfig,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func serviceExposeFrom(row servicesqlc.ServiceExpose) model.ServiceExpose {
	return model.ServiceExpose{
		Id:            row.ID,
		ServiceId:     row.ServiceID,
		ComponentName: row.ComponentName,
		Protocol:      row.Protocol,
		ContainerPort: int(row.ContainerPort),
		PathPrefix:    dbmodel.StringPtr(row.PathPrefix),
		Access:        row.Access,
		ListenPort:    dbmodel.IntPtrFromNullInt64(row.ListenPort),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func serviceListFrom(
	id, applicationId, instanceKey, versionId, runtimeConfigJSON string,
	svcStatus string, createdAt, updatedAt time.Time,
	applicationName, applicationCode, applicationKind, versionLabel string,
) (model.ServiceListItem, error) {
	runtimeConfig, err := unmarshalRuntimeConfig(runtimeConfigJSON)
	if err != nil {
		return model.ServiceListItem{}, fmt.Errorf("decode service runtime config %s: %w", id, err)
	}
	return model.ServiceListItem{
		Id:              id,
		ApplicationId:   applicationId,
		InstanceKey:     instanceKey,
		VersionId:       versionId,
		RuntimeConfig:   runtimeConfig,
		Status:          svcStatus,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		ApplicationName: applicationName,
		ApplicationCode: applicationCode,
		ApplicationKind: applicationKind,
		VersionLabel:    versionLabel,
	}, nil
}

func marshalRuntimeConfig(values map[string]string) (string, error) {
	if values == nil {
		values = map[string]string{}
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode runtime config: %w", err)
	}
	return string(raw), nil
}

func unmarshalRuntimeConfig(raw string) (map[string]string, error) {
	values := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}
