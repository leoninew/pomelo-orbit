package deploymentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	deploymentsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/deployment"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.DeploymentStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *deploymentsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *deploymentsqlc.Queries {
		return deploymentsqlc.New(dbtx)
	})
}

func optionalTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func (r Repository) CreateDeployment(ctx context.Context, deployment model.Deployment) error {
	startedAt := deployment.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	err := r.q(ctx).CreateDeployment(ctx, deploymentsqlc.CreateDeploymentParams{
		ID:                       deployment.Id,
		ProjectID:                dbmodel.NullString(deployment.ProjectId),
		ApplicationID:            dbmodel.NullString(deployment.ApplicationId),
		ApplicationName:          deployment.ApplicationName,
		VersionID:                dbmodel.NullString(deployment.VersionId),
		ServiceID:                dbmodel.NullString(deployment.ServiceId),
		OptionsJson:              dbmodel.NullString(deployment.OptionsJSON),
		EffectivePlanHash:        dbmodel.NullString(deployment.EffectivePlanHash),
		OperationType:            deployment.OperationType,
		TriggerType:              deployment.TriggerType,
		CommandText:              deployment.CommandText,
		Status:                   deployment.Status,
		StartedAt:                startedAt,
		IsRollback:               dbmodel.BoolInt(deployment.IsRollback),
		RollbackFromDeploymentID: dbmodel.NullString(deployment.RollbackFromDeploymentId),
	})
	if err != nil {
		return fmt.Errorf("create deployment %s: %w", deployment.Id, err)
	}
	return nil
}

