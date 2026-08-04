package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

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

type WebhookCreateInput struct {
	Name         string
	TemplateId   string
	Secret       string
	BranchFilter *string
}

type WebhookUpdateInput struct {
	Name         *string
	TemplateId   *string
	Secret       *string
	BranchFilter *string
	BranchSet    bool
	Enabled      *bool
}

type WebhookReceiveInput struct {
	WebhookId string
	Headers   map[string]string
	Payload   []byte
}

type WebhookReceiveResult struct {
	Status string
	Reason string
	RunId  string
}
