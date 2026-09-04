package model

import "time"

// GatewayConfig is the configuration owned by a gateway application.
type GatewayConfig struct {
	ApplicationId           string                  `db:"application_id"`
	TraefikComponentName    string                  `db:"traefik_component_name"`
	RestApiUrl              string                  `db:"rest_api_url"`
	RestReadyTimeoutSeconds int                     `db:"rest_ready_timeout_seconds"`
	BaseDomain              string                  `db:"base_domain"`
	DefaultEntrypoint       string                  `db:"default_entrypoint"`
	TLSMode                 string                  `db:"tls_mode"`
	AcmeProfile             string                  `db:"acme_profile"`
	AcmeEmail               string                  `db:"acme_email"`
	DNSApiToken             string                  `db:"dns_api_token"`
	VersionBindings         []GatewayVersionBinding `db:"-"`
	RuntimeServiceCode      string                  `db:"-"`
	NetworkName             string                  `db:"-" json:"network_name"`
	CreatedAt               time.Time               `db:"created_at"`
	UpdatedAt               time.Time               `db:"updated_at"`
}

// GatewayVersionBinding identifies the ordinary Version for one Gateway role.
// The base role is distinct from the three ACME profile values.
type GatewayVersionBinding struct {
	Profile   string `db:"profile"`
	VersionId string `db:"version_id"`
}

func (g GatewayConfig) VersionIDForProfile(profile string) string {
	for _, binding := range g.VersionBindings {
		if binding.Profile == profile {
			return binding.VersionId
		}
	}
	return ""
}
