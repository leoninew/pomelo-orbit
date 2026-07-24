package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

const CredentialExportVersion = "1.0"

type CredentialCreateInput struct {
	ProjectId string
	Name      string
	Type      string
	Data      string
}

type CredentialUpdateInput struct {
	Name *string
	Data *string
}

type CredentialExport struct {
	Version string
	Name    string
	Type    string
	Data    string
}

type CredentialDetail struct {
	Credential model.Credential
	Data       string
}
