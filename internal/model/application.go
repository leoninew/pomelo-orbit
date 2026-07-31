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
	CreatedFromVersionId *string   `db:"created_from_version_id"`
	Note                 *string   `db:"note"`
	ComponentSummary     string    `db:"component_summary"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

// VersionComponentSummary returns the stable, user-facing component name list for a version.
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
		parts = append(parts, component.Name)
	}
	return strings.Join(parts, ", ")
}

// VersionComponent is a version-scoped specification unit (own table).
type VersionComponent struct {
	Id            string `db:"id"`
	VersionId     string `db:"version_id"`
	Name          string `db:"name"`
	Image         string `db:"image"`
	Command       []string
	Env           []VersionComponentEnv
	Ports         []VersionComponentPort
	Mounts        []VersionComponentMount
	Dependencies  []VersionComponentDependency
	Healthcheck   *VersionComponentHealthcheck
	Resources     *VersionComponentResources
	PullPolicy    string  `db:"pull_policy"`
	RestartPolicy *string `db:"restart_policy"`
	Tmpfs         []VersionComponentTmpfs
	Ulimits       []VersionComponentUlimit
	Devices       []VersionComponentDeviceRequest
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type VersionComponentEnv struct {
	Key   string
	Value string
}

type VersionComponentPort struct {
	HostPort      int
	ContainerPort int
}

type VersionComponentMount struct {
	SourceType       string
	Source           string
	Target           string
	ReadOnly         bool
	SourceIsHostPath bool
	Content          string
	Mode             string
	IgnoreIfExists   bool
}

type VersionComponentDependency struct {
	Name      string
	Condition string
}

type VersionComponentHealthcheck struct {
	TestMode      string
	Test          string
	Interval      *string
	Timeout       *string
	Retries       *int
	StartPeriod   *string
	StartInterval *string
	Disabled      bool
}

type VersionComponentResources struct {
	LimitCPUs         *string
	LimitMemory       *string
	ReservationCPUs   *string
	ReservationMemory *string
}

type VersionComponentTmpfs struct {
	Target    string
	SizeBytes int64
	Mode      string
}

type VersionComponentUlimit struct {
	Name string
	Soft int64
	Hard int64
}

// VersionComponentDeviceRequest maps to Compose deploy.resources.reservations.devices.
type VersionComponentDeviceRequest struct {
	Driver       string
	Count        string
	Capabilities []string
}
