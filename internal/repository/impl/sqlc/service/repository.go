package servicerepo

import (
	"context"
	"database/sql"
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
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *servicesqlc.Queries {
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
		items = append(items, serviceFrom(row))
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
		items = append(items, serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.VersionID, row.LastSuccessfulVersionID, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel, row.LastSuccessfulVersionLabel))
	}
	return repository.Page[model.ServiceListItem]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error) {
	row, err := r.q(ctx).ServiceListItemByID(ctx, id)
	if err != nil {
		return model.ServiceListItem{}, fmt.Errorf("load service list item %s: %w", id, sqlcommon.TranslateError(err))
	}
	return serviceListFrom(row.ID, row.ApplicationID, row.InstanceKey, row.VersionID, row.LastSuccessfulVersionID, row.Status, row.CreatedAt, row.UpdatedAt, row.ApplicationName, row.ApplicationCode, row.ApplicationKind, row.VersionLabel, row.LastSuccessfulVersionLabel), nil
}

func (r Repository) ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error) {
	row, err := r.q(ctx).ServiceByKey(ctx, servicesqlc.ServiceByKeyParams{
		ApplicationID: applicationId,
		InstanceKey:   instanceKey,
	})
	if err != nil {
		return model.Service{}, fmt.Errorf("load service by key: %w", sqlcommon.TranslateError(err))
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

func (r Repository) UpsertService(ctx context.Context, svc model.Service) error {
	q := r.q(ctx)
	existingID, err := q.ServiceIDByKey(ctx, servicesqlc.ServiceIDByKeyParams{
		ApplicationID: svc.ApplicationId,
		InstanceKey:   svc.InstanceKey,
	})
	now := time.Now().UTC()
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
			ID:                      svc.Id,
			ApplicationID:           svc.ApplicationId,
			InstanceKey:             svc.InstanceKey,
			VersionID:               svc.VersionId,
			LastSuccessfulVersionID: dbmodel.NullString(svc.LastSuccessfulVersionId),
			Status:                  svc.Status,
			CreatedAt:               createdAt,
			UpdatedAt:               updatedAt,
		}); err != nil {
			return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
		}
		return nil
	}
	id := existingID
	if strings.TrimSpace(svc.Id) != "" {
		id = svc.Id
	}
	if err := q.UpdateService(ctx, servicesqlc.UpdateServiceParams{
		VersionID:               svc.VersionId,
		LastSuccessfulVersionID: dbmodel.NullString(svc.LastSuccessfulVersionId),
		Status:                  svc.Status,
		UpdatedAt:               now,
		ID:                      id,
	}); err != nil {
		return fmt.Errorf("update service %s: %w", id, err)
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

func (r Repository) UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error {
	err := r.q(ctx).UpdateServiceAfterDeploy(ctx, servicesqlc.UpdateServiceAfterDeployParams{
		Status:                  status,
		VersionID:               versionId,
		LastSuccessfulVersionID: dbmodel.NullString(lastSuccessfulVersionId),
		UpdatedAt:               time.Now().UTC(),
		ID:                      id,
	})
	if err != nil {
		return fmt.Errorf("update service after deploy %s: %w", id, err)
	}
	return nil
}

func serviceFrom(row servicesqlc.Service) model.Service {
	return model.Service{
		Id:                      row.ID,
		ApplicationId:           row.ApplicationID,
		InstanceKey:             row.InstanceKey,
		VersionId:               row.VersionID,
		LastSuccessfulVersionId: dbmodel.StringPtr(row.LastSuccessfulVersionID),
		Status:                  row.Status,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
}

func serviceListFrom(
	id, applicationID, instanceKey, versionID string, lastSuccessfulVersionID sql.NullString,
	svcStatus string, createdAt, updatedAt time.Time,
	applicationName, applicationCode, applicationKind, versionLabel string,
	lastSuccessfulVersionLabel sql.NullString,
) model.ServiceListItem {
	return model.ServiceListItem{
		Id:                         id,
		ApplicationId:              applicationID,
		InstanceKey:                instanceKey,
		VersionId:                  versionID,
		LastSuccessfulVersionId:    dbmodel.StringPtr(lastSuccessfulVersionID),
		Status:                     svcStatus,
		CreatedAt:                  createdAt,
		UpdatedAt:                  updatedAt,
		ApplicationName:            applicationName,
		ApplicationCode:            applicationCode,
		ApplicationKind:            applicationKind,
		VersionLabel:               versionLabel,
		LastSuccessfulVersionLabel: dbmodel.StringPtr(lastSuccessfulVersionLabel),
	}
}
