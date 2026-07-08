package authz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type Store interface {
	UserById(ctx context.Context, id string) (model.User, error)
	UserPermissions(ctx context.Context, userId string) ([]string, error)
}

type CurrentUserContext struct {
	User        model.User
	Permissions []string
}

type Authenticator struct {
	logger *slog.Logger
	store  Store
	tokens jwt.TokenService
}

func New(logger *slog.Logger, store Store, tokens jwt.TokenService) Authenticator {
	return Authenticator{logger: logger, store: store, tokens: tokens}
}

func (a Authenticator) CurrentUser(c *gin.Context) (model.User, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Not authenticated"})
		return model.User{}, false
	}
	claims, err := a.tokens.Verify(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token"})
		return model.User{}, false
	}
	user, err := a.store.UserById(c.Request.Context(), claims.Sub)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token"})
			return model.User{}, false
		}
		a.logger.Error("load current user failed", "user_id", claims.Sub, "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"detail": "Authentication service unavailable"})
		return model.User{}, false
	}
	if user.Status != "enabled" {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token"})
		return model.User{}, false
	}
	return user, true
}

func (a Authenticator) RequirePermission(c *gin.Context, permission string) (CurrentUserContext, bool) {
	user, ok := a.CurrentUser(c)
	if !ok {
		return CurrentUserContext{}, false
	}
	permissions, err := a.store.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		a.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to load user permissions"})
		return CurrentUserContext{}, false
	}
	if HasPermission(permissions, permission) {
		return CurrentUserContext{User: user, Permissions: permissions}, true
	}
	c.JSON(http.StatusForbidden, gin.H{"detail": "Permission denied"})
	return CurrentUserContext{}, false
}

func HasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}
