package model

import "time"

type Route struct {
	Id                    string    `db:"id"`
	ProjectId             *string   `db:"project_id"`
	Name                  string    `db:"name"`
	Protocol              string    `db:"protocol"`
	Domain                string    `db:"domain"`
	PathPrefix            string    `db:"path_prefix"`
	TargetUrl             string    `db:"target_url"`
	ListenPort            *int      `db:"listen_port"`
	ServiceId             *string   `db:"service_id"`
	ComponentName         *string   `db:"component_name"`
	EndpointProtocol      *string   `db:"endpoint_protocol"`
	EndpointContainerPort *int      `db:"endpoint_container_port"`
	TargetAddress         string    `db:"-"`
	TargetPort            int       `db:"-"`
	Enabled               bool      `db:"enabled"`
	HTTPSEnabled          bool      `db:"https_enabled"`
	CertPEM               *string   `db:"cert_pem"`
	CertKey               *string   `db:"cert_key"`
	CertType              string    `db:"cert_type"`
	AcmeChallenge         string    `db:"acme_challenge"`
	GatewayApplicationId  string    `db:"-"`
	HTTP01Available       bool      `db:"-"`
	DNS01Available        bool      `db:"-"`
	ACMEChallengeHint     string    `db:"-"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}
