package model

import (
	"sort"
	"strings"
	"time"
)

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
	ComponentSummary     string    `db:"component_summary"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

// VersionComponentSummary returns the stable, user-facing image list for a version.
func VersionComponentSummary(components []VersionComponent) string {
	if len(components) == 0 {
		return ""
	}
	items := append([]VersionComponent(nil), components...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	parts := make([]string, 0, len(items))
	for _, component := range items {
		parts = append(parts, component.Image)
	}
	return strings.Join(parts, ", ")
}

// VersionComponent is a version-scoped specification unit (own table).
type VersionComponent struct {
	Id              string  `db:"id"`
	VersionId       string  `db:"version_id"`
	Name            string  `db:"name"`
	Image           string  `db:"image"`
	CommandJSON     *string `db:"command_json"`
	ArgsJSON        *string `db:"args_json"`
	EnvJSON         *string `db:"env_json"`
	PortsJSON       *string `db:"ports_json"`
	MountsJSON      *string `db:"mounts_json"`
	NetworksJSON    *string `db:"networks_json"`
	DependsOnJSON   *string `db:"depends_on_json"`
	HealthcheckJSON *string `db:"healthcheck_json"`
	ResourcesJSON   *string `db:"resources_json"`
	PullPolicy      *string `db:"pull_policy"`
	RestartPolicy   *string `db:"restart_policy"`
	TmpfsJSON       *string `db:"tmpfs_json"`
	UlimitsJSON     *string `db:"ulimits_json"`
	SecretEnvRefs   []VersionComponentSecretEnvRef
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// VersionComponentSecretEnvRef identifies one runtime_env credential value.
// Credential data is deliberately never part of the Version model.
type VersionComponentSecretEnvRef struct {
	ComponentId  string `db:"component_id"`
	EnvKey       string `db:"env_key"`
	CredentialId string `db:"credential_id"`
	DataKey      string `db:"data_key"`
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
