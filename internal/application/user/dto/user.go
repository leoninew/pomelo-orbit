package dto

import "gitee.com/leoninew/PomeloOrbit-go/internal/model"

type CreateInput struct {
	Username string
	Password string
	Email    *string
}

type UpdateInput struct {
	User     model.User
	Username *string
	Password *string
	Status   *string
}
