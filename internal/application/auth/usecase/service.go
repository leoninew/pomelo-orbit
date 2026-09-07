package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	secretKey string
}

const (
	mcpAccessTokenPrefix      = "orbit_mcp_pat_"
	mcpAccessTokenEntropySize = 32
	maxMCPAccessTokenNameLen  = 100
	maxMCPAccessTokenLifetime = 3650
)

func New(repo repository.UserStore, auth repository.AuthStore, tokens jwt.TokenService, logger *slog.Logger, secretKey string) Service {
	return Service{repo: repo, auth: auth, tokens: tokens, logger: logger, secretKey: secretKey}
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
	return s.authenticateUser(ctx, claims.Sub)
}

// AuthenticateMCPAccessToken resolves a local stdio PAT and verifies that its
// owner remains enabled before an MCP tool call can proceed.
func (s Service) AuthenticateMCPAccessToken(ctx context.Context, rawToken string) (authdto.AuthenticatedUser, error) {
	tokenHash := hashMCPAccessToken(rawToken)
	if tokenHash == "" {
		return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "Invalid MCP access token")
	}
	token, err := s.auth.MCPAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "Invalid MCP access token")
		}
		return authdto.AuthenticatedUser{}, err
	}
	if token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now().UTC()) {
		return authdto.AuthenticatedUser{}, apperror.New(apperror.KindUnauthorized, "MCP access token has expired")
	}
	return s.authenticateUser(ctx, token.UserId)
}

func (s Service) CreateMCPAccessToken(ctx context.Context, userId string, input authdto.MCPAccessTokenCreateInput) (authdto.MCPAccessTokenCreated, error) {
	userId = strings.TrimSpace(userId)
	name := strings.TrimSpace(input.Name)
	if userId == "" || name == "" || len(name) > maxMCPAccessTokenNameLen || input.ExpiresInDays < 0 || input.ExpiresInDays > maxMCPAccessTokenLifetime {
		return authdto.MCPAccessTokenCreated{}, apperror.New(apperror.KindValidation, "Invalid MCP access token fields")
	}
	if _, err := s.authenticateUser(ctx, userId); err != nil {
		return authdto.MCPAccessTokenCreated{}, err
	}

	rawToken, err := newMCPAccessToken()
	if err != nil {
		return authdto.MCPAccessTokenCreated{}, apperror.Wrap(apperror.KindInternal, "Failed to create MCP access token", err)
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if input.ExpiresInDays > 0 {
		expires := now.AddDate(0, 0, int(input.ExpiresInDays))
		expiresAt = &expires
	}
	accessToken := model.MCPAccessToken{Id: idutil.NewId(), UserId: userId, Name: name, TokenHash: hashMCPAccessToken(rawToken), ExpiresAt: expiresAt, CreatedAt: now}
	if err := s.auth.CreateMCPAccessToken(ctx, accessToken); err != nil {
		return authdto.MCPAccessTokenCreated{}, apperror.Wrap(apperror.KindInternal, "Failed to create MCP access token", err)
	}
	return authdto.MCPAccessTokenCreated{AccessToken: accessToken, Token: rawToken}, nil
}

func (s Service) ListMCPAccessTokens(ctx context.Context, userId string) ([]model.MCPAccessToken, error) {
	items, err := s.auth.ListMCPAccessTokens(ctx, strings.TrimSpace(userId))
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list MCP access tokens", err)
	}
	return items, nil
}

func (s Service) RevokeMCPAccessToken(ctx context.Context, userId string, tokenId string) error {
	tokenId = strings.TrimSpace(tokenId)
	_, err := s.auth.MCPAccessTokenForUser(ctx, strings.TrimSpace(userId), tokenId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "MCP access token "+tokenId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load MCP access token", err)
	}
	if err := s.auth.DeleteMCPAccessToken(ctx, tokenId); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to revoke MCP access token", err)
	}
	return nil
}

func (s Service) authenticateUser(ctx context.Context, userId string) (authdto.AuthenticatedUser, error) {
	user, err := s.repo.UserById(ctx, userId)
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

func newMCPAccessToken() (string, error) {
	bytes := make([]byte, mcpAccessTokenEntropySize)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("read token entropy: %w", err)
	}
	return mcpAccessTokenPrefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashMCPAccessToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
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
