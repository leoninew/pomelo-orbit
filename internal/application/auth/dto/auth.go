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
