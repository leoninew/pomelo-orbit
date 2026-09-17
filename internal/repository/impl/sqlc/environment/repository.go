package environmentrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	environmentsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/environment"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.EnvironmentStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *environmentsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *environmentsqlc.Queries {
		return environmentsqlc.New(dbtx)
	})
}

func (r Repository) Environment(ctx context.Context, id string) (model.Environment, error) {
	row, err := r.q(ctx).EnvironmentById(ctx, id)
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return environmentFrom(row), nil
}

func (r Repository) EnvironmentByProject(ctx context.Context, projectId string) (model.Environment, error) {
	row, err := r.q(ctx).EnvironmentByProjectId(ctx, projectId)
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment for project %s: %w", projectId, sqlcommon.TranslateError(err))
	}
	return environmentFrom(row), nil
}

// EnvironmentByTarget finds another Project bound to the same Docker target.
// A target is exclusive because managed Gateway ports and network are fixed.
func (r Repository) EnvironmentByTarget(ctx context.Context, projectId, targetType, host string, port int) (model.Environment, error) {
	row, err := r.q(ctx).EnvironmentByTarget(ctx, environmentsqlc.EnvironmentByTargetParams{ProjectId: projectId, TargetType: targetType, Column3: targetType, Host: sql.NullString{String: host, Valid: host != ""}, Port: sql.NullInt64{Int64: int64(port), Valid: port > 0}})
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment by target: %w", sqlcommon.TranslateError(err))
	}
	return environmentFrom(row), nil
}

func (r Repository) CreateEnvironment(ctx context.Context, environment model.Environment) error {
	now := time.Now().UTC()
	createdAt, updatedAt := environment.CreatedAt, environment.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	if err := r.q(ctx).CreateEnvironment(ctx, environmentsqlc.CreateEnvironmentParams{
		Id:                    environment.Id,
		ProjectId:             environment.ProjectId,
		Code:                  environment.Code,
		TargetType:            environment.TargetType,
		Platform:              environmentSSHPlatform(environment),
		Host:                  environmentSSHHost(environment),
		Port:                  environmentSSHPort(environment),
		Username:              environmentSSHUsername(environment),
		WorkspaceRoot:         environmentWorkspaceRoot(environment),
		SSHCredentialId:       environmentSSHCredentialId(environment),
		SshCredentialRevision: environmentSSHCredentialRevision(environment),
		HostKeyFingerprint:    environmentSSHHostKeyFingerprint(environment),
		TargetRevision:        environment.TargetRevision,
		LastProbeRevision:     nullableInt64(environment.LastProbeRevision),
		LastProbeStatus:       dbmodel.NullString(environment.LastProbeStatus),
		LastProbeAt:           dbmodel.NullTime(environment.LastProbeAt),
		LastProbeDiagnostic:   dbmodel.NullString(environment.LastProbeDiagnostic),
		GatewayApplicationId:  dbmodel.NullString(environment.GatewayApplicationId),
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
	}); err != nil {
		return fmt.Errorf("create environment for project %s: %w", environment.ProjectId, err)
	}
	return nil
}

func (r Repository) UpdateEnvironment(ctx context.Context, environment model.Environment) error {
	if err := r.q(ctx).UpdateEnvironment(ctx, environmentsqlc.UpdateEnvironmentParams{
		Code:                  environment.Code,
		TargetType:            environment.TargetType,
		Platform:              environmentSSHPlatform(environment),
		Host:                  environmentSSHHost(environment),
		Port:                  environmentSSHPort(environment),
		Username:              environmentSSHUsername(environment),
		WorkspaceRoot:         environmentWorkspaceRoot(environment),
		SSHCredentialId:       environmentSSHCredentialId(environment),
		SshCredentialRevision: environmentSSHCredentialRevision(environment),
		HostKeyFingerprint:    environmentSSHHostKeyFingerprint(environment),
		TargetRevision:        environment.TargetRevision,
		LastProbeRevision:     nullableInt64(environment.LastProbeRevision),
		LastProbeStatus:       dbmodel.NullString(environment.LastProbeStatus),
		LastProbeAt:           dbmodel.NullTime(environment.LastProbeAt),
		LastProbeDiagnostic:   dbmodel.NullString(environment.LastProbeDiagnostic),
		GatewayApplicationId:  dbmodel.NullString(environment.GatewayApplicationId),
		UpdatedAt:             time.Now().UTC(),
		Id:                    environment.Id,
	}); err != nil {
		return fmt.Errorf("update environment %s: %w", environment.Id, err)
	}
	return nil
}

