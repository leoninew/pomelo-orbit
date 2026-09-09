package model

import "time"

type Deployment struct {
	Id                        string     `db:"id"`
	ProjectId                 *string    `db:"project_id"`
	ApplicationId             *string    `db:"application_id"`
	ApplicationName           string     `db:"application_name"`
	VersionId                 *string    `db:"version_id"`
	ServiceId                 *string    `db:"service_id"`
	ServiceInstanceKey        *string    `db:"service_instance_key"`
	EnvironmentId             *string    `db:"environment_id"`
	EnvironmentTargetType     *string    `db:"environment_target_type"`
	EnvironmentTargetRevision *int64     `db:"environment_target_revision"`
	SSHCredentialId           *string    `db:"ssh_credential_id"`
	SSHCredentialRevision     *int64     `db:"ssh_credential_revision"`
	GatewayApplicationId      *string    `db:"gateway_application_id"`
	OptionsJSON               *string    `db:"options_json"`
	OperationType             string     `db:"operation_type"`
	TriggerType               string     `db:"trigger_type"`
	CommandText               string     `db:"command_text"`
	Status                    string     `db:"status"`
	StartedAt                 time.Time  `db:"started_at"`
	FinishedAt                *time.Time `db:"finished_at"`
	DurationMs                *int       `db:"duration_ms"`
	LogText                   *string    `db:"log_text"`
	ErrorMessage              *string    `db:"error_message"`
	IsRollback                bool       `db:"is_rollback"`
	RollbackFromDeploymentId  *string    `db:"rollback_from_deployment_id"`
	EffectivePlanHash         *string    `db:"effective_plan_hash"`
}
