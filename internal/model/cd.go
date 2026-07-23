package model

import "time"

type Application struct {
	Id              string    `db:"id"`
	ProjectId       *string   `db:"project_id"`
	Name            string    `db:"name"`
	Code            string    `db:"code"`
	Kind            string    `db:"kind"`
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

// VersionComponent is a version-scoped specification unit (own table).
type VersionComponent struct {
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

// VersionExpose is version-scoped protocol/port exposure (no domain).
type VersionExpose struct {
	Id            string    `db:"id"`
	VersionId     string    `db:"version_id"`
	ComponentName string    `db:"component_name"`
	Protocol      string    `db:"protocol"`
	ContainerPort int       `db:"container_port"`
	PathPrefix    *string   `db:"path_prefix"`
	Access        string    `db:"access"`      // local | public
	ListenPort    *int      `db:"listen_port"` // nil/0 → container_port
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// Environment is a project-scoped deploy target (metadata only; ingress is on Gateway).
type Environment struct {
	Id          string    `db:"id"`
	ProjectId   string    `db:"project_id"`
	Code        string    `db:"code"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// GatewayConfig is 1:1 with Application(kind=gateway): rest URL, base_domain, entrypoint/TLS SoT.
type GatewayConfig struct {
	ApplicationId     string    `db:"application_id"`
	RestApiUrl        string    `db:"rest_api_url"`
	BaseDomain        string    `db:"base_domain"`
	Image             *string   `db:"image"`
	DefaultEntrypoint string    `db:"default_entrypoint"`
	TLSMode           string    `db:"tls_mode"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Service is the runtime binding of an application instance (business data).
// Identity: application + environment + instance_key; bound version drives runtime.
type Service struct {
	Id                      string    `db:"id"`
	ApplicationId           string    `db:"application_id"`
	EnvironmentId           string    `db:"environment_id"`
	InstanceKey             string    `db:"instance_key"`
	VersionId               string    `db:"version_id"`
	LastSuccessfulVersionId *string   `db:"last_successful_version_id"`
	Status                  string    `db:"status"`
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
}

// ServiceListItem is Service plus joined application / environment / version labels.
type ServiceListItem struct {
	Id                         string    `db:"id"`
	ApplicationId              string    `db:"application_id"`
	EnvironmentId              string    `db:"environment_id"`
	InstanceKey                string    `db:"instance_key"`
	VersionId                  string    `db:"version_id"`
	LastSuccessfulVersionId    *string   `db:"last_successful_version_id"`
	Status                     string    `db:"status"`
	CreatedAt                  time.Time `db:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at"`
	ApplicationName            string    `db:"application_name"`
	ApplicationCode            string    `db:"application_code"`
	ApplicationKind            string    `db:"application_kind"`
	EnvironmentName            string    `db:"environment_name"`
	EnvironmentCode            string    `db:"environment_code"`
	VersionLabel               string    `db:"version_label"`
	LastSuccessfulVersionLabel *string   `db:"last_successful_version_label"`
}

// Service returns the base runtime binding row.
func (item ServiceListItem) Service() Service {
	return Service{
		Id:                      item.Id,
		ApplicationId:           item.ApplicationId,
		EnvironmentId:           item.EnvironmentId,
		InstanceKey:             item.InstanceKey,
		VersionId:               item.VersionId,
		LastSuccessfulVersionId: item.LastSuccessfulVersionId,
		Status:                  item.Status,
		CreatedAt:               item.CreatedAt,
		UpdatedAt:               item.UpdatedAt,
	}
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
