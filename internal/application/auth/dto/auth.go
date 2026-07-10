package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type LoginInput struct {
	Username  string
	Password  string
	CSRFToken string
	IP        string
	UserAgent string
}

type ChangePasswordInput struct {
	User        model.User
	OldPassword string
	NewPassword string
}

// AuthenticatedUser is the enabled account resolved from a verified bearer token.
type AuthenticatedUser struct {
	User        model.User
	Roles       []string
	Permissions []string
}
