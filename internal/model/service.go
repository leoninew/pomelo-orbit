package model

import "time"

// Service is the runtime binding of an application instance.
type Service struct {
	Id            string    `db:"id"`
	ApplicationId string    `db:"application_id"`
	InstanceKey   string    `db:"instance_key"`
	Code          string    `db:"code"`
	VersionId     string    `db:"version_id"`
	Status        string    `db:"status"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// ServiceComponent maps one runtime service to a component declaration in the
// selected Version. Its child records are sparse overlays, never copies.
type ServiceComponent struct {
	Id                       string
	ServiceId                string
	SourceVersionComponentId string
	ComponentName            string
	Entrypoint               []string
	Command                  []string
	PullPolicy               *string
	RestartPolicy            *string
	Status                   string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Env                      []ServiceComponentEnv
	Mounts                   []ServiceComponentMount
	Resources                *ServiceComponentResources
	Endpoints                []ServiceComponentEndpoint
}

type ServiceComponentOverlayState string

const (
	ServiceComponentOverlayOverride ServiceComponentOverlayState = "override"
	ServiceComponentOverlayDeleted  ServiceComponentOverlayState = "deleted"
)

type ServiceComponentEnv struct {
	Key   string
	Value *string
	State ServiceComponentOverlayState
}

// ServiceEnv is a service-scoped environment value consumed by Version
// declarations through a complete ${NAME...} placeholder.
type ServiceEnv struct {
	Key   string
	Value string
}

type ServiceComponentMount struct {
	Target           string
	Source           *string
	SourceIsHostPath *bool
	State            ServiceComponentOverlayState
}

type ServiceComponentResources struct {
	LimitCPUs         *string
	LimitMemory       *string
	ReservationCPUs   *string
	ReservationMemory *string
	State             ServiceComponentOverlayState
}

type ServiceComponentEndpoint struct {
	Name        string
	Mode        *string
	BindAddress *string
	ListenPort  *int
	Entrypoint  *string
	PathPrefix  *string
	State       ServiceComponentOverlayState
}

// ServiceListItem is Service plus application and version labels.
type ServiceListItem struct {
	Id              string    `db:"id"`
	ApplicationId   string    `db:"application_id"`
	InstanceKey     string    `db:"instance_key"`
	Code            string    `db:"code"`
	VersionId       string    `db:"version_id"`
	Status          string    `db:"status"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	ApplicationName string    `db:"application_name"`
	ApplicationCode string    `db:"application_code"`
	ApplicationKind string    `db:"application_kind"`
	VersionLabel    string    `db:"version_label"`
}

// Service returns the base runtime binding row.
func (item ServiceListItem) Service() Service {
	return Service{
		Id:            item.Id,
		ApplicationId: item.ApplicationId,
		InstanceKey:   item.InstanceKey,
		Code:          item.Code,
		VersionId:     item.VersionId,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
