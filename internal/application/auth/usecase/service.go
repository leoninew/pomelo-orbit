package authsvc

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	"github.com/leoninew/pomelo-orbit/internal/auth/csrf"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      repository.UserStore
	auth      repository.AuthStore
	tokens    jwt.TokenService
	logger    *slog.Logger
	grants    *mcpGrantStore
	secretKey string
}

func New(repo repository.UserStore, auth repository.AuthStore, tokens jwt.TokenService, logger *slog.Logger, secretKey string) Service {
	return Service{repo: repo, auth: auth, tokens: tokens, logger: logger, grants: newMCPGrantStore(), secretKey: secretKey}
}

func (s Service) Login(ctx context.Context, input authdto.LoginInput) (string, error) {
	if err := ValidateLoginInput(input); err != nil {
		return "", err
	}
	if err := csrf.Verify(s.secretKey, input.CSRFToken); err != nil {
		return "", ErrInvalidCSRFToken
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	user, err := s.repo.UserByEmail(ctx, email)
	if err != nil || user.Status != "enabled" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return "", ErrInvalidCredentials
	}
	if err := s.repo.MarkUserLoggedIn(ctx, user.Id); err != nil {
		s.logger.Warn("mark user login time failed", "user_id", user.Id, "error", err)
	}
	if err := s.auth.SaveLoginHistory(ctx, model.LoginHistory{Id: idutil.NewId(), UserId: user.Id, Username: user.Username, IpAddress: optionalString(input.IP), UserAgent: optionalString(input.UserAgent), LoginAt: time.Now().UTC(), Success: true}); err != nil {
		s.logger.Warn("save login history failed", "user_id", user.Id, "error", err)
	}
	return s.tokens.Sign(user.Id, user.Username)
}

func (s Service) Authenticate(ctx context.Context, rawJwt string) (authdto.AuthenticatedUser, error) {
	claims, err := s.tokens.Verify(strings.TrimSpace(rawJwt))
	if err != nil {
		return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "Invalid token")
	}
	user, err := s.repo.UserById(ctx, claims.Sub)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "Invalid token")
		}
		return authdto.AuthenticatedUser{}, err
	}
	if user.Status != "enabled" {
		return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "Invalid token")
	}
	roles, err := s.repo.UserRoles(ctx, user.Id)
	if err != nil {
		return authdto.AuthenticatedUser{}, err
	}
	permissions, err := s.repo.UserPermissions(ctx, user.Id)
	if err != nil {
		return authdto.AuthenticatedUser{}, err
	}
	return authdto.AuthenticatedUser{User: user, Roles: emptyStrings(roles), Permissions: emptyStrings(permissions)}, nil
}

func (s Service) UserRoles(ctx context.Context, userId string) ([]string, error) {
	return s.repo.UserRoles(ctx, userId)
}

func (s Service) UserPermissions(ctx context.Context, userId string) ([]string, error) {
	return s.repo.UserPermissions(ctx, userId)
}

func (s Service) ChangePassword(ctx context.Context, input authdto.ChangePasswordInput) error {
	if input.OldPassword == "" || len(input.NewPassword) < 6 || len(input.NewPassword) > 36 {
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
	return s.auth.ListLoginHistory(ctx, page, perPage, search)
}

func (s Service) NewCSRFToken() (string, error) {
	return csrf.Issue(s.secretKey)
}

func ValidateLoginInput(input authdto.LoginInput) error {
	if strings.TrimSpace(input.Email) == "" || input.Password == "" || strings.TrimSpace(input.CSRFToken) == "" {
		return ErrMissingLoginFields
	}
	return nil
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

var (
	ErrMissingLoginFields    = apperror.New(apperror.KindValidation, "Missing required login fields")
	ErrInvalidCSRFToken      = apperror.New(apperror.KindValidation, "Request token is invalid or expired, please refresh the page")
	ErrInvalidPasswordFields = apperror.New(apperror.KindValidation, "Invalid password fields")
	ErrInvalidCredentials    = apperror.New(apperror.KindUnauthorized, "Invalid email or password")
)
