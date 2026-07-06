package authz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"backend/internal/repository/model"
	authsvc "backend/internal/service/auth"
	transportresponse "backend/internal/transport/http/response"
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
	tokens authsvc.TokenService
}

func New(logger *slog.Logger, store Store, tokens authsvc.TokenService) Authenticator {
	return Authenticator{logger: logger, store: store, tokens: tokens}
}

func (a Authenticator) CurrentUser(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		transportresponse.Error(a.logger, w, http.StatusUnauthorized, "Not authenticated")
		return model.User{}, false
	}
	claims, err := a.tokens.Verify(token)
	if err != nil {
		transportresponse.Error(a.logger, w, http.StatusUnauthorized, "Invalid token")
		return model.User{}, false
	}
	user, err := a.store.UserById(r.Context(), claims.Sub)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			transportresponse.Error(a.logger, w, http.StatusUnauthorized, "Invalid token")
			return model.User{}, false
		}
		a.logger.Error("load current user failed", "user_id", claims.Sub, "error", err)
		transportresponse.Error(a.logger, w, http.StatusServiceUnavailable, "Authentication service unavailable")
		return model.User{}, false
	}
	if user.Status != "enabled" {
		transportresponse.Error(a.logger, w, http.StatusUnauthorized, "Invalid token")
		return model.User{}, false
	}
	return user, true
}

func (a Authenticator) RequirePermission(w http.ResponseWriter, r *http.Request, permission string) (CurrentUserContext, bool) {
	user, ok := a.CurrentUser(w, r)
	if !ok {
		return CurrentUserContext{}, false
	}
	permissions, err := a.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		a.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.Error(a.logger, w, http.StatusInternalServerError, "Failed to load user permissions")
		return CurrentUserContext{}, false
	}
	if HasPermission(permissions, permission) {
		return CurrentUserContext{User: user, Permissions: permissions}, true
	}
	transportresponse.Error(a.logger, w, http.StatusForbidden, "Permission denied")
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
