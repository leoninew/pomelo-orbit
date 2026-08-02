package model

import "time"

// GatewayConfig is the configuration owned by a gateway application.
type GatewayConfig struct {
	ApplicationId     string    `db:"application_id"`
	RestApiUrl        string    `db:"rest_api_url"`
	BaseDomain        string    `db:"base_domain"`
	DefaultEntrypoint string    `db:"default_entrypoint"`
	TLSMode           string    `db:"tls_mode"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
