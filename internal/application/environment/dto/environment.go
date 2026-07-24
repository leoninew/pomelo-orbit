package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type EnvironmentCreateInput struct {
	ProjectId   string
	Code        string
	Name        string
	Description *string
}

type EnvironmentUpdateInput struct {
	Name        *string
	Description *string
}

type EnvironmentView struct {
	Environment model.Environment
}
