package authz

import "backend/internal/repository/model"

type CurrentUserContext struct {
	User        model.User
	Permissions []string
}
