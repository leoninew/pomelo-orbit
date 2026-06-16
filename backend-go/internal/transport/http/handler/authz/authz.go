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

type CurrentUserResp struct {
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
		transportresponse.JSON(w, http.StatusUnauthorized, map[string]string{"detail": "Not authenticated"})
		return model.User{}, false
	}
	claims, err := a.tokens.Verify(token)
	if err != nil {
		transportresponse.JSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid token"})
		return model.User{}, false
	}
	user, err := a.store.UserById(r.Context(), claims.Sub)
	if err != nil || user.Status != "enabled" {
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			a.logger.Warn("load current user failed", "user_id", claims.Sub, "error", err)
		}
		transportresponse.JSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid token"})
		return model.User{}, false
	}
	return user, true
}

func (a Authenticator) RequirePermission(w http.ResponseWriter, r *http.Request, permission string) (CurrentUserResp, bool) {
	user, ok := a.CurrentUser(w, r)
	if !ok {
		return CurrentUserResp{}, false
	}
	permissions, err := a.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		a.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user permissions"})
		return CurrentUserResp{}, false
	}
	if HasPermission(permissions, permission) {
		return CurrentUserResp{User: user, Permissions: permissions}, true
	}
	transportresponse.JSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
	return CurrentUserResp{}, false
}

func HasPermission(permissions []string, permission string) bool {
	for _, item := range permissions {
		if item == permission {
			return true
		}
	}
	return false
}
