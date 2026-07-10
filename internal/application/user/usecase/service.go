package usersvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo repository.UserStore
}

func New(repo repository.UserStore) Service {
	return Service{repo: repo}
}

func (s Service) Create(ctx context.Context, input userdto.CreateInput) (model.User, error) {
	username := strings.TrimSpace(input.Username)
	email := normalizeEmail(input.Email)
	if username == "" || len(username) > 50 || len(input.Password) < 6 || len(input.Password) > 255 || (email != nil && len(*email) > 255) {
		return model.User{}, ErrInvalidUserFields
	}
	if _, err := s.repo.UserByUsername(ctx, username); err == nil {
		return model.User{}, apperror.New(apperror.KindConflict, "Username "+username+" already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, err
	}
	if email != nil {
		if _, err := s.repo.UserByEmail(ctx, *email); err == nil {
			return model.User{}, apperror.New(apperror.KindConflict, "Email "+*email+" already exists")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return model.User{}, err
		}
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	now := time.Now().UTC()
	user := model.User{Id: idutil.NewId(), Username: username, PasswordHash: string(passwordHash), Status: "enabled", OAuthProvider: "", OAuthProviderId: "", Email: email, AuthSource: "password", CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (s Service) Update(ctx context.Context, input userdto.UpdateInput) (model.User, error) {
	user := input.User
	if input.Username != nil {
		username := strings.TrimSpace(*input.Username)
		if username == "" || len(username) > 50 {
			return model.User{}, ErrInvalidUserFields
		}
		user.Username = username
	}
	if input.Password != nil {
		password := *input.Password
		if password != "" {
			if len(password) < 6 || len(password) > 255 {
				return model.User{}, ErrInvalidUserFields
			}
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return model.User{}, err
			}
			user.PasswordHash = string(passwordHash)
		}
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if status != "enabled" && status != "disabled" {
			return model.User{}, ErrInvalidUserStatus
		}
		user.Status = status
	}
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (s Service) SetStatus(ctx context.Context, userId string, status string) error {
	if status != "enabled" && status != "disabled" {
		return ErrInvalidUserStatus
	}
	return s.repo.SetUserStatus(ctx, userId, status)
}

func (s Service) Delete(ctx context.Context, userId string) error {
	return s.repo.DeleteUser(ctx, userId)
}

func normalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*email)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

var (
	ErrInvalidUserFields = apperror.New(apperror.KindValidation, "Invalid user fields")
	ErrInvalidUserStatus = apperror.New(apperror.KindValidation, "Invalid user status")
)
