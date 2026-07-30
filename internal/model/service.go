package model

import "time"

// Service is the runtime binding of an application instance.
type Service struct {
	Id            string `db:"id"`
	ApplicationId string `db:"application_id"`
	InstanceKey   string `db:"instance_key"`
	VersionId     string `db:"version_id"`
	RuntimeConfig map[string]string
	Status        string    `db:"status"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// ServiceExpose is the persisted runtime routing and port binding of a service.
type ServiceExpose struct {
	Id            string    `db:"id"`
	ServiceId     string    `db:"service_id"`
	ComponentName string    `db:"component_name"`
	Protocol      string    `db:"protocol"`
	ContainerPort int       `db:"container_port"`
	PathPrefix    *string   `db:"path_prefix"`
	Access        string    `db:"access"`
	ListenPort    *int      `db:"listen_port"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// ServiceListItem is Service plus application and version labels.
type ServiceListItem struct {
	Id              string `db:"id"`
	ApplicationId   string `db:"application_id"`
	InstanceKey     string `db:"instance_key"`
	VersionId       string `db:"version_id"`
	RuntimeConfig   map[string]string
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
		VersionId:     item.VersionId,
		RuntimeConfig: item.RuntimeConfig,
		Status:        item.Status,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
