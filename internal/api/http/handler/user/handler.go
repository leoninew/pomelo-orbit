package userhandler

import (
	"log/slog"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	userv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/user"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logger        *slog.Logger
	service       usersvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service usersvc.Service, authenticator security.Authenticator) Handler {
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
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	items := make([]userv1.UserListResp, 0, len(users.Items))
	for _, user := range users.Items {
		items = append(items, userListResponse(user))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &userv1.UserPaginatedResp{Items: transportresponse.Ptrs(items), Total: int32(users.Total), Page: int32(users.Page), PerPage: int32(users.PerPage), Pages: int32(transportresponse.PageCount(users.Total, users.PerPage))})
}

func (h Handler) CreateUser(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "user:write"); !ok {
		return
	}
	var req userv1.UserCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	user, err := h.service.Create(c.Request.Context(), userCreateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), user.Id)
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := userDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetUser(c *gin.Context) {
	if _, ok := h.authenticator.RequirePermission(c, "user:read"); !ok {
		return
	}
	detail, err := h.service.Detail(c.Request.Context(), userId(c))
	if err != nil {
		transportresponse.WriteError(c, err)
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
	var req userv1.UserUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.UpdateByActor(c.Request.Context(), actor(current), userId(c), userUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
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
	var req userv1.UserRoleUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.SetRoles(c.Request.Context(), actor(current), userId(c), req.RoleIds)
	if err != nil {
		transportresponse.WriteError(c, err)
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
	var req userv1.UserDisableReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.SetStatusByActor(c.Request.Context(), actor(current), userId(c), "disabled"); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) EnableUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	var req userv1.UserEnableReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.SetStatusByActor(c.Request.Context(), actor(current), userId(c), "enabled"); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) DeleteUser(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "user:write")
	if !ok {
		return
	}
	if err := h.service.DeleteByActor(c.Request.Context(), actor(current), userId(c)); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func userId(c *gin.Context) string {
	return strings.TrimSpace(c.Param("user_id"))
}
