package dto

import "github.com/leoninew/pomelo-orbit/internal/model"

type CreateInput struct {
	Username string
	Password string
	Email    string
}

type UpdateInput struct {
	Username *string
	Password *string
	Status   *string
	Email    *string
}

// Actor identifies the account making an actor-sensitive user change.
type Actor struct {
	UserId      string
	Permissions []string
}

type ListItem struct {
	User  model.User
	Roles []model.Role
}

type Detail struct {
	User                model.User
	Roles               []model.Role
	Permissions         []string
	PermissionsByRoleId map[string][]string
}
