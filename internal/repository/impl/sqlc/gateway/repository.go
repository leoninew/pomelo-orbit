package gatewayrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

func (r Repository) GatewayConfigByProject(ctx context.Context, projectId string) (model.GatewayConfig, error) {
	binding, err := r.q(ctx).GatewayBindingByProjectId(ctx, strings.TrimSpace(projectId))
	if err != nil {
		return model.GatewayConfig{}, fmt.Errorf("load gateway binding for project %s: %w", projectId, sqlcommon.TranslateError(err))
	}
	cfg, err := r.GatewayConfig(ctx, binding)
	if err != nil {
		return model.GatewayConfig{}, err
	}
	cfg.NetworkName = model.GatewayNetworkName()
	return cfg, nil
}

func (r Repository) ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error) {
	trimmed := strings.TrimSpace(projectId)
	if trimmed == "" {
		return nil, fmt.Errorf("project id is required")
	}
	rows, err := r.q(ctx).ListGatewayApplications(ctx, sql.NullString{String: trimmed, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list gateway applications: %w", err)
	}
	items := make([]model.Application, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.Application{
			Id:        row.Id,
			ProjectId: dbmodel.StringPtr(row.ProjectId),
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
			ApplicationId:           cfg.ApplicationId,
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
		RestApiUrl:              cfg.RestApiUrl,
		RestReadyTimeoutSeconds: int64(cfg.RestReadyTimeoutSeconds),
		BaseDomain:              cfg.BaseDomain,
		DefaultEntrypoint:       cfg.DefaultEntrypoint,
		TlsMode:                 cfg.TLSMode,
		AcmeProfile:             cfg.AcmeProfile,
		AcmeEmail:               cfg.AcmeEmail,
		DnsApiToken:             cfg.DNSApiToken,
		UpdatedAt:               now,
		ApplicationId:           cfg.ApplicationId,
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
				ApplicationId: applicationId,
				Profile:       binding.Profile,
				VersionId:     binding.VersionId,
			}); err != nil {
				return fmt.Errorf("insert gateway Version binding %s/%s: %w", applicationId, binding.Profile, err)
			}
		}
		return nil
	})
}

func (r Repository) gatewayFrom(ctx context.Context, row gatewaysqlc.GatewayConfig) (model.GatewayConfig, error) {
	bindings, err := r.q(ctx).GatewayVersionBindingsByApplication(ctx, row.ApplicationId)
	if err != nil {
		return model.GatewayConfig{}, fmt.Errorf("load gateway Version bindings %s: %w", row.ApplicationId, sqlcommon.TranslateError(err))
	}
	result := model.GatewayConfig{
		ApplicationId:           row.ApplicationId,
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
			Profile: binding.Profile, VersionId: binding.VersionId,
		})
	}
	serviceCode, err := r.q(ctx).GatewayRuntimeServiceCode(ctx, row.ApplicationId)
	if err == nil {
		result.RuntimeServiceCode = serviceCode
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.GatewayConfig{}, fmt.Errorf("load gateway service code %s: %w", row.ApplicationId, sqlcommon.TranslateError(err))
	}
	return result, nil
}
