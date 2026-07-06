package userhandler

import (
	"context"
	"database/sql"
	"errors"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitee.com/leoninew/PomeloOrbit-go/internal/apperror"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/user"
	"gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

type Store interface {
	ListUsers(ctx context.Context, page int, perPage int, search string) (repository.Page[model.User], error)
	UserById(ctx context.Context, id string) (model.User, error)
	UserRolesByUserIds(ctx context.Context, userIds []string) (map[string][]model.Role, error)
	UserRoleDetails(ctx context.Context, userId string) ([]model.Role, error)
	UserPermissions(ctx context.Context, userId string) ([]string, error)
	SetUserRoles(ctx context.Context, userId string, roleIds []string) error
}

type RoleStore interface {
	RoleById(ctx context.Context, id string) (model.Role, error)
	RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error)
}

type Handler struct {
	logger        *slog.Logger
	service       usersvc.Service
	authenticator authz.Authenticator
	store         Store
	roleStore     RoleStore
}

func New(logger *slog.Logger, service usersvc.Service, authenticator authz.Authenticator, store Store, roleStore RoleStore) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator, store: store, roleStore: roleStore}
}

func (h Handler) Register(r router) {
	r.Get("/api/user", h.listUsers)
	r.Post("/api/user", h.createUser)
	r.Get("/api/user/{user_id}", h.getUser)
	r.Put("/api/user/{user_id}", h.updateUser)
	r.Put("/api/user/{user_id}/role", h.updateUserRoles)
	r.Post("/api/user/{user_id}/disable", h.disableUser)
	r.Post("/api/user/{user_id}/enable", h.enableUser)
	r.Delete("/api/user/{user_id}", h.deleteUser)
}

