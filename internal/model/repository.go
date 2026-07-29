package model

import "time"

type Repository struct {
	Id                string    `db:"id"`
	ProjectId         *string   `db:"project_id"`
	Name              string    `db:"name"`
	Code              string    `db:"code"`
	RepositoryUrl     string    `db:"repository_url"`
	GitCredentialId   *string   `db:"git_credential_id"`
	VariableOverrides string    `db:"variable_overrides"`
	DefaultBranch     string    `db:"default_branch"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type RepositoryWebhook struct {
	Id              string    `db:"id"`
	RepositoryId    string    `db:"repository_id"`
	Name            string    `db:"name"`
	TemplateId      string    `db:"template_id"`
	BranchFilter    *string   `db:"branch_filter"`
	EncryptedSecret string    `db:"encrypted_secret"`
	Enabled         bool      `db:"enabled"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}
