package deploymentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	deploymentsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/deployment"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
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

func optionalTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func optionalInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func projectScopeId(value string) sql.NullString {
	return sql.NullString{String: strings.TrimSpace(value), Valid: true}
}

func int64Pointer(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func (r Repository) CreateDeployment(ctx context.Context, projectId string, deployment model.Deployment) error {
	projectId = strings.TrimSpace(projectId)
	if deployment.ProjectId == nil || strings.TrimSpace(*deployment.ProjectId) != projectId {
		return fmt.Errorf("deployment project scope does not match create request")
	}
	startedAt := deployment.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	err := r.q(ctx).CreateDeployment(ctx, deploymentsqlc.CreateDeploymentParams{
		Id:                        deployment.Id,
		ProjectId:                 projectScopeId(projectId),
		ApplicationId:             dbmodel.NullString(deployment.ApplicationId),
		ApplicationName:           deployment.ApplicationName,
		VersionId:                 dbmodel.NullString(deployment.VersionId),
		ServiceId:                 dbmodel.NullString(deployment.ServiceId),
		EnvironmentId:             dbmodel.NullString(deployment.EnvironmentId),
		EnvironmentTargetType:     dbmodel.NullString(deployment.EnvironmentTargetType),
		EnvironmentTargetRevision: optionalInt64(deployment.EnvironmentTargetRevision),
		SSHCredentialId:           dbmodel.NullString(deployment.SSHCredentialId),
		SshCredentialRevision:     optionalInt64(deployment.SSHCredentialRevision),
		GatewayApplicationId:      dbmodel.NullString(deployment.GatewayApplicationId),
		OptionsJson:               dbmodel.NullString(deployment.OptionsJSON),
		EffectivePlanHash:         dbmodel.NullString(deployment.EffectivePlanHash),
		OperationType:             deployment.OperationType,
		TriggerType:               deployment.TriggerType,
		CommandText:               deployment.CommandText,
		Status:                    deployment.Status,
		StartedAt:                 startedAt,
		IsRollback:                dbmodel.BoolInt(deployment.IsRollback),
		RollbackFromDeploymentId:  dbmodel.NullString(deployment.RollbackFromDeploymentId),
	})
	if err != nil {
		return fmt.Errorf("create deployment %s: %w", deployment.Id, err)
	}
	return nil
}

func (r Repository) CompleteDeployment(ctx context.Context, projectId string, id string, deployStatus string, message string) (bool, error) {
	q := r.q(ctx)
	startedAt, err := q.DeploymentStartedAt(ctx, deploymentsqlc.DeploymentStartedAtParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return false, fmt.Errorf("load deployment started_at %s: %w", id, sqlcommon.TranslateError(err))
	}
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	rows, err := q.CompleteDeployment(ctx, deploymentsqlc.CompleteDeploymentParams{
		Status:        deployStatus,
		FinishedAt:    sql.NullTime{Time: now, Valid: true},
		DurationMs:    sql.NullInt64{Int64: durationMs, Valid: true},
		ErrorMessage:  message,
		Id:            id,
		ProjectId:     projectScopeId(projectId),
		CurrentStatus: status.WorkStatusRunning,
	})
	if err != nil {
		return false, fmt.Errorf("complete deployment %s: %w", id, err)
	}
	return rows == 1, nil
}

func (r Repository) ListDeployments(ctx context.Context, projectId string, applicationId string, statusFilter string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.Deployment], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	appFilter := strings.TrimSpace(applicationId)
	stFilter := strings.TrimSpace(statusFilter)
	appNS := sql.NullString{String: appFilter, Valid: appFilter != ""}
	statusValue := sql.NullString{String: stFilter, Valid: stFilter != ""}
	applicationNamePattern := sql.NullString{String: pattern, Valid: raw != ""}
	projectId = strings.TrimSpace(projectId)
	q := r.q(ctx)
	total, err := q.CountDeployments(ctx, deploymentsqlc.CountDeploymentsParams{
		ProjectId:              sql.NullString{String: projectId, Valid: true},
		ApplicationId:          appNS,
		Status:                 statusValue,
		ApplicationNamePattern: applicationNamePattern,
		DateFrom:               optionalTime(dateFrom),
		DateTo:                 optionalTime(dateTo),
	})
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("count deployments: %w", err)
	}
	rows, err := q.ListDeployments(ctx, deploymentsqlc.ListDeploymentsParams{
		ProjectId:              sql.NullString{String: projectId, Valid: true},
		ApplicationId:          appNS,
		Status:                 statusValue,
		ApplicationNamePattern: applicationNamePattern,
		DateFrom:               optionalTime(dateFrom),
		DateTo:                 optionalTime(dateTo),
		Offset:                 int32((page - 1) * perPage),
		Limit:                  int32(perPage),
	})
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("list deployments: %w", err)
	}
	items := make([]model.Deployment, 0, len(rows))
	for _, row := range rows {
		items = append(items, deploymentFrom(
			row.Id, row.ProjectId, row.ApplicationId, row.ApplicationName, row.VersionId, row.ServiceId,
			row.ServiceInstanceKey, row.EnvironmentId, row.EnvironmentTargetType, row.EnvironmentTargetRevision, row.SSHCredentialId, row.SshCredentialRevision, row.GatewayApplicationId,
			row.OptionsJson, row.EffectivePlanHash, row.OperationType, row.TriggerType, row.CommandText, row.Status, row.StartedAt, row.FinishedAt,
			row.DurationMs, row.LogText, row.ErrorMessage, row.IsRollback, row.RollbackFromDeploymentId,
		))
	}
	return repository.Page[model.Deployment]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Deployment(ctx context.Context, projectId string, id string) (model.Deployment, error) {
	row, err := r.q(ctx).DeploymentById(ctx, deploymentsqlc.DeploymentByIdParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return model.Deployment{}, fmt.Errorf("load deployment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return deploymentFrom(
		row.Id, row.ProjectId, row.ApplicationId, row.ApplicationName, row.VersionId, row.ServiceId,
		row.ServiceInstanceKey, row.EnvironmentId, row.EnvironmentTargetType, row.EnvironmentTargetRevision, row.SSHCredentialId, row.SshCredentialRevision, row.GatewayApplicationId,
		row.OptionsJson, row.EffectivePlanHash, row.OperationType, row.TriggerType, row.CommandText, row.Status, row.StartedAt, row.FinishedAt,
		row.DurationMs, row.LogText, row.ErrorMessage, row.IsRollback, row.RollbackFromDeploymentId,
	), nil
}

func (r Repository) DeleteDeployment(ctx context.Context, projectId string, id string) error {
	if err := r.q(ctx).DeleteDeployment(ctx, deploymentsqlc.DeleteDeploymentParams{Id: id, ProjectId: projectScopeId(projectId)}); err != nil {
		return fmt.Errorf("delete deployment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return nil
}

func (r Repository) CancelDeployment(ctx context.Context, projectId string, id string) (bool, error) {
	q := r.q(ctx)
	startedAt, err := q.DeploymentStartedAt(ctx, deploymentsqlc.DeploymentStartedAtParams{Id: id, ProjectId: projectScopeId(projectId)})
	if err != nil {
		return false, fmt.Errorf("load deployment started_at %s: %w", id, sqlcommon.TranslateError(err))
	}
	now := time.Now().UTC()
	durationMs := now.Sub(startedAt).Milliseconds()
	rows, err := q.CancelDeployment(ctx, deploymentsqlc.CancelDeploymentParams{
		NewStatus:     status.WorkStatusCanceled,
		FinishedAt:    sql.NullTime{Time: now, Valid: true},
		DurationMs:    sql.NullInt64{Int64: durationMs, Valid: true},
		ErrorMessage:  sql.NullString{String: "Cancelled by user", Valid: true},
		Id:            id,
		ProjectId:     projectScopeId(projectId),
		WaitingStatus: status.WorkStatusWaitingToRun,
		RunningStatus: status.WorkStatusRunning,
	})
	if err != nil {
		return false, fmt.Errorf("cancel deployment %s: %w", id, err)
	}
	return rows == 1, nil
}

func (r Repository) BeginDeployment(ctx context.Context, projectId string, id string) (bool, error) {
	rows, err := r.q(ctx).BeginDeployment(ctx, deploymentsqlc.BeginDeploymentParams{
		Status:        status.WorkStatusRunning,
		StartedAt:     time.Now().UTC(),
		Id:            id,
		ProjectId:     projectScopeId(projectId),
		WaitingStatus: status.WorkStatusWaitingToRun,
	})
	if err != nil {
		return false, fmt.Errorf("begin deployment %s: %w", id, err)
	}
	return rows == 1, nil
}

func (r Repository) HasActiveDeployment(ctx context.Context, projectId string, serviceId string) (bool, error) {
	count, err := r.q(ctx).CountActiveDeploymentsByService(ctx, deploymentsqlc.CountActiveDeploymentsByServiceParams{
		ServiceId:     sql.NullString{String: serviceId, Valid: strings.TrimSpace(serviceId) != ""},
		ProjectId:     projectScopeId(projectId),
		WaitingStatus: status.WorkStatusWaitingToRun,
		RunningStatus: status.WorkStatusRunning,
	})
	if err != nil {
		return false, fmt.Errorf("count active deployments for service %s: %w", serviceId, err)
	}
	return count > 0, nil
}

func (r Repository) LatestSuccessfulDeploymentPlanHash(ctx context.Context, projectId string, serviceId string) (*string, error) {
	value, err := r.q(ctx).LatestSuccessfulDeploymentPlanHash(ctx, deploymentsqlc.LatestSuccessfulDeploymentPlanHashParams{
		ServiceId: sql.NullString{String: serviceId, Valid: serviceId != ""},
		ProjectId: projectScopeId(projectId),
	})
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
	versionId, serviceId, serviceInstanceKey, environmentId, environmentTargetType sql.NullString,
	environmentTargetRevision sql.NullInt64, sshCredentialId sql.NullString, sshCredentialRevision sql.NullInt64,
	gatewayApplicationId, optionsJSON, effectivePlanHash sql.NullString,
	operationType, triggerType, commandText, deployStatus string,
	startedAt time.Time, finishedAt sql.NullTime, durationMs sql.NullInt64,
	logText, errorMessage sql.NullString, isRollback int64, rollbackFrom sql.NullString,
) model.Deployment {
	return model.Deployment{
		Id:                        id,
		ProjectId:                 dbmodel.StringPtr(projectId),
		ApplicationId:             dbmodel.StringPtr(applicationId),
		ApplicationName:           applicationName,
		VersionId:                 dbmodel.StringPtr(versionId),
		ServiceId:                 dbmodel.StringPtr(serviceId),
		ServiceInstanceKey:        dbmodel.StringPtr(serviceInstanceKey),
		EnvironmentId:             dbmodel.StringPtr(environmentId),
		EnvironmentTargetType:     dbmodel.StringPtr(environmentTargetType),
		EnvironmentTargetRevision: int64Pointer(environmentTargetRevision),
		SSHCredentialId:           dbmodel.StringPtr(sshCredentialId),
		SSHCredentialRevision:     int64Pointer(sshCredentialRevision),
		GatewayApplicationId:      dbmodel.StringPtr(gatewayApplicationId),
		OptionsJSON:               dbmodel.StringPtr(optionsJSON),
		EffectivePlanHash:         dbmodel.StringPtr(effectivePlanHash),
		OperationType:             operationType,
		TriggerType:               triggerType,
		CommandText:               commandText,
		Status:                    deployStatus,
		StartedAt:                 startedAt,
		FinishedAt:                dbmodel.TimePtr(finishedAt),
		DurationMs:                dbmodel.IntPtrFromNullInt64(durationMs),
		LogText:                   dbmodel.StringPtr(logText),
		ErrorMessage:              dbmodel.StringPtr(errorMessage),
		IsRollback:                dbmodel.IntBool(isRollback),
		RollbackFromDeploymentId:  dbmodel.StringPtr(rollbackFrom),
	}
}
