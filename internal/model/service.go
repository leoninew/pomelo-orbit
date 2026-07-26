package model

import "time"

// Service is the runtime binding of an application instance.
type Service struct {
	Id                      string    `db:"id"`
	ApplicationId           string    `db:"application_id"`
	InstanceKey             string    `db:"instance_key"`
	VersionId               string    `db:"version_id"`
	LastSuccessfulVersionId *string   `db:"last_successful_version_id"`
	Status                  string    `db:"status"`
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
}

// ServiceListItem is Service plus application and version labels.
type ServiceListItem struct {
	Id                         string    `db:"id"`
	ApplicationId              string    `db:"application_id"`
	InstanceKey                string    `db:"instance_key"`
	VersionId                  string    `db:"version_id"`
	LastSuccessfulVersionId    *string   `db:"last_successful_version_id"`
	Status                     string    `db:"status"`
	CreatedAt                  time.Time `db:"created_at"`
	UpdatedAt                  time.Time `db:"updated_at"`
	ApplicationName            string    `db:"application_name"`
	ApplicationCode            string    `db:"application_code"`
	ApplicationKind            string    `db:"application_kind"`
	VersionLabel               string    `db:"version_label"`
	LastSuccessfulVersionLabel *string   `db:"last_successful_version_label"`
}

// Service returns the base runtime binding row.
func (item ServiceListItem) Service() Service {
	return Service{
		Id:                      item.Id,
		ApplicationId:           item.ApplicationId,
		InstanceKey:             item.InstanceKey,
		VersionId:               item.VersionId,
		LastSuccessfulVersionId: item.LastSuccessfulVersionId,
		Status:                  item.Status,
		CreatedAt:               item.CreatedAt,
		UpdatedAt:               item.UpdatedAt,
	}
}
