package dto

import "time"

type ServiceStatusCheckItem struct {
	ServiceId      string
	ServiceCode    string
	RecordedStatus string
	ObservedStatus string
}

type ServiceStatusCheckPreview struct {
	Token      string
	ObservedAt time.Time
	Items      []ServiceStatusCheckItem
}
