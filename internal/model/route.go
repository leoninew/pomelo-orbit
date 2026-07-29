package model

import "time"

type Route struct {
	Id           string    `db:"id"`
	ProjectId    *string   `db:"project_id"`
	Name         string    `db:"name"`
	Domain       string    `db:"domain"`
	PathPrefix   string    `db:"path_prefix"`
	TargetUrl    string    `db:"target_url"`
	Enabled      bool      `db:"enabled"`
	HTTPSEnabled bool      `db:"https_enabled"`
	CertPEM      *string   `db:"cert_pem"`
	CertKey      *string   `db:"cert_key"`
	CertType     string    `db:"cert_type"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
