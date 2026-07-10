package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type SaveInput struct {
	Role            model.Role
	Code            string
	Name            string
	Description     *string
	PermissionCodes []string
}

type Detail struct {
	Role            model.Role
	PermissionCodes []string
}

type Permission struct {
	Id          string
	Code        string
	Name        string
	Description *string
}
