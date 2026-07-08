package authsvc

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"

	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	UserByUsername(ctx context.Context, username string) (model.User, error)
	MarkUserLoggedIn(ctx context.Context, id string) error
	SaveLoginHistory(ctx context.Context, history model.LoginHistory) error
	ListLoginHistory(ctx context.Context, page int, perPage int, search string) (repository.Page[model.LoginHistory], error)
	UpdateUser(ctx context.Context, user model.User) error
}

type Service struct {
	repo   Repository
	tokens jwt.TokenService
	logger *slog.Logger
}

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

func New(repo Repository, tokens jwt.TokenService, logger *slog.Logger) Service {
	return Service{repo: repo, tokens: tokens, logger: logger}
}

func (s Service) Login(ctx context.Context, input LoginInput) (string, error) {
	if err := ValidateLoginInput(input); err != nil {
		return "", err
	}
	username := strings.TrimSpace(input.Username)
	user, err := s.repo.UserByUsername(ctx, username)
	if err != nil || user.Status != "enabled" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return "", ErrInvalidCredentials
	}
	if err := s.repo.MarkUserLoggedIn(ctx, user.Id); err != nil {
		s.logger.Warn("mark user login time failed", "user_id", user.Id, "error", err)
	}
	if err := s.repo.SaveLoginHistory(ctx, model.LoginHistory{Id: idutil.NewId(), UserId: user.Id, Username: user.Username, IpAddress: optionalString(input.IP), UserAgent: optionalString(input.UserAgent), LoginAt: time.Now().UTC(), Success: true}); err != nil {
		s.logger.Warn("save login history failed", "user_id", user.Id, "error", err)
	}
	return s.tokens.Sign(user.Id, user.Username)
}

func (s Service) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	if input.OldPassword == "" || len(input.NewPassword) < 6 {
		return ErrInvalidPasswordFields
	}
	if bcrypt.CompareHashAndPassword([]byte(input.User.PasswordHash), []byte(input.OldPassword)) != nil {
		return ErrInvalidCredentials
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	input.User.PasswordHash = string(passwordHash)
	return s.repo.UpdateUser(ctx, input.User)
}

func (s Service) ListLoginHistory(ctx context.Context, page int, perPage int, search string) (repository.Page[model.LoginHistory], error) {
	return s.repo.ListLoginHistory(ctx, page, perPage, search)
}

func (s Service) NewCSRFToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func ValidateLoginInput(input LoginInput) error {
	if strings.TrimSpace(input.Username) == "" || input.Password == "" || strings.TrimSpace(input.CSRFToken) == "" {
		return ErrMissingLoginFields
	}
	return nil
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

var (
	ErrMissingLoginFields    = apperror.New(apperror.KindValidation, "Missing required login fields")
	ErrInvalidPasswordFields = apperror.New(apperror.KindValidation, "Invalid password fields")
	ErrInvalidCredentials    = apperror.New(apperror.KindUnauthorized, "Invalid username or password")
)
