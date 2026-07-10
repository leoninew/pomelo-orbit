package userhandler

import (
	"log/slog"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger        *slog.Logger
	service       usersvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service usersvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListUsers(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "user:read"); !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	users, err := h.service.List(c.Request.Context(), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.logger.Error("list users failed", "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to list users")
		return
	}
	items := make([]pomeloorbit.UserListResp, 0, len(users.Items))
	for _, user := range users.Items {
		items = append(items, userListResponse(user))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.UserPaginatedResp{Items: transportresponse.Ptrs(items), Total: int32(users.Total), Page: int32(users.Page), PerPage: int32(users.PerPage), Pages: int32(transportresponse.PageCount(users.Total, users.PerPage))})
}

func (h Handler) CreateUser(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "user:write"); !ok {
		return
	}
	var req pomeloorbit.UserCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.service.Create(c.Request.Context(), userdto.CreateInput{Username: req.Username, Password: req.Password, Email: req.Email})
	if err != nil {
		h.writeServiceError(c, err, "Failed to create user")
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), user.Id)
	if err != nil {
		h.writeServiceError(c, err, "Failed to load user")
		return
	}
	resp := userDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetUser(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "user:read"); !ok {
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), userID(c))
	if err != nil {
		h.writeServiceError(c, err, "Failed to load user")
		return
	}
	resp := userDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	var req pomeloorbit.UserUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.UpdateByActor(c.Request.Context(), actor(current), userID(c), userdto.UpdateInput{Username: req.Username, Password: req.Password, Status: req.Status})
	if err != nil {
		h.writeServiceError(c, err, "Failed to update user")
		return
	}
	resp := userDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateUserRoles(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	var req pomeloorbit.UserRoleUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.SetRoles(c.Request.Context(), actor(current), userID(c), req.RoleIds)
	if err != nil {
		h.writeServiceError(c, err, "Failed to update user roles")
		return
	}
	resp := userDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DisableUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	var req pomeloorbit.UserDisableReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.SetStatusByActor(c.Request.Context(), actor(current), userID(c), "disabled"); err != nil {
		h.writeServiceError(c, err, "Failed to disable user")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) EnableUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	var req pomeloorbit.UserEnableReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.SetStatusByActor(c.Request.Context(), actor(current), userID(c), "enabled"); err != nil {
		h.writeServiceError(c, err, "Failed to enable user")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) DeleteUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	if err := h.service.DeleteByActor(c.Request.Context(), actor(current), userID(c)); err != nil {
		h.writeServiceError(c, err, "Failed to delete user")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) writeServiceError(c *gin.Context, err error, internalDetail string) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindConflict) {
		transportresponse.Error(c, http.StatusConflict, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindForbidden) {
		transportresponse.Error(c, http.StatusForbidden, err.Error())
		return
	}
	if apperror.IsKind(err, apperror.KindNotFound) {
		transportresponse.Error(c, http.StatusNotFound, err.Error())
		return
	}
	h.logger.Error("user service failed", "error", err)
	transportresponse.Error(c, http.StatusInternalServerError, internalDetail)
}

func actor(current authz.CurrentUserContext) userdto.Actor {
	return userdto.Actor{UserId: current.User.Id, Permissions: current.Permissions}
}

func userID(c *gin.Context) string {
	return strings.TrimSpace(c.Param("user_id"))
}

func userDetailResponse(detail userdto.Detail) pomeloorbit.UserResp {
	user := detail.User
	return pomeloorbit.UserResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Roles: roleCodes(detail.Roles), Permissions: emptyStrings(detail.Permissions), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(detail.Roles, detail.PermissionsByRoleId))}
}

func userListResponse(item userdto.ListItem) pomeloorbit.UserListResp {
	user := item.User
	return pomeloorbit.UserListResp{Id: user.Id, Username: user.Username, Email: user.Email, AuthSource: user.AuthSource, CreatedAt: transportresponse.FormatTime(user.CreatedAt), LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt), Status: user.Status, UpdatedAt: transportresponse.FormatTime(user.UpdatedAt), RoleItems: transportresponse.Ptrs(roleResponses(item.Roles, nil))}
}

func roleCodes(roles []model.Role) []string {
	codes := make([]string, 0, len(roles))
	for _, role := range roles {
		codes = append(codes, role.Code)
	}
	return codes
}

func roleResponses(roles []model.Role, permissionsByRoleID map[string][]string) []pomeloorbit.UserRoleResp {
	items := make([]pomeloorbit.UserRoleResp, 0, len(roles))
	for _, role := range roles {
		permissionCodes := []string{}
		if permissionsByRoleID != nil && permissionsByRoleID[role.Id] != nil {
			permissionCodes = permissionsByRoleID[role.Id]
		}
		items = append(items, pomeloorbit.UserRoleResp{Id: role.Id, Code: role.Code, Name: role.Name, PermissionCodes: permissionCodes})
	}
	return items
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
