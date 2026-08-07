package model

import "time"

const (
	RepositoryTypeRemoteGit      = "remote_git"
	RepositoryTypeLocalDirectory = "local_directory"
)

type Repository struct {
	Id                string    `db:"id"`
	ProjectId         *string   `db:"project_id"`
	Name              string    `db:"name"`
	Code              string    `db:"code"`
	RepositoryType    string    `db:"repository_type"`
	RepositoryUrl     string    `db:"repository_url"`
	GitCredentialId   *string   `db:"git_credential_id"`
	VariableOverrides string    `db:"variable_overrides"`
	DefaultBranch     string    `db:"default_branch"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}
