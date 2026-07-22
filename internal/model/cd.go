package model

import "time"

type Application struct {
	Id              string    `db:"id"`
	ProjectId       *string   `db:"project_id"`
	Name            string    `db:"name"`
	Code            string    `db:"code"`
	ImagePullPolicy string    `db:"image_pull_policy"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// Version is application static specification metadata (business data).
type Version struct {
	Id                   string    `db:"id"`
	ApplicationId        string    `db:"application_id"`
	Label                string    `db:"label"`
	Status               string    `db:"status"`
	EnvJSON              *string   `db:"env_json"`
	CreatedFromVersionId *string   `db:"created_from_version_id"`
	Note                 *string   `db:"note"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

// Component is a version-scoped specification unit (own table).
type Component struct {
	Id              string    `db:"id"`
	VersionId       string    `db:"version_id"`
	Name            string    `db:"name"`
	Image           string    `db:"image"`
	CommandJSON     *string   `db:"command_json"`
	ArgsJSON        *string   `db:"args_json"`
	EnvJSON         *string   `db:"env_json"`
	PortsJSON       *string   `db:"ports_json"`
	MountsJSON      *string   `db:"mounts_json"`
	NetworksJSON    *string   `db:"networks_json"`
	DependsOnJSON   *string   `db:"depends_on_json"`
	HealthcheckJSON *string   `db:"healthcheck_json"`
	ResourcesJSON   *string   `db:"resources_json"`
	PullPolicy      *string   `db:"pull_policy"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// Expose is version-scoped protocol/port exposure (no domain).
type Expose struct {
	Id            string    `db:"id"`
	VersionId     string    `db:"version_id"`
	ComponentName string    `db:"component_name"`
	Protocol      string    `db:"protocol"`
	ContainerPort int       `db:"container_port"`
	PathPrefix    *string   `db:"path_prefix"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// Environment is a project-scoped deploy target with embedded IngressPolicy.
type Environment struct {
	Id                string    `db:"id"`
	ProjectId         string    `db:"project_id"`
	Code              string    `db:"code"`
	Name              string    `db:"name"`
	Description       *string   `db:"description"`
	BaseDomain        string    `db:"base_domain"`
	DomainTemplate    *string   `db:"domain_template"`
	DefaultEntrypoint string    `db:"default_entrypoint"`
	TCPEntrypoint     *string   `db:"tcp_entrypoint"`
	TLSMode           string    `db:"tls_mode"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Service is the runtime binding of an application instance (business data).
type Service struct {
	Id                      string    `db:"id"`
	ApplicationId           string    `db:"application_id"`
	EnvironmentId           string    `db:"environment_id"`
	InstanceKey             string    `db:"instance_key"`
	IsIngress               bool      `db:"is_ingress"`
	VersionId               string    `db:"version_id"`
	LastSuccessfulVersionId *string   `db:"last_successful_version_id"`
	Status                  string    `db:"status"`
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
}

type Deployment struct {
	Id                       string     `db:"id"`
	ProjectId                *string    `db:"project_id"`
	ApplicationId            *string    `db:"application_id"`
	ApplicationName          string     `db:"application_name"`
	VersionId                *string    `db:"version_id"`
	ServiceId                *string    `db:"service_id"`
	EnvironmentId            *string    `db:"environment_id"`
	OptionsJSON              *string    `db:"options_json"`
	OperationType            string     `db:"operation_type"`
	TriggerType              string     `db:"trigger_type"`
	CommandText              string     `db:"command_text"`
	Status                   string     `db:"status"`
	StartedAt                time.Time  `db:"started_at"`
	FinishedAt               *time.Time `db:"finished_at"`
	DurationMs               *int       `db:"duration_ms"`
	LogText                  *string    `db:"log_text"`
	ErrorMessage             *string    `db:"error_message"`
	IsRollback               bool       `db:"is_rollback"`
	RollbackFromDeploymentId *string    `db:"rollback_from_deployment_id"`
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