func (r Repository) RecordProbe(ctx context.Context, environmentId string, targetRevision int64, status string, probedAt time.Time, diagnostic string) (bool, error) {
	rows, err := r.q(ctx).RecordEnvironmentProbe(ctx, environmentsqlc.RecordEnvironmentProbeParams{
		LastProbeRevision:   sql.NullInt64{Int64: targetRevision, Valid: true},
		LastProbeStatus:     sql.NullString{String: status, Valid: true},
		LastProbeAt:         sql.NullTime{Time: probedAt, Valid: true},
		LastProbeDiagnostic: sql.NullString{String: diagnostic, Valid: diagnostic != ""},
		UpdatedAt:           time.Now().UTC(),
		Id:                  environmentId,
		TargetRevision:      targetRevision,
	})
	if err != nil {
		return false, fmt.Errorf("record environment probe %s: %w", environmentId, err)
	}
	return rows == 1, nil
}
func (r Repository) BindGatewayApplication(ctx context.Context, environmentId string, gatewayApplicationId string) (bool, error) {
	rows, err := r.q(ctx).BindGatewayApplication(ctx, environmentsqlc.BindGatewayApplicationParams{
		GatewayApplicationId: sql.NullString{String: gatewayApplicationId, Valid: true},
		UpdatedAt:            time.Now().UTC(),
		Id:                   environmentId,
	})
	if err != nil {
		return false, fmt.Errorf("bind gateway application %s to environment %s: %w", gatewayApplicationId, environmentId, err)
	}
	return rows == 1, nil
}

func (r Repository) UnbindGatewayApplication(ctx context.Context, environmentId string, gatewayApplicationId string) (bool, error) {
	rows, err := r.q(ctx).UnbindGatewayApplication(ctx, environmentsqlc.UnbindGatewayApplicationParams{
		UpdatedAt:            time.Now().UTC(),
		Id:                   environmentId,
		GatewayApplicationId: sql.NullString{String: gatewayApplicationId, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("unbind gateway application %s from environment %s: %w", gatewayApplicationId, environmentId, err)
	}
	return rows == 1, nil
}

func environmentFrom(row environmentsqlc.Environment) model.Environment {
	item := model.Environment{
		Id: row.Id, ProjectId: row.ProjectId, Code: row.Code, TargetType: row.TargetType,
		WorkspaceRoot: row.WorkspaceRoot.String, TargetRevision: row.TargetRevision, LastProbeRevision: int64Ptr(row.LastProbeRevision),
		LastProbeStatus: dbmodel.StringPtr(row.LastProbeStatus), LastProbeAt: dbmodel.TimePtr(row.LastProbeAt),
		LastProbeDiagnostic: dbmodel.StringPtr(row.LastProbeDiagnostic), GatewayApplicationId: dbmodel.StringPtr(row.GatewayApplicationId),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if item.TargetType == model.EnvironmentTargetTypeSSH {
		item.SSH = &model.EnvironmentSSHTarget{
			Platform: row.Platform.String, Host: row.Host.String, Port: int(row.Port.Int64),
			Username:     row.Username.String,
			CredentialId: row.SSHCredentialId.String, CredentialRevision: row.SshCredentialRevision.Int64,
			HostKeyFingerprint: row.HostKeyFingerprint.String,
		}
	}
	return item
}

func environmentSSHPlatform(value model.Environment) sql.NullString {
	if value.SSH == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.SSH.Platform, Valid: true}
}

func environmentSSHHost(value model.Environment) sql.NullString {
	if value.SSH == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.SSH.Host, Valid: true}
}

func environmentSSHPort(value model.Environment) sql.NullInt64 {
	if value.SSH == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value.SSH.Port), Valid: true}
}

func environmentSSHUsername(value model.Environment) sql.NullString {
	if value.SSH == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.SSH.Username, Valid: true}
}

func environmentWorkspaceRoot(value model.Environment) sql.NullString {
	if value.WorkspaceRoot == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value.WorkspaceRoot, Valid: true}
}

func environmentSSHCredentialId(value model.Environment) sql.NullString {
	if value.SSH == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.SSH.CredentialId, Valid: true}
}

func environmentSSHCredentialRevision(value model.Environment) sql.NullInt64 {
	if value.SSH == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value.SSH.CredentialRevision, Valid: true}
}

func environmentSSHHostKeyFingerprint(value model.Environment) sql.NullString {
	if value.SSH == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: value.SSH.HostKeyFingerprint, Valid: value.SSH.HostKeyFingerprint != ""}
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func int64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	result := value.Int64
	return &result
}
