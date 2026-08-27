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

func (r Repository) GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error) {
	row, err := r.q(ctx).GatewayConfigByApplication(ctx, applicationId)
	if err != nil {
		return model.GatewayConfig{}, fmt.Errorf("load gateway config %s: %w", applicationId, sqlcommon.TranslateError(err))
	}
	return r.gatewayFrom(ctx, row)
}

func (r Repository) ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error) {
	row, err := r.q(ctx).ResolveActiveGatewayConfig(ctx, status.ServiceStatusRunning)
	if err == nil {
		return r.gatewayFrom(ctx, row)
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
		all, listErr := r.q(ctx).ListAllGatewayApplications(ctx)
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
	rows, err = r.q(ctx).ListGatewayApplications(ctx, sql.NullString{String: trimmed, Valid: true})
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
			ApplicationID:           cfg.ApplicationId,
			TraefikComponentName:    cfg.TraefikComponentName,
			RestApiUrl:              cfg.RestApiUrl,
			RestReadyTimeoutSeconds: int64(cfg.RestReadyTimeoutSeconds),
			BaseDomain:              cfg.BaseDomain,
			DefaultEntrypoint:       cfg.DefaultEntrypoint,
			TlsMode:                 cfg.TLSMode,
			AcmeProfile:             cfg.AcmeProfile,
			AcmeEmail:               cfg.AcmeEmail,
			DnsApiToken:             cfg.DNSApiToken,
			CreatedAt:               createdAt,
			UpdatedAt:               updatedAt,
		}); err != nil {
			return fmt.Errorf("insert gateway config %s: %w", cfg.ApplicationId, err)
		}
	} else if err := q.UpdateGatewayConfig(ctx, gatewaysqlc.UpdateGatewayConfigParams{
		TraefikComponentName:    cfg.TraefikComponentName,
		RestApiUrl:              cfg.RestApiUrl,
		RestReadyTimeoutSeconds: int64(cfg.RestReadyTimeoutSeconds),
		BaseDomain:              cfg.BaseDomain,
		DefaultEntrypoint:       cfg.DefaultEntrypoint,
		TlsMode:                 cfg.TLSMode,
		AcmeProfile:             cfg.AcmeProfile,
		AcmeEmail:               cfg.AcmeEmail,
		DnsApiToken:             cfg.DNSApiToken,
		UpdatedAt:               now,
		ApplicationID:           cfg.ApplicationId,
	}); err != nil {
		return fmt.Errorf("update gateway config %s: %w", cfg.ApplicationId, err)
	}
	return nil
}

func (r Repository) ReplaceGatewayVersionBindings(ctx context.Context, applicationId string, bindings []model.GatewayVersionBinding) error {
	return tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		if err := q.DeleteGatewayVersionBindings(txCtx, applicationId); err != nil {
			return fmt.Errorf("delete gateway Version bindings %s: %w", applicationId, err)
		}
		for _, binding := range bindings {
			if err := q.InsertGatewayVersionBinding(txCtx, gatewaysqlc.InsertGatewayVersionBindingParams{
				ApplicationID: applicationId,
				Profile:       binding.Profile,
				VersionID:     binding.VersionId,
			}); err != nil {
				return fmt.Errorf("insert gateway Version binding %s/%s: %w", applicationId, binding.Profile, err)
			}
		}
		return nil
	})
}

func (r Repository) gatewayFrom(ctx context.Context, row gatewaysqlc.GatewayConfig) (model.GatewayConfig, error) {
	bindings, err := r.q(ctx).GatewayVersionBindingsByApplication(ctx, row.ApplicationID)
	if err != nil {
		return model.GatewayConfig{}, fmt.Errorf("load gateway Version bindings %s: %w", row.ApplicationID, sqlcommon.TranslateError(err))
	}
	result := model.GatewayConfig{
		ApplicationId:           row.ApplicationID,
		TraefikComponentName:    row.TraefikComponentName,
		RestApiUrl:              row.RestApiUrl,
		RestReadyTimeoutSeconds: int(row.RestReadyTimeoutSeconds),
		BaseDomain:              row.BaseDomain,
		DefaultEntrypoint:       row.DefaultEntrypoint,
		TLSMode:                 row.TlsMode,
		AcmeProfile:             row.AcmeProfile,
		AcmeEmail:               row.AcmeEmail,
		DNSApiToken:             row.DnsApiToken,
		VersionBindings:         make([]model.GatewayVersionBinding, 0, len(bindings)),
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
	}
	for _, binding := range bindings {
		result.VersionBindings = append(result.VersionBindings, model.GatewayVersionBinding{
			Profile: binding.Profile, VersionId: binding.VersionID,
		})
	}
	serviceCode, err := r.q(ctx).GatewayRuntimeServiceCode(ctx, row.ApplicationID)
	if err == nil {
		result.RuntimeServiceCode = serviceCode
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.GatewayConfig{}, fmt.Errorf("load gateway service code %s: %w", row.ApplicationID, sqlcommon.TranslateError(err))
	}
	return result, nil
}
