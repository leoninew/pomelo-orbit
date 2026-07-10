package authz

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type CurrentUserContext struct {
	User        model.User
	Permissions []string
}

type Authenticator struct {
	logger  *slog.Logger
	service authsvc.Service
}

func New(logger *slog.Logger, service authsvc.Service) Authenticator {
	return Authenticator{logger: logger, service: service}
}

func (a Authenticator) CurrentUser(c *gin.Context) (model.User, bool) {
	token := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if token == "" {
		transportresponse.Error(c, http.StatusUnauthorized, "Not authenticated")
		return model.User{}, false
	}
	authenticated, err := a.service.Authenticate(c.Request.Context(), token)
	if err != nil {
		if apperror.IsKind(err, apperror.KindUnauthorized) {
			transportresponse.Error(c, http.StatusUnauthorized, "Invalid token")
			return model.User{}, false
		}
		a.logger.Error("load current user failed", "error", err)
		transportresponse.Error(c, http.StatusServiceUnavailable, "Authentication service unavailable")
		return model.User{}, false
	}
	return authenticated.User, true
}

func (a Authenticator) RequirePermission(c *gin.Context, permission string) (CurrentUserContext, bool) {
	user, ok := a.CurrentUser(c)
	if !ok {
		return CurrentUserContext{}, false
	}
	permissions, err := a.service.UserPermissions(c.Request.Context(), user.Id)
	if err != nil {
		a.logger.Error("load current user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to load user permissions")
		return CurrentUserContext{}, false
	}
	if HasPermission(permissions, permission) {
		return CurrentUserContext{User: user, Permissions: permissions}, true
	}
	transportresponse.Error(c, http.StatusForbidden, "Permission denied")
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