func (r Repository) CompleteDeployment(ctx context.Context, id string, deployStatus string, message string) error {
	q := r.q(ctx)
	startedAt, err := q.DeploymentStartedAt(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment started_at %s: %w", id, sqlcommon.TranslateError(err))
	}
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	err = q.CompleteDeployment(ctx, deploymentsqlc.CompleteDeploymentParams{
		Status:     deployStatus,
		FinishedAt: sql.NullTime{Time: now, Valid: true},
		DurationMs: sql.NullInt64{Int64: durationMs, Valid: true},
		NULLIF:     message,
		ID:         id,
	})
	if err != nil {
		return fmt.Errorf("complete deployment %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListDeployments(ctx context.Context, projectId string, applicationId string, statusFilter string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.Deployment], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	appFilter := strings.TrimSpace(applicationId)
	stFilter := strings.TrimSpace(statusFilter)
	appNS := sql.NullString{String: appFilter, Valid: appFilter != ""}
	q := r.q(ctx)
	total, err := q.CountDeployments(ctx, deploymentsqlc.CountDeploymentsParams{
		ProjectID:             sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		ApplicationFilter:     appFilter,
		ApplicationID:         appNS,
		StatusFilter:          stFilter,
		Status:                stFilter,
		ApplicationNameFilter: raw,
		ApplicationName:       pattern,
		DateFrom:              optionalTime(dateFrom),
		DateTo:                optionalTime(dateTo),
	})
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("count deployments: %w", err)
	}
	rows, err := q.ListDeployments(ctx, deploymentsqlc.ListDeploymentsParams{
		ProjectID:             sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		ApplicationFilter:     appFilter,
		ApplicationID:         appNS,
		StatusFilter:          stFilter,
		Status:                stFilter,
		ApplicationNameFilter: raw,
		ApplicationName:       pattern,
		DateFrom:              optionalTime(dateFrom),
		DateTo:                optionalTime(dateTo),
		Offset:                int64((page - 1) * perPage),
		Limit:                 int64(perPage),
	})
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("list deployments: %w", err)
	}
	items := make([]model.Deployment, 0, len(rows))
	for _, row := range rows {
		items = append(items, deploymentFrom(
			row.ID, row.ProjectID, row.ApplicationID, row.ApplicationName, row.VersionID, row.ServiceID,
			row.OptionsJson, row.EffectivePlanHash, row.OperationType, row.TriggerType, row.CommandText, row.Status, row.StartedAt, row.FinishedAt,
			row.DurationMs, row.LogText, row.ErrorMessage, row.IsRollback, row.RollbackFromDeploymentID,
		))
	}
	return repository.Page[model.Deployment]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Deployment(ctx context.Context, id string) (model.Deployment, error) {
	row, err := r.q(ctx).DeploymentByID(ctx, id)
	if err != nil {
		return model.Deployment{}, fmt.Errorf("load deployment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return deploymentFrom(
		row.ID, row.ProjectID, row.ApplicationID, row.ApplicationName, row.VersionID, row.ServiceID,
		row.OptionsJson, row.EffectivePlanHash, row.OperationType, row.TriggerType, row.CommandText, row.Status, row.StartedAt, row.FinishedAt,
		row.DurationMs, row.LogText, row.ErrorMessage, row.IsRollback, row.RollbackFromDeploymentID,
	), nil
}

func (r Repository) CancelDeployment(ctx context.Context, id string) error {
	q := r.q(ctx)
	startedAt, err := q.DeploymentStartedAt(ctx, id)
	if err != nil {
		return fmt.Errorf("load deployment started_at %s: %w", id, sqlcommon.TranslateError(err))
	}
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	err = q.CancelDeployment(ctx, deploymentsqlc.CancelDeploymentParams{
		Status:       status.WorkStatusCanceled,
		FinishedAt:   sql.NullTime{Time: now, Valid: true},
		DurationMs:   sql.NullInt64{Int64: durationMs, Valid: true},
		ErrorMessage: sql.NullString{String: "Cancelled by user", Valid: true},
		ID:           id,
	})
	if err != nil {
		return fmt.Errorf("cancel deployment %s: %w", id, err)
	}
	return nil
}

func (r Repository) MarkDeploymentRunning(ctx context.Context, id string) error {
	err := r.q(ctx).MarkDeploymentRunning(ctx, deploymentsqlc.MarkDeploymentRunningParams{
		Status:    status.WorkStatusRunning,
		StartedAt: time.Now().UTC(),
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("mark deployment running %s: %w", id, err)
	}
	return nil
}

func (r Repository) LatestSuccessfulDeploymentPlanHash(ctx context.Context, serviceId string) (*string, error) {
	value, err := r.q(ctx).LatestSuccessfulDeploymentPlanHash(ctx, sql.NullString{String: serviceId, Valid: serviceId != ""})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load latest successful deployment plan hash for service %s: %w", serviceId, err)
	}
	return dbmodel.StringPtr(value), nil
}

func deploymentFrom(
	id string, projectId, applicationId sql.NullString, applicationName string,
	versionId, serviceId, optionsJSON, effectivePlanHash sql.NullString,
	operationType, triggerType, commandText, deployStatus string,
	startedAt time.Time, finishedAt sql.NullTime, durationMs sql.NullInt64,
	logText, errorMessage sql.NullString, isRollback int64, rollbackFrom sql.NullString,
) model.Deployment {
	return model.Deployment{
		Id:                       id,
		ProjectId:                dbmodel.StringPtr(projectId),
		ApplicationId:            dbmodel.StringPtr(applicationId),
		ApplicationName:          applicationName,
		VersionId:                dbmodel.StringPtr(versionId),
		ServiceId:                dbmodel.StringPtr(serviceId),
		OptionsJSON:              dbmodel.StringPtr(optionsJSON),
		EffectivePlanHash:        dbmodel.StringPtr(effectivePlanHash),
		OperationType:            operationType,
		TriggerType:              triggerType,
		CommandText:              commandText,
		Status:                   deployStatus,
		StartedAt:                startedAt,
		FinishedAt:               dbmodel.TimePtr(finishedAt),
		DurationMs:               dbmodel.IntPtrFromNullInt64(durationMs),
		LogText:                  dbmodel.StringPtr(logText),
		ErrorMessage:             dbmodel.StringPtr(errorMessage),
		IsRollback:               dbmodel.IntBool(isRollback),
		RollbackFromDeploymentId: dbmodel.StringPtr(rollbackFrom),
	}
}
