package repository

import "time"

type Application struct {
	Id              string    `db:"id"`
	ProjectId       *string   `db:"project_id"`
	Name            string    `db:"name"`
	Code            string    `db:"code"`
	ImagePullPolicy string    `db:"image_pull_policy"`
	Status          string    `db:"status"`
	RouteManaged    bool      `db:"route_managed"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

type Deployment struct {
	Id                       string     `db:"id"`
	ProjectId                *string    `db:"project_id"`
	ApplicationId            *string    `db:"application_id"`
	ApplicationName          string     `db:"application_name"`
	OperationType            string     `db:"operation_type"`
	TriggerType              string     `db:"trigger_type"`
	Status                   string     `db:"status"`
	StartedAt                time.Time  `db:"started_at"`
	FinishedAt               *time.Time `db:"finished_at"`
	DurationMs               *int       `db:"duration_ms"`
	LogText                  *string    `db:"log_text"`
	ErrorMessage             *string    `db:"error_message"`
	IsRollback               bool       `db:"is_rollback"`
	RollbackFromDeploymentId *string    `db:"rollback_from_deployment_id"`
}

type ApplicationConfigFile struct {
	Id            string    `db:"id"`
	ApplicationId string    `db:"application_id"`
	Path          string    `db:"path"`
	Content       string    `db:"content"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type ApplicationServiceConfig struct {
	Id            string    `db:"id"`
	ApplicationId string    `db:"application_id"`
	ServiceName   string    `db:"service_name"`
	Image         *string   `db:"image"`
	Environment   *string   `db:"environment"`
	Volumes       *string   `db:"volumes"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type ApplicationRoute struct {
	Id            string    `db:"id"`
	ApplicationId string    `db:"application_id"`
	ServiceName   string    `db:"service_name"`
	Domain        string    `db:"domain"`
	Port          int       `db:"port"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type Route struct {
	Id           string    `db:"id"`
	ProjectId    *string   `db:"project_id"`
	Name         string    `db:"name"`
	Domain       string    `db:"domain"`
	PathPrefix   string    `db:"path_prefix"`
	TargetURL    string    `db:"target_url"`
	Enabled      bool      `db:"enabled"`
	HTTPSEnabled bool      `db:"https_enabled"`
	CertPEM      *string   `db:"cert_pem"`
	CertKey      *string   `db:"cert_key"`
	CertType     string    `db:"cert_type"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