func (h Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "user:read"); !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	users, err := h.store.ListUsers(r.Context(), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list users failed", "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to list users")
		return
	}
	userIds := make([]string, 0, len(users.Items))
	for _, user := range users.Items {
		userIds = append(userIds, user.Id)
	}
	rolesByUserId, err := h.store.UserRolesByUserIds(r.Context(), userIds)
	if err != nil {
		h.logger.Error("load users roles failed", "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user roles")
		return
	}
	resp := mapPage(users, func(user model.User) pomeloorbit.UserListResp {
		return userListResponse(user, rolesByUserId[user.Id])
	})
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.UserPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "user:write"); !ok {
		return
	}
	var req pomeloorbit.UserCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.service.Create(r.Context(), usersvc.CreateInput{Username: req.Username, Password: req.Password, Email: req.Email})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp, ok := h.userDetailResp(w, r, user)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "user:read"); !ok {
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	resp, ok := h.userDetailResp(w, r, user)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.RequirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.UserUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Password != nil && *req.Password != "" && !h.canResetPassword(r, current, user.Id) {
		transportresponse.Error(h.logger, w, http.StatusForbidden, "Permission denied")
		return
	}
	if req.Status != nil && current.User.Id == user.Id && strings.TrimSpace(*req.Status) == "disabled" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Cannot disable current user")
		return
	}
	if _, err := h.service.Update(r.Context(), usersvc.UpdateInput{User: user, Username: req.Username, Password: req.Password, Status: req.Status}); err != nil {
		h.writeServiceError(w, err)
		return
	}
	updated, err := h.store.UserById(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load updated user failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user")
		return
	}
	resp, ok := h.userDetailResp(w, r, updated)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateUserRoles(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.RequirePermission(w, r, "user:write")
	if !ok {
		return
	}
	if !authz.HasPermission(current.Permissions, "role:write") {
		transportresponse.Error(h.logger, w, http.StatusForbidden, "Permission denied")
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.UserRoleUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	seen := map[string]struct{}{}
	for _, roleId := range req.RoleIds {
		roleId = strings.TrimSpace(roleId)
		if roleId == "" {
			transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid role id")
			return
		}
		if _, exists := seen[roleId]; exists {
			transportresponse.Error(h.logger, w, http.StatusBadRequest, "role_ids must be unique")
			return
		}
		seen[roleId] = struct{}{}
		if _, err := h.roleStore.RoleById(r.Context(), roleId); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				transportresponse.Error(h.logger, w, http.StatusNotFound, "Role "+roleId+" not found")
				return
			}
			h.logger.Error("load role failed", "role_id", roleId, "error", err)
			transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load role")
			return
		}
	}
	if err := h.store.SetUserRoles(r.Context(), user.Id, req.RoleIds); err != nil {
		h.logger.Error("update user roles failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to update user roles")
		return
	}
	updated, err := h.store.UserById(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load updated user failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user")
		return
	}
	resp, ok := h.userDetailResp(w, r, updated)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) disableUser(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.RequirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.UserDisableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if current.User.Id == user.Id {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Cannot disable current user")
		return
	}
	if user.Status != "disabled" {
		if err := h.service.SetStatus(r.Context(), user.Id, "disabled"); err != nil {
			h.logger.Error("disable user failed", "user_id", user.Id, "error", err)
			transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to disable user")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) enableUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authenticator.RequirePermission(w, r, "user:write"); !ok {
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.UserEnableReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if user.Status != "enabled" {
		if err := h.service.SetStatus(r.Context(), user.Id, "enabled"); err != nil {
			h.logger.Error("enable user failed", "user_id", user.Id, "error", err)
			transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to enable user")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.RequirePermission(w, r, "user:write")
	if !ok {
		return
	}
	user, ok := h.loadUserFromPath(w, r)
	if !ok {
		return
	}
	if current.User.Id == user.Id {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Cannot delete current user")
		return
	}
	if err := h.service.Delete(r.Context(), user.Id); err != nil {
		h.logger.Error("delete user failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindConflict) {
		transportresponse.Error(h.logger, w, http.StatusConflict, err.Error())
		return
	}
	h.logger.Error("user service failed", "error", err)
	transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to create user")
}

func (h Handler) loadUserFromPath(w http.ResponseWriter, r *http.Request) (model.User, bool) {
	userId := strings.TrimSpace(chi.URLParam(r, "user_id"))
	user, err := h.store.UserById(r.Context(), userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			transportresponse.Error(h.logger, w, http.StatusNotFound, "User "+userId+" not found")
			return model.User{}, false
		}
		h.logger.Error("load user failed", "user_id", userId, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user")
		return model.User{}, false
	}
	return user, true
}

func (h Handler) userDetailResp(w http.ResponseWriter, r *http.Request, user model.User) (pomeloorbit.UserResp, bool) {
	roles, err := h.store.UserRoleDetails(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user roles failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user roles")
		return pomeloorbit.UserResp{}, false
	}
	permissions, err := h.store.UserPermissions(r.Context(), user.Id)
	if err != nil {
		h.logger.Error("load user permissions failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load user permissions")
		return pomeloorbit.UserResp{}, false
	}
	roleIds := make([]string, 0, len(roles))
	roleCodes := make([]string, 0, len(roles))
	for _, role := range roles {
		roleIds = append(roleIds, role.Id)
		roleCodes = append(roleCodes, role.Code)
	}
	permissionsByRoleId, err := h.roleStore.RolePermissionCodesByRoleIds(r.Context(), roleIds)
	if err != nil {
		h.logger.Error("load role permissions failed", "user_id", user.Id, "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to load role permissions")
		return pomeloorbit.UserResp{}, false
	}
	return pomeloorbit.UserResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roleCodes, Permissions: permissions, Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(roles, permissionsByRoleId))}, true
}

func userListResponse(user model.User, roles []model.Role) pomeloorbit.UserListResp {
	return pomeloorbit.UserListResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(roles, nil))}
}

func roleResponses(roles []model.Role, permissionsByRoleId map[string][]string) []pomeloorbit.UserRoleResp {
	if roles == nil {
		return []pomeloorbit.UserRoleResp{}
	}
	items := make([]pomeloorbit.UserRoleResp, 0, len(roles))
	for _, role := range roles {
		permissionCodes := []string{}
		if permissionsByRoleId != nil {
			permissionCodes = permissionsByRoleId[role.Id]
			if permissionCodes == nil {
				permissionCodes = []string{}
			}
		}
		items = append(items, pomeloorbit.UserRoleResp{Id: role.Id, Code: role.Code, Name: role.Name, PermissionCodes: permissionCodes})
	}
	return items
}

func (h Handler) canResetPassword(r *http.Request, current authz.CurrentUserContext, targetUserId string) bool {
	if current.User.Id == targetUserId || authz.HasPermission(current.Permissions, "role:write") {
		return true
	}
	targetPermissions, err := h.store.UserPermissions(r.Context(), targetUserId)
	if err != nil {
		h.logger.Warn("load target user permissions failed", "user_id", targetUserId, "error", err)
		return false
	}
	return !authz.HasPermission(targetPermissions, "role:write")
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}
