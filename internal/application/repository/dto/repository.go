package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

type RepositoryCreateInput struct {
	ProjectId         string
	Name              string
	Code              string
	RepositoryType    string
	RepositoryUrl     string
	GitCredentialId   *string
	VariableOverrides []map[string]any
	DefaultBranch     string
}
type RepositoryUpdateInput struct {
	Name              *string
	RepositoryType    *string
	RepositoryUrl     *string
	GitCredentialId   *string
	VariableOverrides *[]map[string]any
	DefaultBranch     *string
}
type RepositoryDetail struct {
	Repository           model.Repository
	GitCredentialName    *string
	VariableDeclarations []map[string]any
}
