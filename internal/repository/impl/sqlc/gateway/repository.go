package gatewayrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	gatewaysqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/gateway"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.GatewayStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *gatewaysqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *gatewaysqlc.Queries {
		return gatewaysqlc.New(dbtx)
	})
}

func (r Repository) HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error) {
	exclude := strings.TrimSpace(excludeApplicationId)
	count, err := r.q(ctx).CountActiveGatewayServices(ctx, gatewaysqlc.CountActiveGatewayServicesParams{
		Kind:                 status.ApplicationKindGateway,
		ExcludeApplicationID: sql.NullString{String: exclude, Valid: exclude != ""},
		ServiceStatus:        status.ServiceStatusRunning,
		WaitingStatus:        status.WorkStatusWaitingToRun,
		RunningStatus:        status.WorkStatusRunning,
	})
	if err != nil {
		return false, fmt.Errorf("count active gateway services: %w", err)
	}
	return count > 0, nil
}

func (r Repository) GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error) {
	row, err := r.q(ctx).GatewayConfigByApplication(ctx, applicationId)
	if err != nil {
		return model.GatewayConfig{}, fmt.Errorf("load gateway config %s: %w", applicationId, sqlcommon.TranslateError(err))
	}
	return gatewayFrom(row), nil
}

func (r Repository) ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error) {
	row, err := r.q(ctx).ResolveActiveGatewayConfig(ctx, gatewaysqlc.ResolveActiveGatewayConfigParams{
		Kind:   status.ApplicationKindGateway,
		Status: status.ServiceStatusRunning,
	})
	if err == nil {
		return gatewayFrom(row), nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		translated := sqlcommon.TranslateError(err)
		if !errors.Is(translated, repository.ErrNotFound) {
			return model.GatewayConfig{}, fmt.Errorf("resolve active gateway config: %w", err)
		}
	}

	// Fallback: when no gateway service is running/deploying, use the sole
	// gateway application config so routes/certs can render before first deploy.
	apps, listErr := r.ListGatewayApplications(ctx, "")
	if listErr != nil {
		return model.GatewayConfig{}, fmt.Errorf("list gateways for resolve: %w", listErr)
	}
	if len(apps) == 0 {
		return model.GatewayConfig{}, fmt.Errorf("no gateway configured: %w", repository.ErrNotFound)
	}
	if len(apps) > 1 {
		return model.GatewayConfig{}, fmt.Errorf("multiple gateways exist and none is active; deploy one or remove extras: %w", repository.ErrNotFound)
	}
	return r.GatewayConfig(ctx, apps[0].Id)
}

func (r Repository) ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error) {
	trimmed := strings.TrimSpace(projectId)
	var rows []gatewaysqlc.ListGatewayApplicationsRow
	var err error
	if trimmed == "" {
		all, listErr := r.q(ctx).ListAllGatewayApplications(ctx, status.ApplicationKindGateway)
		if listErr != nil {
			return nil, fmt.Errorf("list all gateway applications: %w", listErr)
		}
		items := make([]model.Application, 0, len(all))
		for _, row := range all {
			items = append(items, model.Application{
				Id:        row.ID,
				ProjectId: dbmodel.StringPtr(row.ProjectID),
				Name:      row.Name,
				Code:      row.Code,
				Kind:      row.Kind,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		return items, nil
	}
	rows, err = r.q(ctx).ListGatewayApplications(ctx, gatewaysqlc.ListGatewayApplicationsParams{
		ProjectID: sql.NullString{String: trimmed, Valid: true},
		Kind:      status.ApplicationKindGateway,
	})
	if err != nil {
		return nil, fmt.Errorf("list gateway applications: %w", err)
	}
	items := make([]model.Application, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.Application{
			Id:        row.ID,
			ProjectId: dbmodel.StringPtr(row.ProjectID),
			Name:      row.Name,
			Code:      row.Code,
			Kind:      row.Kind,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return items, nil
}

func (r Repository) UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error {
	q := r.q(ctx)
	now := time.Now().UTC()
	_, err := q.GatewayConfigByApplication(ctx, cfg.ApplicationId)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lookup gateway config %s: %w", cfg.ApplicationId, err)
		}
		createdAt, updatedAt := cfg.CreatedAt, cfg.UpdatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		if updatedAt.IsZero() {
			updatedAt = now
		}
		if err := q.InsertGatewayConfig(ctx, gatewaysqlc.InsertGatewayConfigParams{
			ApplicationID:     cfg.ApplicationId,
			RestApiUrl:        cfg.RestApiUrl,
			BaseDomain:        cfg.BaseDomain,
			DefaultEntrypoint: cfg.DefaultEntrypoint,
			TlsMode:           cfg.TLSMode,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}); err != nil {
			return fmt.Errorf("insert gateway config %s: %w", cfg.ApplicationId, err)
		}
		return nil
	}
	if err := q.UpdateGatewayConfig(ctx, gatewaysqlc.UpdateGatewayConfigParams{
		RestApiUrl:        cfg.RestApiUrl,
		BaseDomain:        cfg.BaseDomain,
		DefaultEntrypoint: cfg.DefaultEntrypoint,
		TlsMode:           cfg.TLSMode,
		UpdatedAt:         now,
		ApplicationID:     cfg.ApplicationId,
	}); err != nil {
		return fmt.Errorf("update gateway config %s: %w", cfg.ApplicationId, err)
	}
	return nil
}

func gatewayFrom(row gatewaysqlc.GatewayConfig) model.GatewayConfig {
	return model.GatewayConfig{
		ApplicationId:     row.ApplicationID,
		RestApiUrl:        row.RestApiUrl,
		BaseDomain:        row.BaseDomain,
		DefaultEntrypoint: row.DefaultEntrypoint,
		TLSMode:           row.TlsMode,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
